// Copyright (c) 2014 Square, Inc

package pidstat

import (
	"fmt"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
	"math"
	_ "os/user"
	"reflect"
	"sort"
	"time"
	"unsafe"
)

/*
#include <mach/mach.h>
#include <mach/task_info.h>
#include <sys/sysctl.h>
*/
import "C"

type ProcessStat struct {
	Processes map[string]*PerProcessStat
	m         *metrics.MetricContext
	hport     C.host_t
}

// NewProcessStat allocates a new ProcessStat object
// Arguments:
// m - *metricContext

// Collects metrics every m.Step seconds
// Drops refresh interval by m.Step for every additional
// 1024 processes
// TODO: Implement better heuristics to manage load
//   * Collect metrics for newer processes at faster rate
//   * Slower rate for processes with neglible rate?

func NewProcessStat(m *metrics.MetricContext) *ProcessStat {
	c := new(ProcessStat)
	c.m = m

	c.Processes = make(map[string]*PerProcessStat, 1024)
	c.hport = C.host_t(C.mach_host_self())

	var n int
	ticker := time.NewTicker(m.Step)
	go func() {
		for _ = range ticker.C {
			p := int(len(c.Processes) / 1024)
			if n == 0 {
				c.Collect(true)
			}
			// always collect all metrics for first two samples
			// and if number of processes < 1024
			if p < 1 || n%p == 0 {
				c.Collect(false)
			}
			n++
		}
	}()

	return c
}

// ByCPUUsage implements sort.Interface for []*PerProcessStat based on
// the Usage() method

type ByCPUUsage []*PerProcessStat

func (a ByCPUUsage) Len() int           { return len(a) }
func (a ByCPUUsage) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByCPUUsage) Less(i, j int) bool { return a[i].CPUUsage() > a[j].CPUUsage() }

// ByCPUUsage() returns an slice of *PerProcessStat entries sorted
// by CPU usage
func (c *ProcessStat) ByCPUUsage() []*PerProcessStat {
	v := make([]*PerProcessStat, 0)
	for _, o := range c.Processes {
		if !math.IsNaN(o.CPUUsage()) {
			v = append(v, o)
		}
	}
	sort.Sort(ByCPUUsage(v))
	return v
}

type ByMemUsage []*PerProcessStat

func (a ByMemUsage) Len() int           { return len(a) }
func (a ByMemUsage) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByMemUsage) Less(i, j int) bool { return a[i].MemUsage() > a[j].MemUsage() }

// ByMemUsage() returns an slice of *PerProcessStat entries sorted
// by Memory usage
func (c *ProcessStat) ByMemUsage() []*PerProcessStat {
	v := make([]*PerProcessStat, 0)
	for _, o := range c.Processes {
		if !math.IsNaN(o.MemUsage()) {
			v = append(v, o)
		}
	}
	sort.Sort(ByMemUsage(v))
	return v
}

// reference /usr/include/mach/task_info.h
// works on MacOSX 10.9.2; YMMV might vary

func (c *ProcessStat) Collect(collectAttributes bool) {

	h := c.Processes
	for _, v := range h {
		v.dead = true
	}

	var pDefaultSet C.processor_set_name_t
	var pDefaultSetControl C.processor_set_t
	var tasks C.task_array_t
	var taskCount C.mach_msg_type_number_t

	if C.processor_set_default(c.hport, &pDefaultSet) != C.KERN_SUCCESS {
		return
	}

	// get privileged port to get information about all tasks

	if C.host_processor_set_priv(C.host_priv_t(c.hport),
		pDefaultSet, &pDefaultSetControl) != C.KERN_SUCCESS {
		return
	}

	if C.processor_set_tasks(pDefaultSetControl, &tasks, &taskCount) != C.KERN_SUCCESS {
		return
	}

	// convert tasks to a Go slice
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(tasks)),
		Len:  int(taskCount),
		Cap:  int(taskCount),
	}

	goTaskList := *(*[]C.task_name_t)(unsafe.Pointer(&hdr))

	// mach_msg_type_number_t - type natural_t = uint32_t
	now := time.Now().UnixNano()
	var i uint32
	for i = 0; i < uint32(taskCount); i++ {

		taskId := goTaskList[i]
		var pid C.int
		// var tinfo C.task_info_data_t
		var count C.mach_msg_type_number_t
		var taskBasicInfo C.mach_task_basic_info_data_t
		var taskAbsoluteInfo C.task_absolutetime_info_data_t

		if (C.pid_for_task(C.mach_port_name_t(taskId), &pid) != C.KERN_SUCCESS) ||
			(pid < 0) {
			continue
		}

		count = C.MACH_TASK_BASIC_INFO_COUNT
		kr := C.task_info(taskId, C.MACH_TASK_BASIC_INFO,
			(C.task_info_t)(unsafe.Pointer(&taskBasicInfo)),
			&count)
		if kr != C.KERN_SUCCESS {
			continue
		}

		spid := fmt.Sprintf("%v", pid)
		pidstat, ok := h[spid]
		if !ok {
			pidstat = NewPerProcessStat(c.m, spid)
			h[spid] = pidstat
		}

		if collectAttributes || !ok {
			pidstat.CollectAttributes()
		}

		pidstat.Metrics.VirtualSize.Set(float64(taskBasicInfo.virtual_size))
		pidstat.Metrics.ResidentSize.Set(float64(taskBasicInfo.resident_size))
		pidstat.Metrics.ResidentSizeMax.Set(float64(taskBasicInfo.resident_size_max))

		count = C.TASK_ABSOLUTETIME_INFO_COUNT
		kr = C.task_info(taskId, C.TASK_ABSOLUTETIME_INFO,
			(C.task_info_t)(unsafe.Pointer(&taskAbsoluteInfo)),
			&count)
		if kr != C.KERN_SUCCESS {
			continue
		}
		pidstat.Metrics.UserTime.Set(uint64(taskAbsoluteInfo.total_user))
		pidstat.Metrics.SystemTime.Set(uint64(taskAbsoluteInfo.total_system))
		pidstat.Metrics.UpdatedAt.Set(uint64(now - pidstat.Metrics.StartedAt))
		pidstat.dead = false
	}

	// remove dead processes
	for k, v := range h {
		if v.dead {
			delete(h, k)
		}
	}

}

// Per Process functions
type PerProcessStat struct {
	Pid     string
	Uid     int
	User    string
	Comm    string
	Metrics *PerProcessStatMetrics
	m       *metrics.MetricContext
	dead    bool
}

func NewPerProcessStat(m *metrics.MetricContext, p string) *PerProcessStat {
	c := new(PerProcessStat)
	c.m = m
	c.Pid = p
	c.Metrics = NewPerProcessStatMetrics(m)
	return c
}

func (s *PerProcessStat) CPUUsage() float64 {
	o := s.Metrics
	t := o.UpdatedAt.CurRate()
	return ((o.UserTime.CurRate() + o.SystemTime.CurRate()) / t) * 100
}

func (s *PerProcessStat) MemUsage() float64 {
	o := s.Metrics
	return o.ResidentSize.Get()
}

type PerProcessStatMetrics struct {
	VirtualSize     *metrics.Gauge
	ResidentSize    *metrics.Gauge
	ResidentSizeMax *metrics.Gauge
	UserTime        *metrics.Counter
	SystemTime      *metrics.Counter
	UpdatedAt       *metrics.Counter
	StartedAt       int64
}

func NewPerProcessStatMetrics(m *metrics.MetricContext) *PerProcessStatMetrics {
	s := new(PerProcessStatMetrics)
	s.StartedAt = time.Now().UnixNano()
	// initialize all metrics
	misc.InitializeMetrics(s, m)
	return s
}

func (s *PerProcessStat) CollectAttributes() {

}
