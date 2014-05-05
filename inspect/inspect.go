// Copyright (c) 2014 Square, Inc
// +build linux darwin

package main

import (
	"flag"
	"fmt"
	"github.com/square/prodeng/inspect/cpustat"
	"github.com/square/prodeng/inspect/memstat"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/inspect/osmain"
	"github.com/square/prodeng/inspect/pidstat"
	"github.com/square/prodeng/metrics"
	"time"
)

func main() {
	// options
	var batchmode bool

	flag.BoolVar(&batchmode, "b", false, "Run in batch mode; suitable for parsing")
	flag.BoolVar(&batchmode, "batchmode", false, "Run in batch mode; suitable for parsing")
	flag.Parse()

	fmt.Println("Gathering statistics......")

	// Initialize a metric context with step 1 second and maximum
	// history of 3 samples
	m := metrics.NewMetricContext("system", time.Millisecond*1000*1, 2)

	// Collect cpu/memory/disk/per-pid metrics
	cstat := cpustat.New(m)
	mstat := memstat.New(m)
	procs := pidstat.NewProcessStat(m)

	// pass the collected metrics to OS dependent set if they
	// need it
	osind := new(osmain.OsIndependentStats)
	osind.Cstat = cstat
	osind.Mstat = mstat
	osind.Procs = procs

	// register os dependent metrics
	// these could be specific to the OS (say cgroups)
	// or stats which are implemented not on all supported
	// platforms yet
	d := osmain.RegisterOsDependent(m, osind)

	// Check metrics every 2s
	ticker := time.NewTicker(time.Millisecond * 1100 * 2)
	for _ = range ticker.C {
		if !batchmode {
			fmt.Printf("\033[2J") // clear screen
			fmt.Printf("\033[H")  // move cursor top left top
		}
		fmt.Println("--------------------------")
		fmt.Printf(
			"total: cpu: %3.1f%%, mem: %3.1f%% (%s/%s)\n",
			cstat.Usage(), (mstat.Usage()/mstat.Total())*100,
			misc.ByteSize(mstat.Usage()), misc.ByteSize(mstat.Total()))

		// Top processes by usage
		procs_by_usage := procs.ByCPUUsage()
		fmt.Println("Top processes by CPU usage:")
		n := 5
		if len(procs_by_usage) < n {
			n = len(procs_by_usage)
		}

		for i := 0; i < n; i++ {
			fmt.Printf("cpu: %3.1f%%  command: %s user: %s pid: %v\n",
				procs_by_usage[i].CPUUsage(),
				procs_by_usage[i].Comm(),
				procs_by_usage[i].User(),
				procs_by_usage[i].Pid())
		}

		fmt.Println("---")
		procs_by_usage = procs.ByMemUsage()
		fmt.Println("Top processes by Mem usage:")
		n = 5
		if len(procs_by_usage) < n {
			n = len(procs_by_usage)
		}

		for i := 0; i < n; i++ {
			fmt.Printf("mem: %s command: %s user: %s pid: %v\n",
				misc.ByteSize(procs_by_usage[i].MemUsage()),
				procs_by_usage[i].Comm(),
				procs_by_usage[i].User(),
				procs_by_usage[i].Pid())
		}

		osmain.PrintOsDependent(d)
	}
}
