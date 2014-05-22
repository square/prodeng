// Copyright (c) 2014 Square, Inc

package memstat

import (
	"bufio"
	"fmt"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
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

	mountpoint, err := misc.FindCgroupMount("memory")
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

// Free returns free physical memory including cache
// Use soft_limit_in_bytes as upper bound or if not
// set use system memory
// NOT IMPLEMENTED YET
// rename to Free() when done
func (s *PerCgroupStat) free() float64 {
	return 0
}

// Usage returns physical memory in use; not including buffers/cached/sreclaimable
func (s *PerCgroupStat) Usage() float64 {
	o := s.Metrics
	return o.Rss.Get() + o.Mapped_file.Get()
}

// SoftLimit returns soft-limit for the cgroup
func (s *PerCgroupStat) SoftLimit() float64 {
	o := s.Metrics
	return o.Soft_Limit_In_Bytes.Get()
}

type PerCgroupStatMetrics struct {
	// memory.stat
	Cache                     *metrics.Gauge
	Rss                       *metrics.Gauge
	Mapped_file               *metrics.Gauge
	Pgpgin                    *metrics.Gauge
	Pgpgout                   *metrics.Gauge
	Swap                      *metrics.Gauge
	Active_anon               *metrics.Gauge
	Inactive_anon             *metrics.Gauge
	Active_file               *metrics.Gauge
	Inactive_file             *metrics.Gauge
	Unevictable               *metrics.Gauge
	Hierarchical_memory_limit *metrics.Gauge
	Hierarchical_memsw_limit  *metrics.Gauge
	Total_cache               *metrics.Gauge
	Total_rss                 *metrics.Gauge
	Total_mapped_file         *metrics.Gauge
	Total_pgpgin              *metrics.Gauge
	Total_pgpgout             *metrics.Gauge
	Total_swap                *metrics.Gauge
	Total_inactive_anon       *metrics.Gauge
	Total_active_anon         *metrics.Gauge
	Total_inactive_file       *metrics.Gauge
	Total_active_file         *metrics.Gauge
	Total_unevictable         *metrics.Gauge
	// memory.soft_limit_in_bytes
	Soft_Limit_In_Bytes *metrics.Gauge
	path                string
}

func NewPerCgroupStatMetrics(m *metrics.MetricContext,
	path string, mp string) *PerCgroupStatMetrics {

	c := new(PerCgroupStatMetrics)
	c.path = path

	prefix, _ := filepath.Rel(mp, path)
	// initialize all metrics and register them
	misc.InitializeMetrics(c, m, "memstat.cgroup."+prefix, true)

	return c
}

func (s *PerCgroupStatMetrics) Collect() {
	file, err := os.Open(s.path + "/" + "memory.stat")
	if err != nil {
		fmt.Println(err)
		return
	}

	d := map[string]*metrics.Gauge{}
	// Get all fields we care about
	r := reflect.ValueOf(s).Elem()
	typeOfT := r.Type()
	for i := 0; i < r.NumField(); i++ {
		f := r.Field(i)
		if f.Kind().String() == "ptr" && f.Type().Elem() == reflect.TypeOf(metrics.Gauge{}) {
			d[strings.ToLower(typeOfT.Field(i).Name)] =
				f.Interface().(*metrics.Gauge)
		}
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		f := regexp.MustCompile("[\\s]+").Split(scanner.Text(), 2)
		g, ok := d[strings.ToLower(f[0])]
		if ok {
			parseCgroupMemLine(g, f)
		}
	}

	s.Soft_Limit_In_Bytes.Set(
		float64(misc.ReadUintFromFile(
			s.path + "/" + "memory.soft_limit_in_bytes")))
}

// Unexported functions
func parseCgroupMemLine(g *metrics.Gauge, f []string) {
	length := len(f)
	val := math.NaN()

	if length < 2 {
		goto fail
	}

	val = float64(misc.ParseUint(f[1]))
	g.Set(val)
	return

fail:
	g.Set(math.NaN())
	return
}
