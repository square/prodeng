// Copyright (c) 2014 Square, Inc

package cpustat

import (
	"fmt"
	"bufio"
	"os"
	"regexp"
	"time"
	"syscall"
	"strings"
	"io/ioutil"
	"path"
	"path/filepath"
	"container/heap"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
)

// a heap to store top-N processes stats
type ProcessesHeap []*PerProcessStat

func (h ProcessesHeap) Len() int          { return len(h) }
func (h ProcessesHeap) Less(i,j int) bool { return h[i].Usage() < h[j].Usage() }
func (h ProcessesHeap) Swap(i,j int)      { h[i], h[j] = h[j], h[i] }

func (h *ProcessesHeap) Push(x interface{}) {
	*h = append(*h, x.(*PerProcessStat))
}

func (h *ProcessesHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (h *ProcessesHeap) Print() {
	for _,x := range *h {
		fmt.Printf("Top process: %s(%f) \n", x.Metrics.Cmdline,x.Usage())
	}
}


type ProcessStat struct {
	Mountpoint   string
	m            *metrics.MetricContext
	Processes    *ProcessesHeap
	n            int
}

// NewProcessStat allocates a new ProcessStat object
// Arguments:
// m - *metricContext
// n - int (number of process to track)

func NewProcessStat(m *metrics.MetricContext,n int) *ProcessStat {
	c := new(ProcessStat)
	c.m = m
	c.n = n

	c.Processes = new(ProcessesHeap)
	heap.Init(c.Processes)


	ticker := time.NewTicker(m.Step)
	go func() {
		for _ = range ticker.C {
		 c.Processes = new(ProcessesHeap)
	         heap.Init(c.Processes)
			c.Collect()
			c.Processes.Print()
		}
	}()

	return c
}

// Collect walks through /proc and updates stats
// Collect is usually called internally based on
// parameters passed via metric context
// XXX: restructure to avoid right indent

func (c *ProcessStat) Collect() {
	h := c.Processes
	_ = filepath.Walk(
		"/proc",
		func(p string, f os.FileInfo, _ error) error {
			if f.IsDir() && p != "/proc" {
				//m := metrics.NewMetricContext("tmp",time.Millisecond*5,2)
				pidstat := NewPerProcessStat(c.m,path.Base(p))
				pidstat.Metrics.Collect()
				// lets sleep 1ms and collect stats and check 
				// if they are worth tracking
				// for now we only track top-N processes by
				// CPU used (we use a heap)
				//time.Sleep(time.Millisecond * 5)
				pidstat.Metrics.Collect()
				heap.Push(h,pidstat)

/*
				if h.Len() < c.n {
					heap.Push(h,pidstat)
				} else {
					x := heap.Pop(h).(*PerProcessStat)
					if pidstat.Usage()  > x.Usage() {
						heap.Push(h,pidstat)
					} else {
						heap.Push(h,x)
					}
				} */
				return filepath.SkipDir
			}
			return nil
		})
}

// Per Process functions
type PerProcessStat struct {
	Metrics *PerProcessStatMetrics
	m       *metrics.MetricContext
}

func NewPerProcessStat(m *metrics.MetricContext, p string) *PerProcessStat {
	c := new(PerProcessStat)
	c.m = m
	c.Metrics = NewPerProcessStatMetrics(m, p)

	return c
}

func (s *PerProcessStat) Usage() float64 {
	o := s.Metrics
	return (o.Utime.CurRate() + o.Stime.CurRate())
}


type PerProcessStatMetrics struct {
	Pid           uint64
	Comm          string
	Cmdline       string
	Cgroup        *map[string]string
	Uid           uint32
	Gid           uint32
	Majflt	      *metrics.Counter
	Utime	      *metrics.Counter
	Stime         *metrics.Counter
	Rss           *metrics.Gauge
	pid           string
}

func NewPerProcessStatMetrics(m *metrics.MetricContext, pid string) *PerProcessStatMetrics {
	s := new(PerProcessStatMetrics)
	s.pid = pid

	// initialize all metrics
	misc.InitializeMetrics(s, m)

	return s
}

func (s *PerProcessStatMetrics) Collect() {
	file, err := os.Open("/proc/" + s.pid + "/stat")
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		f := regexp.MustCompile("\\s+").Split(scanner.Text(), -1)
		s.Pid  =  misc.ParseUint(f[0])
		s.Comm = f[1]
		s.Majflt.Set(misc.ParseUint(f[11]))
		s.Utime.Set(misc.ParseUint(f[13]))
		s.Stime.Set(misc.ParseUint(f[14]))
		s.Rss.Set(float64(misc.ParseUint(f[23])))

	}

	content, err := ioutil.ReadFile("/proc/" + s.pid + "/cmdline")
	if err == nil {
		s.Cmdline = string(content)
	}

	file,err = os.Open("/proc/" + s.pid + "/cgroup")
	if err == nil {
		t := make(map[string]string)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			f := strings.Split(scanner.Text(),":")
			t[f[1]] = f[2]
		}
		s.Cgroup = &t
	}

	fd, err := syscall.Open("/proc/" + s.pid,0,0)
	if err == nil {
		stat := new(syscall.Stat_t)
		err := syscall.Fstat(fd,stat)
		if err == nil {
			s.Uid = stat.Uid
			s.Gid = stat.Gid
		}
	}
}
