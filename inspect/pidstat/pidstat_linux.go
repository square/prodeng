// Copyright (c) 2014 Square, Inc

package pidstat

import (
	"bufio"
	"os"
	// "math/rand"
	//"regexp"
	"fmt"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
	"io/ioutil"
	"math"
	"os/user"
	"path"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"
)

/*
#include <unistd.h>
#include <sys/types.h>
*/
import "C"

var LINUX_TICKS_IN_SEC int = int(C.sysconf(C._SC_CLK_TCK))

type ProcessStat struct {
	Mountpoint string
	m          *metrics.MetricContext
	Processes  map[string]*PerProcessStat
}

// NewProcessStat allocates a new ProcessStat object
// Arguments:
// m - *metricContext

// Collects metrics every Step seconds
// Drops refresh interval by Step for every additional
// 1024 processes
// TODO: Implement better heuristics to manage load
//   * Collect metrics for newer processes at faster rate
//   * Slower rate for processes with neglible rate?

func NewProcessStat(m *metrics.MetricContext, Step time.Duration) *ProcessStat {
	c := new(ProcessStat)
	c.m = m

	c.Processes = make(map[string]*PerProcessStat, 1024)

	var n int
	ticker := time.NewTicker(Step)
	go func() {
		for _ = range ticker.C {
			p := int(len(c.Processes) / 1024)
			if n == 0 {
				c.Collect(true)
			}
			// always collect all metrics for first two samples
			// and if number of processes < 1024
			if p < 1 || n%p == 0 {
				c.Collect(false)
			}
			n++
		}
	}()

	return c
}

// ByCPUUsage implements sort.Interface for []*PerProcessStat based on
// the Usage() method

type ByCPUUsage []*PerProcessStat

func (a ByCPUUsage) Len() int           { return len(a) }
func (a ByCPUUsage) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByCPUUsage) Less(i, j int) bool { return a[i].CPUUsage() > a[j].CPUUsage() }

// ByCPUUsage() returns an slice of *PerProcessStat entries sorted
// by CPU usage
func (c *ProcessStat) ByCPUUsage() []*PerProcessStat {
	v := make([]*PerProcessStat, 0)
	for _, o := range c.Processes {
		if !math.IsNaN(o.CPUUsage()) {
			v = append(v, o)
		}
	}
	sort.Sort(ByCPUUsage(v))
	return v
}

type ByMemUsage []*PerProcessStat

func (a ByMemUsage) Len() int           { return len(a) }
func (a ByMemUsage) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByMemUsage) Less(i, j int) bool { return a[i].MemUsage() > a[j].MemUsage() }

// ByMemUsage() returns an slice of *PerProcessStat entries sorted
// by Memory usage
func (c *ProcessStat) ByMemUsage() []*PerProcessStat {
	v := make([]*PerProcessStat, 0)
	for _, o := range c.Processes {
		if !math.IsNaN(o.MemUsage()) {
			v = append(v, o)
		}
	}
	sort.Sort(ByMemUsage(v))
	return v
}

// CPUUsagePerCgroup returns cumulative CPU usage by cgroup
func (c *ProcessStat) CPUUsagePerCgroup(cgroup string) float64 {
	var ret float64
	if !path.IsAbs(cgroup) {
		cgroup = "/" + cgroup
	}

	for _, o := range c.Processes {
		if (o.Metrics.Cgroup["cpu"] == cgroup) && !math.IsNaN(o.CPUUsage()) {
			ret += o.CPUUsage()
		}
	}
	return ret
}

// MemUsagePerCgroup returns cumulative Memory usage by cgroup
func (c *ProcessStat) MemUsagePerCgroup(cgroup string) float64 {
	var ret float64
	if !path.IsAbs(cgroup) {
		cgroup = "/" + cgroup
	}
	for _, o := range c.Processes {
		if (o.Metrics.Cgroup["memory"] == cgroup) && !math.IsNaN(o.MemUsage()) {
			ret += o.MemUsage()
		}
	}
	return ret
}

// Collect walks through /proc and updates stats
// Collect is usually called internally based on
// parameters passed via metric context
// Takes a single boolean parameter which specifies
// if we should collect/refresh all process attributes

func (c *ProcessStat) Collect(collectAttributes bool) {
	h := c.Processes
	for _, v := range h {
		v.Metrics.dead = true
	}

	pids, err := ioutil.ReadDir("/proc")
	if err != nil {
		return
	}

	pidre := regexp.MustCompile("^\\d+")

	for _, f := range pids {
		p := f.Name()
		st := f.Sys()
		if f.IsDir() && pidre.MatchString(p) {
			pidstat, ok := h[p]
			if !ok {
				pidstat = NewPerProcessStat(c.m, path.Base(p))
				h[p] = pidstat
			}
			pidstat.Metrics.Collect()
			pidstat.Metrics.dead = false

			// collect other process attributes like uid,gid,cgroup
			// etc only for new processes or when run for the first
			// time
			if collectAttributes || !ok && st != nil {
				pidstat.Metrics.populateId(st)
				pidstat.Metrics.CollectAttributes()
			}
		}
	}

	// remove dead processes
	for k, v := range h {
		if v.Metrics.dead {
			delete(h, k)
		}
	}
}

// Per Process functions
type PerProcessStat struct {
	Metrics  *PerProcessStatMetrics
	m        *metrics.MetricContext
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
	rate_per_sec := (o.Utime.ComputeRate() + o.Stime.ComputeRate())
	pct_use := (rate_per_sec * 100) / float64(LINUX_TICKS_IN_SEC)
	return pct_use
}

func (s *PerProcessStat) MemUsage() float64 {
	o := s.Metrics
	return o.Rss.Get() * float64(s.pagesize)
}

func (s *PerProcessStat) Pid() string {
	return s.Metrics.Pid
}

func (s *PerProcessStat) Comm() string {
	return s.Metrics.Comm
}

func (s *PerProcessStat) User() string {
	return s.Metrics.User
}

type PerProcessStatMetrics struct {
	Pid       string
	Comm      string
	Cmdline   string
	Cgroup    map[string]string
	Uid       uint32
	Gid       uint32
	User      string
	Majflt    *metrics.Counter
	Utime     *metrics.Counter
	Stime     *metrics.Counter
	UpdatedAt *metrics.Counter
	StartedAt int64
	Rss       *metrics.Gauge
	pid       string
	dead      bool
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
	if now-s.StartedAt < 0 {
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		f := strings.Split(scanner.Text(), " ")
		s.Pid = f[0]
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

	file, err := os.Open("/proc/" + s.pid + "/cgroup")
	defer file.Close()

	if err == nil {
		t := make(map[string]string)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			f := strings.Split(scanner.Text(), ":")
			t[f[1]] = f[2]
		}
		s.Cgroup = t
	}
}

// unexported
func (s *PerProcessStatMetrics) populateId(st interface{}) {
	s.Uid = st.(*syscall.Stat_t).Uid
	s.Gid = st.(*syscall.Stat_t).Gid
	u, err := user.LookupId(fmt.Sprintf("%v", s.Uid))
	if err == nil {
		s.User = u.Username
	}
}
