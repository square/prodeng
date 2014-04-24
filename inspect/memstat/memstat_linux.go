// Copyright (c) 2014 Square, Inc

package memstat

import (
	"bufio"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
	"math"
	"os"
	"reflect"
	"regexp"
	"time"
)

type MemStat struct {
	Metrics       *MemStatMetrics
	m             *metrics.MetricContext
	Cgroups       map[string]*CgroupStat
	EnableCgroups bool
}

func New(m *metrics.MetricContext) *MemStat {
	s := new(MemStat)
	s.Metrics = MemStatMetricsNew(m)
	return s
}

// Free returns free physical memory including buffers/caches/sreclaimable
func (s *MemStat) Free() float64 {
	o := s.Metrics
	return o.MemFree.V + o.Buffers.V + o.Cached.V + o.SReclaimable.V
}

// Usage returns physical memory in use; not including buffers/cached/sreclaimable
func (s *MemStat) Usage() float64 {
	o := s.Metrics
	return o.MemTotal.V - s.Free()
}

// Usage returns total physical memory
func (s *MemStat) Total() float64 {
	o := s.Metrics
	return o.MemTotal.V
}

type MemStatMetrics struct {
	MemTotal          *metrics.Gauge
	MemFree           *metrics.Gauge
	Buffers           *metrics.Gauge
	Cached            *metrics.Gauge
	SwapCached        *metrics.Gauge
	Active            *metrics.Gauge
	Inactive          *metrics.Gauge
	Active_anon       *metrics.Gauge
	Inactive_anon     *metrics.Gauge
	Active_file       *metrics.Gauge
	Inactive_file     *metrics.Gauge
	Unevictable       *metrics.Gauge
	Mlocked           *metrics.Gauge
	SwapTotal         *metrics.Gauge
	SwapFree          *metrics.Gauge
	Dirty             *metrics.Gauge
	Writeback         *metrics.Gauge
	AnonPages         *metrics.Gauge
	Mapped            *metrics.Gauge
	Shmem             *metrics.Gauge
	Slab              *metrics.Gauge
	SReclaimable      *metrics.Gauge
	SUnreclaim        *metrics.Gauge
	KernelStack       *metrics.Gauge
	PageTables        *metrics.Gauge
	NFS_Unstable      *metrics.Gauge
	Bounce            *metrics.Gauge
	WritebackTmp      *metrics.Gauge
	CommitLimit       *metrics.Gauge
	Committed_AS      *metrics.Gauge
	VmallocTotal      *metrics.Gauge
	VmallocUsed       *metrics.Gauge
	VmallocChunk      *metrics.Gauge
	HardwareCorrupted *metrics.Gauge
	AnonHugePages     *metrics.Gauge
	HugePages_Total   *metrics.Gauge
	HugePages_Free    *metrics.Gauge
	HugePages_Rsvd    *metrics.Gauge
	HugePages_Surp    *metrics.Gauge
	Hugepagesize      *metrics.Gauge
	DirectMap4k       *metrics.Gauge
	DirectMap2M       *metrics.Gauge
}

func MemStatMetricsNew(m *metrics.MetricContext) *MemStatMetrics {
	c := new(MemStatMetrics)

	// initialize all gauges
	s := reflect.ValueOf(c).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if f.Type().Elem() == reflect.TypeOf(metrics.Gauge{}) {
			f.Set(reflect.ValueOf(m.NewGauge(typeOfT.Field(i).Name)))
		}
	}

	// collect metrics every m.Step
	ticker := time.NewTicker(m.Step)
	go func() {
		for _ = range ticker.C {
			c.Collect()
		}
	}()

	return c
}

func (s *MemStatMetrics) Collect() {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}

	d := map[string]*metrics.Gauge{}
	// Get all fields we care about
	r := reflect.ValueOf(s).Elem()
	typeOfT := r.Type()
	for i := 0; i < r.NumField(); i++ {
		f := r.Field(i)
		if f.Type().Elem() == reflect.TypeOf(metrics.Gauge{}) {
			d[typeOfT.Field(i).Name] = f.Interface().(*metrics.Gauge)
		}
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		f := regexp.MustCompile("[:\\s]+").Split(scanner.Text(), 3)
		g, ok := d[f[0]]
		if ok {
			parseMemLine(g, f)
		}
	}
}

// Unexported functions
func parseMemLine(g *metrics.Gauge, f []string) {
	length := len(f)
	val := math.NaN()

	if length < 2 {
		goto fail
	}

	val = float64(misc.ParseUint(f[1]))
	if length > 2 && f[2] == "kB" {
		val *= 1024
	}

	g.Set(val)
	return

fail:
	g.Set(math.NaN())
	return
}
