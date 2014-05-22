// Copyright (c) 2014 Square, Inc

package cpustat

import (
	"bufio"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
	"math"
	"os"
	"regexp"
	"time"
)

type CPUStat struct {
	All           *CPUStatPerCPU
	Procs_running *metrics.Counter
	Procs_blocked *metrics.Counter
	cpus          map[string]*CPUStatPerCPU
	m             *metrics.MetricContext
}

type CPUStatPerCPU struct {
	User        *metrics.Counter
	UserLowPrio *metrics.Counter
	System      *metrics.Counter
	Idle        *metrics.Counter
	Iowait      *metrics.Counter
	Irq         *metrics.Counter
	Softirq     *metrics.Counter
	Steal       *metrics.Counter
	Guest       *metrics.Counter
	Total       *metrics.Counter // total jiffies
}

func New(m *metrics.MetricContext, Step time.Duration) *CPUStat {
	c := new(CPUStat)
	c.All = NewCPUStatPerCPU(m, "cpu")
	c.m = m
	c.cpus = make(map[string]*CPUStatPerCPU, 1)
	ticker := time.NewTicker(Step)
	go func() {
		for _ = range ticker.C {
			c.Collect()
		}
	}()
	return c
}

// XXX: break this up into two smaller functions
func (s *CPUStat) Collect() {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		f := regexp.MustCompile("\\s+").Split(scanner.Text(), -1)

		is_cpu, err := regexp.MatchString("^cpu\\d*", f[0])
		if err == nil && is_cpu {
			if f[0] == "cpu" {
				parseCPUline(s.All, f)
			} else {
				per_cpu, ok := s.cpus[f[0]]
				if !ok {
					per_cpu = NewCPUStatPerCPU(s.m, f[0])
					s.cpus[f[0]] = per_cpu
				}
				parseCPUline(per_cpu, f)
			}
		}
	}
}

// Usage returns current total CPU usage in percentage across all CPUs
func (o *CPUStat) Usage() float64 {
	return o.All.Usage()
}

// UserSpace returns time spent in userspace as percentage across all
// CPUs
func (o *CPUStat) UserSpace() float64 {
	return o.All.UserSpace()
}

// Kernel returns time spent in userspace as percentage across all
// CPUs
func (o *CPUStat) Kernel() float64 {
	return o.All.Kernel()
}

// CPUS returns all CPUS found as a slice of strings
func (o *CPUStat) CPUS() []string {
	ret := make([]string, 1)
	for k, _ := range o.cpus {
		ret = append(ret, k)
	}

	return ret
}

// PerCPUStat returns per-CPU stats for argument "cpu"
func (o *CPUStat) PerCPUStat(cpu string) *CPUStatPerCPU {
	return o.cpus[cpu]
}

// NewCPUStatPerCPU returns a struct representing counters for
// per CPU statistics
func NewCPUStatPerCPU(m *metrics.MetricContext, name string) *CPUStatPerCPU {
	o := new(CPUStatPerCPU)

	// initialize all metrics and register them
	misc.InitializeMetrics(o, m, "cpustat."+name, true)
	return o
}

// Usage returns total percentage of CPU used
func (o *CPUStatPerCPU) Usage() float64 {
	u := o.User.ComputeRate()
	n := o.UserLowPrio.ComputeRate()
	s := o.System.ComputeRate()
	t := o.Total.ComputeRate()

	if u != math.NaN() && n != math.NaN() && s != math.NaN() &&
		t != math.NaN() && t > 0 {
		return (u + s + n) / t * 100
	} else {
		return math.NaN()
	}
}

// UserSpace returns percentage of time spent in userspace
// on this CPU
func (o *CPUStatPerCPU) UserSpace() float64 {
	u := o.User.ComputeRate()
	n := o.UserLowPrio.ComputeRate()
	t := o.Total.ComputeRate()
	if u != math.NaN() && t != math.NaN() && n != math.NaN() && t > 0 {
		return (u + n) / t * 100
	}
	return math.NaN()
}

// Kernel returns percentage of time spent in kernel
// on this CPU
func (o *CPUStatPerCPU) Kernel() float64 {
	s := o.System.ComputeRate()
	t := o.Total.ComputeRate()
	if s != math.NaN() && t != math.NaN() && t > 0 {
		return (s / t) * 100
	}
	return math.NaN()
}

// Unexported functions
func parseCPUline(s *CPUStatPerCPU, f []string) {
	s.User.Set(misc.ParseUint(f[1]))
	s.UserLowPrio.Set(misc.ParseUint(f[2]))
	s.System.Set(misc.ParseUint(f[3]))
	s.Idle.Set(misc.ParseUint(f[4]))
	s.Iowait.Set(misc.ParseUint(f[5]))
	s.Irq.Set(misc.ParseUint(f[6]))
	s.Softirq.Set(misc.ParseUint(f[7]))
	s.Steal.Set(misc.ParseUint(f[8]))
	s.Guest.Set(misc.ParseUint(f[9]))
	s.Total.Set(s.User.Get() + s.UserLowPrio.Get() + s.System.Get() + s.Idle.Get())
}
