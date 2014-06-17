// Copyright (c) 2014 Square, Inc
// +build linux

package osmain

import (
	"fmt"
	"github.com/mgutz/ansi"
	"github.com/square/prodeng/inspect/cpustat"
	"github.com/square/prodeng/inspect/diskstat"
	"github.com/square/prodeng/inspect/interfacestat"
	"github.com/square/prodeng/inspect/memstat"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/inspect/pidstat"
	"github.com/square/prodeng/metrics"
	"path/filepath"
	"time"
)

type LinuxStats struct {
	dstat  *diskstat.DiskStat
	ifstat *interfacestat.InterfaceStat
	cg_mem *memstat.CgroupStat
	cg_cpu *cpustat.CgroupStat
	procs  *pidstat.ProcessStat
	cstat  *cpustat.CPUStat
}

func RegisterOsDependent(m *metrics.MetricContext, step time.Duration,
	d *OsIndependentStats) *LinuxStats {

	s := new(LinuxStats)
	s.dstat = diskstat.New(m, step)
	s.ifstat = interfacestat.New(m, step)
	s.procs = d.Procs // grab it because we need to for per cgroup cpu usage
	s.cstat = d.Cstat
	s.cg_mem = memstat.NewCgroupStat(m, step)
	s.cg_cpu = cpustat.NewCgroupStat(m, step)

	return s
}

func PrintOsDependent(s *LinuxStats, batchmode bool) {

	var problems []string

	fmt.Println("---")
	procs_by_usage := s.procs.ByIOUsage()
	fmt.Println("Top processes by IO usage:")
	n := 5
	if len(procs_by_usage) < n {
		n = len(procs_by_usage)
	}

	for i := 0; i < n; i++ {
		fmt.Printf("io: %s/s command: %s user: %s pid: %v\n",
			misc.ByteSize(procs_by_usage[i].IOUsage()),
			procs_by_usage[i].Comm(),
			procs_by_usage[i].User(),
			procs_by_usage[i].Pid())
	}
	type cg_stat struct {
		cpu *cpustat.PerCgroupStat
		mem *memstat.PerCgroupStat
	}

	fmt.Println("---")
	for d, o := range s.dstat.Disks {
		fmt.Printf("diskio: %s usage: %3.1f%%\n", d, o.Usage())
		if o.Usage() > 75.0 {
			problems = append(problems,
				fmt.Sprintf("Disk IO usage on (%v): %3.1f%%", d, o.Usage()))
		}
	}

	fmt.Println("---")
	for iface, o := range s.ifstat.Interfaces {
		fmt.Printf("iface: %s TX: %3.1f%% (%s/s), RX: %3.1f%% (%s/s)\n",
			iface,
			o.TXBandwidthUsage(),
			misc.BitSize(o.TXBandwidth()),
			o.RXBandwidthUsage(),
			misc.BitSize(o.RXBandwidth()))

		if o.TXBandwidthUsage() > 75.0 {
			problems = append(problems,
				fmt.Sprintf("TX bandwidth usage on (%v): %3.1f%%",
					iface, o.TXBandwidthUsage()))
		}

		if o.RXBandwidthUsage() > 75.0 {
			problems = append(problems,
				fmt.Sprintf("RX bandwidth usage on (%v): %3.1f%%",
					iface, o.RXBandwidthUsage()))
		}
	}

	fmt.Println("---")
	// so much for printing cpu/mem stats for cgroup together
	cg_stats := make(map[string]*cg_stat)
	for name, mem := range s.cg_mem.Cgroups {
		name, _ = filepath.Rel(s.cg_mem.Mountpoint, name)
		_, ok := cg_stats[name]
		if !ok {
			cg_stats[name] = new(cg_stat)
		}
		cg_stats[name].mem = mem
	}

	for name, cpu := range s.cg_cpu.Cgroups {
		name, _ = filepath.Rel(s.cg_cpu.Mountpoint, name)
		_, ok := cg_stats[name]
		if !ok {
			cg_stats[name] = new(cg_stat)
		}
		cg_stats[name].cpu = cpu
	}

	for name, v := range cg_stats {
		var out string

		out = fmt.Sprintf("cgroup:%s ", name)
		if v.cpu != nil {
			// get CPU usage per cgroup from pidstat
			// unfortunately this is not exposed at cgroup level
			cpu_usage := s.procs.CPUUsagePerCgroup(name)
			cpu_throttle := v.cpu.Throttle()
			out += fmt.Sprintf("cpu: %3.1f%% ", cpu_usage)
			out += fmt.Sprintf(
				"cpu_throttling: %3.1f%% (%.1f/%d) ",
				cpu_throttle, v.cpu.Quota(),
				(len(s.cstat.CPUS()) - 1))
			if cpu_throttle > 0.5 {
				problems =
					append(problems,
						fmt.Sprintf(
							"CPU throttling on cgroup(%s): %3.1f%%",
							name, cpu_throttle))
			}

		}

		if v.mem != nil {
			out += fmt.Sprintf(
				"mem: %3.1f%% (%s/%s) ",
				(v.mem.Usage()/v.mem.SoftLimit())*100,
				misc.ByteSize(v.mem.Usage()), misc.ByteSize(v.mem.SoftLimit()))
		}
		fmt.Println(out)
	}

	fmt.Println("---")
	for i := range problems {
		msg := problems[i]
		if !batchmode {
			msg = ansi.Color(msg, "red")
		}
		fmt.Println("Problem: ", msg)
	}
}
