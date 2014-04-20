// Copyright (c) 2014 Square, Inc

package pidstat

import (
	//"fmt"
	"bufio"
	"os"
	// "math/rand"
	//"regexp"
	"time"
	"syscall"
	"strings"
	"io/ioutil"
	"math"
	"sort"
	"path"
	"path/filepath"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
)

/*
#include <unistd.h>
#include <sys/types.h>
*/
import "C"

type ProcessStat struct {
	Mountpoint   string
	m            *metrics.MetricContext
	Processes    map[string]*PerProcessStat
	collectAttributes      bool
}

// NewProcessStat allocates a new ProcessStat object
// Arguments:
// m - *metricContext

// Collects metrics every m.Step seconds
// Drops refresh interval by m.Step for every additional
// 1024 processes
// TODO: Implement better heuristics to manage load
//   * Collect metrics for newer processes at faster rate
//   * Slower rate for processes with neglible rate?

func NewProcessStat(m *metrics.MetricContext) *ProcessStat {
	c := new(ProcessStat)
	c.m = m

	c.Processes = make(map[string]*PerProcessStat,4096)

	var n int
	ticker := time.NewTicker(m.Step)
	go func() {
		for _ = range ticker.C {
			p := int(len(c.Processes) / 1024)
			// always collect all metrics for first two samples
			// and if number of processes < 1024
			if n < 2 {
				c.collectAttributes = true
				c.Collect()
			} else if p < 1 || n % p == 0 {
				c.collectAttributes = false
				c.Collect()
			}
			n++
		}
	}()

	return c
}


// ByCPUUsage implements sort.Interface for []*PerProcessStat based on
// the Usage() method

type ByCPUUsage []*PerProcessStat

func (a ByCPUUsage) Len() int             { return len(a) }
func (a ByCPUUsage) Swap(i,j int)         { a[i],a[j] = a[j],a[i] }
func (a ByCPUUsage) Less(i,j int)  bool   { return a[i].CPUUsage() > a[j].CPUUsage() }

func (c *ProcessStat) ByCPUUsage() []*PerProcessStat {
	v := make([]*PerProcessStat,0)
	for _,o := range c.Processes {
		if ! math.IsNaN(o.CPUUsage()) {
			v = append(v,o)
		}
	}
	sort.Sort(ByCPUUsage(v))
	return v
}

type ByMemUsage []*PerProcessStat

func (a ByMemUsage) Len() int             { return len(a) }
func (a ByMemUsage) Swap(i,j int)         { a[i],a[j] = a[j],a[i] }
func (a ByMemUsage) Less(i,j int)  bool   { return a[i].MemUsage() > a[j].MemUsage() }


func (c *ProcessStat) ByMemUsage() []*PerProcessStat {
	v := make([]*PerProcessStat,0)
	for _,o := range c.Processes {
		if ! math.IsNaN(o.MemUsage()) {
			v = append(v,o)
		}
	}
	sort.Sort(ByMemUsage(v))
	return v
}

func (c *ProcessStat) ByCgroup() {



}


// Collect walks through /proc and updates stats
// Collect is usually called internally based on
// parameters passed via metric context
// XXX: restructure to avoid right indent

func (c *ProcessStat) Collect() {
	h := c.Processes
	for _,v := range h {
		v.Metrics.dead = true
	}

	_ = filepath.Walk(
		"/proc",
		func(p string, f os.FileInfo, e error) error {
			if e != nil {
				panic(e)
			}
			if f.IsDir() && p != "/proc" {
				p := path.Base(p)
				pidstat,ok := h[p]
				if !ok {
					pidstat = NewPerProcessStat(c.m,path.Base(p))
					h[p] = pidstat
				}
				if c.collectAttributes {
					st := f.Sys()
					if st != nil {
						pidstat.Metrics.Uid = st.(*syscall.Stat_t).Uid
						pidstat.Metrics.Gid = st.(*syscall.Stat_t).Gid
					}
					pidstat.Metrics.CollectAttributes()
				}
				pidstat.Metrics.Collect()
				pidstat.Metrics.dead = false
				return filepath.SkipDir
			}
			return nil
		})

	// remove dead processes
	for k,v := range h {
		if v.Metrics.dead {
			delete(h,k)
		}
	}
}

// Per Process functions
type PerProcessStat struct {
	Metrics *PerProcessStatMetrics
	m       *metrics.MetricContext
	pagesize int64
}

func NewPerProcessStat(m *metrics.MetricContext, p string) *PerProcessStat {
	c := new(PerProcessStat)
	c.m = m
	c.Metrics = NewPerProcessStatMetrics(m, p)

	c.pagesize = int64(C.sysconf(C._SC_PAGESIZE))

	return c
}

func (s *PerProcessStat) CPUUsage() float64 {
	o := s.Metrics
	t := (o.UpdatedAt.CurRate() / float64(time.Second.Nanoseconds()))
	return (o.Utime.CurRate() + o.Stime.CurRate()) / t
}

func (s *PerProcessStat) MemUsage() float64 {
	o := s.Metrics
	return o.Rss.V * float64(s.pagesize)
}



type PerProcessStatMetrics struct {
	Pid           string
	Comm          string
	Cmdline       string
	Cgroup        *map[string]string
	Uid           uint32
	Gid           uint32
	Majflt	      *metrics.Counter
	Utime	      *metrics.Counter
	Stime         *metrics.Counter
	UpdatedAt     *metrics.Counter
	StartedAt    int64
	Rss           *metrics.Gauge
	pid           string
	dead          bool
}

func NewPerProcessStatMetrics(m *metrics.MetricContext, pid string) *PerProcessStatMetrics {
	s := new(PerProcessStatMetrics)
	s.pid = pid

	s.StartedAt = time.Now().UnixNano()

	// initialize all metrics
	misc.InitializeMetrics(s, m)

	return s
}

func (s *PerProcessStatMetrics) Collect() {
	file, err := os.Open("/proc/" + s.pid + "/stat")
	defer file.Close()

	if err != nil {
		return
	}

	now := time.Now().UnixNano()
	if now - s.StartedAt < 0 {
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		f := strings.Split(scanner.Text()," ")
		s.Pid  =  f[0]
		s.Comm = f[1]
		s.Majflt.Set(misc.ParseUint(f[11]))
		s.Utime.Set(misc.ParseUint(f[13]))
		s.Stime.Set(misc.ParseUint(f[14]))
		s.Rss.Set(float64(misc.ParseUint(f[23])))
	}

	s.UpdatedAt.Set(uint64(time.Now().UnixNano() - s.StartedAt))
}

func (s *PerProcessStatMetrics) CollectAttributes() {
	content, err := ioutil.ReadFile("/proc/" + s.pid + "/cmdline")
	if err == nil {
		s.Cmdline = string(content)
	}

	file,err := os.Open("/proc/" + s.pid + "/cgroup")
	defer file.Close()

	if err == nil {
		t := make(map[string]string)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			f := strings.Split(scanner.Text(),":")
			t[f[1]] = f[2]
		}
		s.Cgroup = &t
	}
}
