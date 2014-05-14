// Copyright (c) 2014 Square, Inc

package metrics

import (
	"math"
	"sync"
)

// Gauges
type Gauge struct {
	v  float64
	mu sync.RWMutex
	m  *MetricContext
}

// NewGauge initializes a Gauge and returns it
func (m *MetricContext) NewGauge(name string) *Gauge {
	g := new(Gauge)
	g.m = m
	g.Register(name)
	g.Reset()
	return g
}

// Register() with metrics context with name
// Usually called from NewGauge but useful if you have to
// re-use and existing object
func (g *Gauge) Register(name string) {
	g.m.Gauges[name] = g
}

// Reset() all values are reset to defaults
// Usually called from NewGauge but useful if you have to
// re-use and existing object
func (g *Gauge) Reset() {
	g.v = math.NaN()
}

// Set value of Gauge
func (g *Gauge) Set(v float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.v = v
}

// Get value of Gauge
func (g *Gauge) Get() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.v
}
