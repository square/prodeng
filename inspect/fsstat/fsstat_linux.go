// Copyright (c) 2014 Square, Inc

// file system disk statistics
package fsstat

import (
	"bufio"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
	"os"
	"strings"
	"syscall"
	"time"
)

type FSStat struct {
	FS map[string]*PerFSStat
	m  *metrics.MetricContext
}

func New(m *metrics.MetricContext, Step time.Duration) *FSStat {
	s := new(FSStat)
	s.FS = make(map[string]*PerFSStat, 0)
	s.m = m

	ticker := time.NewTicker(Step)
	go func() {
		for _ = range ticker.C {
			s.Collect()
		}
	}()

	return s
}

func (s *FSStat) Collect() {
	file, err := os.Open("/etc/mtab")
	defer file.Close()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		f := strings.Split(scanner.Text(), " ")

		//TODO: XXX: exclusions should be configurable

		// ignore few device types
		if f[0] == "proc" || f[0] == "sysfs" || f[0] == "devpts" ||
			f[0] == "none" || f[0] == "sunrpc" {
			continue
		}

		// ignore few types of mounts
		// man fstab
		if f[3] == "swap" || f[3] == "bind" || f[3] == "ignore" || f[3] == "none" {
			continue
		}

		o, ok := s.FS[f[1]]
		if !ok {
			o = NewPerFSStat(s.m, f[1])
			s.FS[f[1]] = o
		}
		o.Collect()
	}
}

type PerFSStat struct {
	Metrics *PerFSStatMetrics
	m       *metrics.MetricContext
	mp      string
}

// man statfs
type PerFSStatMetrics struct {
	Bsize  *metrics.Gauge
	Blocks *metrics.Gauge
	Bfree  *metrics.Gauge
	Bavail *metrics.Gauge
	Files  *metrics.Gauge
	Ffree  *metrics.Gauge
}

func NewPerFSStat(m *metrics.MetricContext, mp string) *PerFSStat {
	c := new(PerFSStat)
	c.mp = mp
	c.Metrics = new(PerFSStatMetrics)
	misc.InitializeMetrics(c.Metrics, m, "fsstat."+mp, true)
	return c
}

func (s *PerFSStat) Collect() {

	// call statfs and populate metrics
	buf := new(syscall.Statfs_t)
	err := syscall.Statfs(s.mp, buf)
	if err != nil {
		return
	}

	s.Metrics.Bsize.Set(float64(buf.Bsize))
	s.Metrics.Blocks.Set(float64(buf.Blocks))
	s.Metrics.Bfree.Set(float64(buf.Bfree))
	s.Metrics.Bavail.Set(float64(buf.Bavail))
	s.Metrics.Files.Set(float64(buf.Files))
	s.Metrics.Ffree.Set(float64(buf.Ffree))
}

// Filesystem block usage in percentage
func (s *PerFSStat) Usage() float64 {
	o := s.Metrics
	total := o.Blocks.Get()
	free := o.Bfree.Get()
	return ((total - free) / total) * 100
}

// Filesystem file node usage
func (s *PerFSStat) FileUsage() float64 {
	o := s.Metrics
	total := o.Files.Get()
	free := o.Ffree.Get()
	return ((total - free) / total) * 100
}
