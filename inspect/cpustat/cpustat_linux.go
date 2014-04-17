// Copyright (c) 2014 Square, Inc
package cpustat

import (
	"bufio"
	"math"
	"os"
	"regexp"
	"time"
	"github.com/square/prodeng/metrics"
	"github.com/square/prodeng/inspect/misc"
)

type CPUStat struct {
	All           *CPUStatPerCPU
	Count         *metrics.Counter
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

func New(m *metrics.MetricContext) *CPUStat {
	c := new(CPUStat)
	c.All = CPUStatPerCPUNew(m)
	c.m = m
	c.cpus = make(map[string]*CPUStatPerCPU, 1)
	ticker := time.NewTicker(m.Step)
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
					per_cpu = CPUStatPerCPUNew(s.m)
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
	return []string{"cpu0"}
}

// PerCPUStat returns per-CPU stats for argument "cpu"
func (o *CPUStat) PerCPUStat(cpu string) *CPUStatPerCPU {
	return o.cpus[cpu]
}

// CPUStatPerCPUNew returns a struct representing counters for
// per CPU statistics
func CPUStatPerCPUNew(m *metrics.MetricContext) *CPUStatPerCPU {
	o := new(CPUStatPerCPU)
	o.User = m.NewCounter("User")
	o.UserLowPrio = m.NewCounter("UserLowPrio")
	o.System = m.NewCounter("System")
	o.Idle = m.NewCounter("Idle")
	o.Iowait = m.NewCounter("Iowait")
	o.Irq = m.NewCounter("Irq")
	o.Softirq = m.NewCounter("Softirq")
	o.Steal = m.NewCounter("Steal")
	o.Guest = m.NewCounter("Guest")
	o.Total = m.NewCounter("Total")
	return o
}

// Usage returns total percentage of CPU used
func (o *CPUStatPerCPU) Usage() float64 {
	u := o.User.CurRate()
	n := o.UserLowPrio.CurRate()
	s := o.System.CurRate()
	t := o.Total.CurRate()

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
	u := o.User.CurRate()
	n := o.UserLowPrio.CurRate()
	t := o.Total.CurRate()
	if u != math.NaN() && t != math.NaN() && n != math.NaN() && t > 0 {
		return (u + n) / t * 100
	}
	return math.NaN()
}

// Kernel returns percentage of time spent in kernel
// on this CPU
func (o *CPUStatPerCPU) Kernel() float64 {
	s := o.System.CurRate()
	t := o.Total.CurRate()
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
	s.Total.Set(s.User.V + s.UserLowPrio.V + s.System.V + s.Idle.V)
}
