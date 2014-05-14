// Copyright (c) 2014 Square, Inc

package metrics

import (
	"sync/atomic"
)

// BasicCounter is a minimal counter - all operations are atomic
// Arguments
//  name string - name to be registered with MetricContext
// Usage:
//   m := metrics.NewMetricContext("namespace")
//   b := m.NewBasicCounter("somecounter")
//   b.Add(1)
//   b.Get()
func (m *MetricContext) NewBasicCounter(name string) *BasicCounter {
	c := new(BasicCounter)
	c.m = m
	c.Register(name)
	c.Reset()
	return c
}

type BasicCounter struct {
	v uint64
	m *MetricContext
}

// Register with MetricContext, typically called from NewBasicCounter
func (c *BasicCounter) Register(name string) {
	if name == "" {
		return
	}
	c.m.BasicCounters[name] = c
}

// Reset counter to zero
func (c *BasicCounter) Reset() {
	atomic.StoreUint64(&c.v, 0)
}

// Set counter to value v.
func (c *BasicCounter) Set(v uint64) {
	atomic.StoreUint64(&c.v, v)
}

// Add delta to counter value v
func (c *BasicCounter) Add(delta uint64) {
	atomic.AddUint64(&c.v, delta)
}

// Get value of counter
func (c *BasicCounter) Get() uint64 {
	return c.v
}
