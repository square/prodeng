// Copyright (c) 2014 Square, Inc

package main

import (
	"fmt"
	"github.com/square/prodeng/inspect/cpustat"
	"github.com/square/prodeng/inspect/memstat"
	"github.com/square/prodeng/metrics"
	"path"
	"time"
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

func main() {

	// Initialize a metric context with step 1 second and maximum
	// history of 3 samples
	m := metrics.NewMetricContext("system", time.Millisecond*1000, 3)

	// Collect cpu/memory metrics
	cpu := cpustat.New(m)
	mem := memstat.New(m)

	cg_mem := memstat.NewCgroupStat(m)

	fmt.Println("Gathering metrics....")

	// Check metrics every 2s
	ticker := time.NewTicker(time.Millisecond * 1000 * 2)
	for _ = range ticker.C {
		fmt.Println("--------------------------")
		fmt.Printf(
			"cpu: %3.1f%%, mem: %3.1f%% (%s/%s)\n",
			cpu.Usage(), (mem.Usage()/mem.Total())*100,
			ByteSize(mem.Usage()), ByteSize(mem.Total()))

		for name, c := range cg_mem.Cgroups {
			fmt.Printf(
				"cgroup: %s, mem: %3.1f%% (%s/%s)\n", path.Base(name),
				(c.Usage()/c.SoftLimit())*100,
				ByteSize(c.Usage()), ByteSize(c.SoftLimit()))
		}
	}
}
