// Copyright (c) 2014 Square, Inc

package main

import (
	"flag"
	"fmt"
	"github.com/square/prodeng/inspect/cpustat"
	"github.com/square/prodeng/inspect/diskstat"
	"github.com/square/prodeng/inspect/interfacestat"
	"github.com/square/prodeng/inspect/memstat"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/inspect/pidstat"
	"github.com/square/prodeng/metrics"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
)

// XXX: make it OS agnostic
func main() {
	// options
	var batchmode bool
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	flag.BoolVar(&batchmode, "b", false, "Run in batch mode; suitable for parsing")
	flag.BoolVar(&batchmode, "batchmode", false, "Run in batch mode; suitable for parsing")
	flag.Parse()

	// Enable Profiling
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	fmt.Println("Gathering statistics......")

	// Initialize a metric context with step 1 second and maximum
	// history of 3 samples
	m := metrics.NewMetricContext("system", time.Millisecond*1000*1, 3)

	// Collect cpu/memory/disk/interface metrics
	cstat := cpustat.New(m)
	mstat := memstat.New(m)
	dstat := diskstat.New(m)
	ifstat := interfacestat.New(m)

	cg_mem := memstat.NewCgroupStat(m)
	cg_cpu := cpustat.NewCgroupStat(m)

	// per process stats
	procs := pidstat.NewProcessStat(m)

	type cg_stat struct {
		cpu *cpustat.PerCgroupStat
		mem *memstat.PerCgroupStat
	}

	cg_stats := make(map[string]*cg_stat)

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

		for d, o := range dstat.Disks {
			fmt.Printf("disk: %s usage: %3.1f\n", d, o.Usage())
		}

		for iface, o := range ifstat.Interfaces {
			fmt.Printf("iface: %s TX: %s/s, RX: %s/s\n", iface,
				misc.BitSize(o.TXBandwidth()),
				misc.BitSize(o.RXBandwidth()))
		}

		// so much for printing cpu/mem stats for cgroup together
		for name, mem := range cg_mem.Cgroups {
			name, _ = filepath.Rel(cg_mem.Mountpoint, name)
			_, ok := cg_stats[name]
			if !ok {
				cg_stats[name] = new(cg_stat)
			}
			cg_stats[name].mem = mem
		}

		for name, cpu := range cg_cpu.Cgroups {
			name, _ = filepath.Rel(cg_cpu.Mountpoint, name)
			_, ok := cg_stats[name]
			if !ok {
				cg_stats[name] = new(cg_stat)
			}
			cg_stats[name].cpu = cpu
		}

		for name, s := range cg_stats {
			var out string

			out = fmt.Sprintf("cgroup:%s ", name)
			if s.cpu != nil {
				out += fmt.Sprintf(
					"cpu_throttling: %3.1f%% (%.1f/%d) ",
					s.cpu.Throttle(), s.cpu.Quota(),
					(len(cstat.CPUS()) - 1))
			}
			if s.mem != nil {
				out += fmt.Sprintf(
					"mem: %3.1f%% (%s/%s) ",
					(s.mem.Usage()/s.mem.SoftLimit())*100,
					misc.ByteSize(s.mem.Usage()), misc.ByteSize(s.mem.SoftLimit()))
			}
			fmt.Println(out)
		}

		// Top processes by usage
		procs_by_usage := procs.ByCPUUsage()
		fmt.Println("Top processes by CPU usage:")
		n := 5
		if len(procs_by_usage) < n {
			n = len(procs_by_usage)
		}

		for i := 0; i < n; i++ {
			fmt.Printf("usage: %3.1f  command: %s user: %s pid: %v\n",
				procs_by_usage[i].CPUUsage(),
				procs_by_usage[i].Metrics.Comm,
				procs_by_usage[i].Metrics.User,
				procs_by_usage[i].Metrics.Pid)
		}

		fmt.Println("---")
		procs_by_usage = procs.ByMemUsage()
		fmt.Println("Top processes by Mem usage:")
		n = 5
		if len(procs_by_usage) < n {
			n = len(procs_by_usage)
		}

		for i := 0; i < n; i++ {
			fmt.Printf("usage: %s, command: %s user: %s pid: %v\n",
				misc.ByteSize(procs_by_usage[i].MemUsage()),
				procs_by_usage[i].Metrics.Comm,
				procs_by_usage[i].Metrics.User,
				procs_by_usage[i].Metrics.Pid)
		}
	}
}
