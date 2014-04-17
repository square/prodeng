// Copyright (c) 2014 Square, Inc

package main

import (
	"fmt"
	"time"
	"path/filepath"
	"github.com/square/prodeng/inspect/cpustat"
	"github.com/square/prodeng/inspect/memstat"
	"github.com/square/prodeng/metrics"
)

type ByteSize float64

const (
	_           = iota
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
	ZB
	YB
)

func (b ByteSize) String() string {
	switch {
	case b >= YB:
		return fmt.Sprintf("%.2fYB", b/YB)
	case b >= ZB:
		return fmt.Sprintf("%.2fZB", b/ZB)
	case b >= EB:
		return fmt.Sprintf("%.2fEB", b/EB)
	case b >= PB:
		return fmt.Sprintf("%.2fPB", b/PB)
	case b >= TB:
		return fmt.Sprintf("%.2fTB", b/TB)
	case b >= GB:
		return fmt.Sprintf("%.2fGB", b/GB)
	case b >= MB:
		return fmt.Sprintf("%.2fMB", b/MB)
	case b >= KB:
		return fmt.Sprintf("%.2fKB", b/KB)
	}
	return fmt.Sprintf("%.2fB", b)
}

// XXX: make it OS agnostic
func main() {

	// Initialize a metric context with step 1 second and maximum
	// history of 3 samples
	m := metrics.NewMetricContext("system", time.Millisecond*1000, 3)

	// Collect cpu/memory metrics
	cstat := cpustat.New(m)
	mstat := memstat.New(m)

	cg_mem := memstat.NewCgroupStat(m)
	cg_cpu := cpustat.NewCgroupStat(m)

	type cg_stat struct {
		cpu *cpustat.PerCgroupStat
		mem *memstat.PerCgroupStat
	}

	cg_stats := make(map[string]*cg_stat)

	// Check metrics every 2s
	ticker := time.NewTicker(time.Millisecond * 1000 * 2)
	for _ = range ticker.C {
		fmt.Println("--------------------------")
		fmt.Printf(
			"total: cpu: %3.1f%%, mem: %3.1f%% (%s/%s)\n",
			cstat.Usage(), (mstat.Usage()/mstat.Total())*100,
			ByteSize(cstat.Usage()), ByteSize(mstat.Total()))

		// so much for printing cpu/mem stats for cgroup together
		for name, mem := range cg_mem.Cgroups {
			name,_ = filepath.Rel(cg_mem.Mountpoint,name)
			_,ok := cg_stats[name]
			if !ok {
				cg_stats[name] = new(cg_stat)
			}
			cg_stats[name].mem = mem
		}

		for name, cpu := range cg_cpu.Cgroups {
			name,_ = filepath.Rel(cg_cpu.Mountpoint,name)
			_,ok := cg_stats[name]
			if !ok {
				cg_stats[name] = new(cg_stat)
			}
			cg_stats[name].cpu = cpu
		}


		for name,s := range cg_stats {
			var out string

			out = fmt.Sprintf("cgroup:%s ",name)
			if s.cpu != nil {
				out += fmt.Sprintf(
				"cpu: %3.1f%% (%.1f/%d) ",
				 s.cpu.Usage(), s.cpu.Quota(),
				(len(cstat.CPUS()) - 1))
			}
			if s.mem != nil {
				out += fmt.Sprintf(
				"mem: %3.1f%% (%s/%s) ",
				(s.mem.Usage()/s.mem.SoftLimit())*100,
				ByteSize(s.mem.Usage()), ByteSize(s.mem.SoftLimit()))
			}
			fmt.Println(out)
		}
	}
}
