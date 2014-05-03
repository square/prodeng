// Copyright (c) 2014 Square, Inc

package metrics

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type MetricContext struct {
	namespace  string
	TrackStats bool
	Step       time.Duration
	Nsamples   int
	Counters   map[string]*Counter
	Gauges     map[string]*Gauge
}

type Counter struct {
	V          uint64
	K          string
	Step       time.Duration
	Nsamples   int
	// Fixed buffer and a marker for tracking statistics
	idx     int
	history []uint64
}

type Gauge struct {
	V          float64
	K          string
	Step       time.Duration
	Nsamples   int
	// Fixed buffer and a marker for tracking statistics
	idx     int
	history []float64
}

// Creates a new metric context. A metric context specifies a namespace
// time duration that is used as step and number of samples to keep
// in-memory

func NewMetricContext(namespace string, Step time.Duration, Nsamples int) *MetricContext {
	m := new(MetricContext)
	m.namespace = namespace
	m.Step = Step
	m.Nsamples = Nsamples
	m.Counters = make(map[string]*Counter)
	m.Gauges = make(map[string]*Gauge)
	return m
}

// print ALL metrics to stdout
func (m *MetricContext) Print() {
	for key, value := range m.Counters {
		fmt.Printf("counter: %s , value: %d history: %v \n", key, value.V, value.history)
	}
	for key, value := range m.Gauges {
		fmt.Printf("gauge: %s , value: %f history: %v \n", key, value.V,
			value.history)
	}
}

// Update statistics for all metrics
func (m *MetricContext) UpdateStats() {
	for _, g := range m.Gauges {
		g.UpdateStats()
	}

	for _, c := range m.Counters {
		c.UpdateStats()
	}
}

// Counters

func (m *MetricContext) NewCounter(name string) *Counter {
	c := new(Counter)
	c.K = name
	c.Nsamples = m.Nsamples
	// We need atleast two samples for rate calcuation
	if c.Nsamples < 2 {
		c.Nsamples = 2
	}
	c.history = make([]uint64, c.Nsamples) // 0 is a-ok as not updated value
	c.Step = m.Step
	m.Counters[name] = c
	return c
}

// Set Counter Value. This is useful if you are reading a metric
// that is already a counter
// Note: calls UpdateStats()
func (c *Counter) Set(v uint64) {
	c.V = v
	c.UpdateStats()
}

// Increment the counter value
// Note: UpdateStats() is not called
func (c *Counter) Inc() {
	c.V++
}

// Store current value in history
// This function or Set needs to be called atleast twice for
// rate calculation
func (c *Counter) UpdateStats() {
	// Store current value in history
	c.history[c.idx] = c.V
	c.idx++
	if c.idx == c.Nsamples {
		c.idx = 0
	}
}

// CurRate() calculates change of value over time as indicated
// step.
// XXX: add detection for counter wrap / counter reset
func (c *Counter) CurRate() float64 {
	var a_idx, b_idx int
	var a, b uint64

	// get two latest points
	// c.idx points to the index of latest
	// element that we stored + 1

	a_idx = c.idx - 1
	if a_idx < 0 {
		a_idx += len(c.history)
	}

	b_idx = c.idx - 2
	if b_idx < 0 {
		b_idx += len(c.history)
	}

	a = c.history[a_idx]
	b = c.history[b_idx]

	if a > b {
		return float64(a-b) / c.Step.Seconds()
	}
	if a == b {
		return 0
	}

	return math.NaN()
}

// Gauges
// NewGauge initializes a Gauge and returns it
func (m *MetricContext) NewGauge(name string) *Gauge {
	g := new(Gauge)
	g.K = name
	g.V = math.NaN()
	g.Nsamples = m.Nsamples
	g.history = make([]float64, g.Nsamples)
	for i, _ := range g.history {
		g.history[i] = math.NaN()
	}
	m.Gauges[name] = g
	return g
}

// Set value of Gauge
func (g *Gauge) Set(v float64) {
	g.V = v
}

// UpdateStats() stores the current value in history
func (g *Gauge) UpdateStats() {
	g.history[g.idx] = g.V
	g.idx = (g.idx + 1) % g.Nsamples
}

// Percentile
// should be in statistics package
func (g *Gauge) Percentile(percentile float64) float64 {
	// Nearest rank implementation
	// http://en.wikipedia.org/wiki/Percentile

	if percentile > 100 {
		panic(fmt.Sprintf("Percentile out of bounds (should be <100): %f",
			percentile))
	}

	// Since slices are zero-indexed, we are naturally rounded up
	nearest_rank := int((percentile / 100) * float64(g.Nsamples))

	if nearest_rank == g.Nsamples {
		nearest_rank = g.Nsamples - 1
	}

	in := make([]float64, g.Nsamples)
	copy(in, g.history)

	sort.Float64s(in)

	return in[nearest_rank]
}
