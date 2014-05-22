// Copyright (c) 2014 Square, Inc

package cpustat

import (
	"bufio"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type CgroupStat struct {
	Cgroups    map[string]*PerCgroupStat
	m          *metrics.MetricContext
	Mountpoint string
}

func NewCgroupStat(m *metrics.MetricContext, Step time.Duration) *CgroupStat {
	c := new(CgroupStat)
	c.m = m

	c.Cgroups = make(map[string]*PerCgroupStat, 1)

	mountpoint, err := misc.FindCgroupMount("cpu")
	if err != nil {
		return c
	}
	c.Mountpoint = mountpoint

	ticker := time.NewTicker(Step)
	go func() {
		for _ = range ticker.C {
			c.Collect(mountpoint)
		}
	}()

	return c
}

func (c *CgroupStat) Collect(mountpoint string) {

	cgroups, err := misc.FindCgroups(mountpoint)
	if err != nil {
		return
	}

	// stop tracking cgroups which don't exist
	// anymore or have no tasks
	cgroupsMap := make(map[string]bool, len(cgroups))
	for _, cgroup := range cgroups {
		cgroupsMap[cgroup] = true
	}

	for cgroup, _ := range c.Cgroups {
		_, ok := cgroupsMap[cgroup]
		if !ok {
			delete(c.Cgroups, cgroup)
		}
	}

	for _, cgroup := range cgroups {
		_, ok := c.Cgroups[cgroup]
		if !ok {
			c.Cgroups[cgroup] = NewPerCgroupStat(c.m, cgroup, mountpoint)
		}
		c.Cgroups[cgroup].Metrics.Collect()
	}
}

// Per Cgroup functions

type PerCgroupStat struct {
	Metrics *PerCgroupStatMetrics
	m       *metrics.MetricContext
}

func NewPerCgroupStat(m *metrics.MetricContext, path string, mp string) *PerCgroupStat {
	c := new(PerCgroupStat)
	c.m = m

	c.Metrics = NewPerCgroupStatMetrics(m, path, mp)

	return c
}

// Throttle returns as percentage of time that
// the cgroup couldn't get enough cpu
// rate ((nr_throttled * period) / quota)
// XXX: add support for real-time scheduler stats

func (s *PerCgroupStat) Throttle() float64 {
	o := s.Metrics
	throttled_sec := o.Throttled_time.ComputeRate()

	return (throttled_sec / (1 * 1000 * 1000 * 1000)) * 100
}

// Quota returns how many logical CPUs can be used

func (s *PerCgroupStat) Quota() float64 {
	o := s.Metrics
	return (o.Cfs_quota_us.Get() / o.Cfs_period_us.Get())
}

type PerCgroupStatMetrics struct {
	Nr_periods     *metrics.Counter
	Nr_throttled   *metrics.Counter
	Throttled_time *metrics.Counter
	Cfs_period_us  *metrics.Gauge
	Cfs_quota_us   *metrics.Gauge
	path           string
}

func NewPerCgroupStatMetrics(m *metrics.MetricContext, path string, mp string) *PerCgroupStatMetrics {
	c := new(PerCgroupStatMetrics)
	c.path = path

	// initialize all metrics and register them
	prefix, _ := filepath.Rel(mp, path)
	misc.InitializeMetrics(c, m, "cpustat.cgroup."+prefix, true)

	return c
}

func (s *PerCgroupStatMetrics) Collect() {
	file, err := os.Open(s.path + "/" + "cpu.stat")
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		f := regexp.MustCompile("\\s+").Split(scanner.Text(), 2)

		if f[0] == "nr_periods" {
			s.Nr_periods.Set(misc.ParseUint(f[1]))
		}

		if f[0] == "nr_throttled" {
			s.Nr_throttled.Set(misc.ParseUint(f[1]))
		}

		if f[0] == "throttled_time" {
			s.Throttled_time.Set(misc.ParseUint(f[1]))
		}
	}

	s.Cfs_period_us.Set(
		float64(misc.ReadUintFromFile(
			s.path + "/" + "cpu.cfs_period_us")))

	s.Cfs_quota_us.Set(
		float64(misc.ReadUintFromFile(
			s.path + "/" + "cpu.cfs_quota_us")))
}
