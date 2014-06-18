// Copyright (c) 2014 Square, Inc
// +build linux darwin

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/mgutz/ansi"
	"github.com/square/prodeng/inspect/cpustat"
	"github.com/square/prodeng/inspect/memstat"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/inspect/osmain"
	"github.com/square/prodeng/inspect/pidstat"
	"github.com/square/prodeng/metrics"
)

const DISPLAY_PID_COUNT = 5

func main() {
	// options
	var batchmode, servermode bool
	var address string
	var stepSec int

	flag.BoolVar(&batchmode, "b", false, "Run in batch mode; suitable for parsing")
	flag.BoolVar(&batchmode, "batchmode", false, "Run in batch mode; suitable for parsing")
	flag.BoolVar(&servermode, "server", false,
		"Runs continously and exposes metrics as JSON on HTTP")
	flag.StringVar(&address, "address", ":19999",
		"address to listen on for http if running in server mode")
	flag.IntVar(&stepSec, "step", 2,
		"metrics are collected every step seconds")
	flag.Parse()

	if servermode {
		batchmode = true
	}

	if !batchmode {
		fmt.Println("Gathering statistics......")
	}

	// Initialize a metric context
	m := metrics.NewMetricContext("system")

	// Default step for collectors
	step := time.Millisecond * time.Duration(stepSec) * 1000

	// Collect cpu/memory/disk/per-pid metrics
	cstat := cpustat.New(m, step)
	mstat := memstat.New(m, step)
	procs := pidstat.NewProcessStat(m, step)

	// Filter processes which have < 1% CPU or < 1% memory
	// and try to keep minimum of 5

	procs.SetPidFilter(pidstat.PidFilterFunc(func(p *pidstat.PerProcessStat) bool {

		if len(procs.Processes) < DISPLAY_PID_COUNT {
			return true
		}

		if p.CPUUsage() > 1.0 {
			return true
		}
		memUsagePct := (p.MemUsage() / mstat.Total()) * 100.0
		if memUsagePct > 1.0 {
			return true
		}
		return false
	}))

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
	d := osmain.RegisterOsDependent(m, step, osind)

	// run http server
	if servermode {
		go func() {
			http.HandleFunc("/api/v1/metrics.json", m.HttpJsonHandler)
			log.Fatal(http.ListenAndServe(address, nil))
		}()
	}

	// command line refresh every 2 step
	ticker := time.NewTicker(step * 2)
	for _ = range ticker.C {

		// Problems
		var problems []string

		if !batchmode {
			fmt.Printf("\033[2J") // clear screen
			fmt.Printf("\033[H")  // move cursor top left top
		}

		fmt.Println("--------------------------")
		mem_pct_usage := (mstat.Usage() / mstat.Total()) * 100
		fmt.Printf(
			"total: cpu: %3.1f%%, mem: %3.1f%% (%s/%s)\n",
			cstat.Usage(), mem_pct_usage,
			misc.ByteSize(mstat.Usage()), misc.ByteSize(mstat.Total()))

		if cstat.Usage() > 80.0 {
			problems = append(problems, "CPU usage > 80%")
		}

		if mem_pct_usage > 80.0 {
			problems = append(problems, "Memory usage > 80%")
		}

		// Top processes by usage
		procs_by_usage := procs.ByCPUUsage()
		fmt.Println("Top processes by CPU usage:")
		n := DISPLAY_PID_COUNT
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
		n = DISPLAY_PID_COUNT
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

		osmain.PrintOsDependent(d, batchmode)

		for i := range problems {
			msg := problems[i]
			if !batchmode {
				msg = ansi.Color(msg, "red")
			}
			fmt.Println("Problem: ", msg)
		}

		// be aggressive about reclaiming memory
		// tradeoff with CPU usage
		runtime.GC()
		debug.FreeOSMemory()
	}
}
