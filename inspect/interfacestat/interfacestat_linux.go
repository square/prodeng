// Copyright (c) 2014 Square, Inc

package interfacestat

import (
	"bufio"
	"fmt"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
	"os"
	"strings"
	"time"
)

type InterfaceStat struct {
	Interfaces map[string]*PerInterfaceStat
	m          *metrics.MetricContext
}

func New(m *metrics.MetricContext, Step time.Duration) *InterfaceStat {
	s := new(InterfaceStat)
	s.Interfaces = make(map[string]*PerInterfaceStat, 4)
	s.m = m

	ticker := time.NewTicker(Step)
	go func() {
		for _ = range ticker.C {
			s.Collect()
		}
	}()

	return s
}

// Collect() collects interface metrics
// TODO: perhaps use sysfs
func (s *InterfaceStat) Collect() {
	file, err := os.Open("/proc/net/dev")
	defer file.Close()
	if err != nil {
		return
	}

	var rx [8]uint64
	var tx [8]uint64

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	for scanner.Scan() {
		f := strings.Split(scanner.Text(), ":")
		if len(f) < 2 {
			continue
		}
		dev := strings.TrimSpace(f[0])
		rest := f[1]
		fmt.Sscanf(rest,
			"%d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d",
			&rx[0], &rx[1], &rx[2], &rx[3], &rx[4], &rx[5], &rx[6], &rx[7],
			&tx[0], &tx[1], &tx[2], &tx[3], &tx[4], &tx[5], &tx[6], &tx[7])

		o, ok := s.Interfaces[dev]
		if !ok {
			o = NewPerInterfaceStat(s.m, dev)
			s.Interfaces[dev] = o
		}

		d := o.Metrics
		d.RXbytes.Set(rx[0])
		d.RXpackets.Set(rx[1])
		d.RXerrs.Set(rx[2])
		d.RXdrop.Set(rx[3])
		d.RXfifo.Set(rx[4])
		d.RXframe.Set(rx[5])
		d.RXcompressed.Set(rx[6])
		d.RXmulticast.Set(rx[7])
		d.TXbytes.Set(tx[0])
		d.TXpackets.Set(tx[1])
		d.TXerrs.Set(tx[2])
		d.TXdrop.Set(tx[3])
		d.TXfifo.Set(tx[4])
		d.TXframe.Set(tx[5])
		d.TXcompressed.Set(tx[6])
		d.TXmulticast.Set(tx[7])
		speed := misc.ReadUintFromFile("/sys/class/net/" + dev + "/speed")
		if speed > 0 {
			d.Speed.Set(float64(speed))
		}
	}
}

type PerInterfaceStat struct {
	Metrics *PerInterfaceStatMetrics
	m       *metrics.MetricContext
}

// bytes    packets errs drop fifo frame compressed multicast
type PerInterfaceStatMetrics struct {
	RXbytes      *metrics.Counter
	RXpackets    *metrics.Counter
	RXerrs       *metrics.Counter
	RXdrop       *metrics.Counter
	RXfifo       *metrics.Counter
	RXframe      *metrics.Counter
	RXcompressed *metrics.Counter
	RXmulticast  *metrics.Counter
	TXbytes      *metrics.Counter
	TXpackets    *metrics.Counter
	TXerrs       *metrics.Counter
	TXdrop       *metrics.Counter
	TXfifo       *metrics.Counter
	TXframe      *metrics.Counter
	TXcompressed *metrics.Counter
	TXmulticast  *metrics.Counter
	Speed        *metrics.Gauge
}

func NewPerInterfaceStat(m *metrics.MetricContext, dev string) *PerInterfaceStat {
	c := new(PerInterfaceStat)
	c.Metrics = new(PerInterfaceStatMetrics)
	// initialize all metrics and register them
	misc.InitializeMetrics(c.Metrics, m, "interfacestat."+dev, true)
	return c
}

// Recieve bandwidth utilization in bits/sec
func (s *PerInterfaceStat) RXBandwidth() float64 {
	o := s.Metrics
	return (o.RXbytes.ComputeRate()) * 8
}

// Transmit bandwidth utilization in bits/sec
func (s *PerInterfaceStat) TXBandwidth() float64 {
	o := s.Metrics
	return (o.TXbytes.ComputeRate()) * 8
}

// Speed of interface in bits/sec
func (s *PerInterfaceStat) Speed() float64 {
	o := s.Metrics
	return o.Speed.Get() * 1024 * 1024 // ethtool interface seems to report in Mb
}

// Recieve bandwidth usage as percentage
func (s *PerInterfaceStat) RXBandwidthUsage() float64 {
	return (s.RXBandwidth() / s.Speed()) * 100
}

// Transmit bandwidth usage as percentage
func (s *PerInterfaceStat) TXBandwidthUsage() float64 {
	return (s.TXBandwidth() / s.Speed()) * 100
}
