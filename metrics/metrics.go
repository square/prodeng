// Copyright (c) 2014 Square, Inc

package metrics

import (
	"fmt"
	"math"
	"sort"
	"time"
	"sync"
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
	v          uint64
	K          string
	// Fixed buffer and a marker for tracking statistics
	idx     int
	history []uint64
	m	*MetricContext
	mu	sync.RWMutex
}

type Gauge struct {
	v          float64
	K          string
	// Fixed buffer and a marker for tracking statistics
	idx     int
	history []float64
	m	*MetricContext
	mu	sync.RWMutex
}

// Creates a new metric context. A metric context specifies a namespace
// time duration that is used as step and number of samples to keep
// in-memory

func NewMetricContext(namespace string, Step time.Duration, Nsamples int) *MetricContext {
	m := new(MetricContext)
	m.namespace = namespace
	m.Step = Step
	if Nsamples < 2 {
		Nsamples = 2
	}
	m.Nsamples = Nsamples
	m.Counters = make(map[string]*Counter)
	m.Gauges = make(map[string]*Gauge)
	return m
}

// print ALL metrics to stdout
func (m *MetricContext) Print() {
	for key, value := range m.Counters {
		fmt.Printf("counter: %s , value: %d history: %v \n", key, value.v, value.history)
	}
	for key, value := range m.Gauges {
		fmt.Printf("gauge: %s , value: %f history: %v \n", key, value.v,
			value.history)
	}
}

// Counters

func (m *MetricContext) NewCounter(name string) *Counter {
	c := new(Counter)
	c.K = name
	c.m = m
	c.history = make([]uint64, c.m.Nsamples) // 0 is a-ok as not updated value
	m.Counters[name] = c
	return c
}

// Set Counter value. This is useful if you are reading a metric
// that is already a counter
func (c *Counter) Set(v uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.v = v
	c.updateStats()
}

// Add value to counter
func (c *Counter) Add(delta uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.v += delta
	c.updateStats()
}

// Get value of counter
func (c *Counter) Get() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.v
}

// unexported
// Store current value in history
// This function or Set needs to be called atleast twice for
// rate calculation
func (c *Counter) updateStats() {
	// Store current value in history
	c.history[c.idx] = c.v
	c.idx++
	if c.idx == c.m.Nsamples {
		c.idx = 0
	}
}

// CurRate() calculates change of value over Step
// Unit: seconds
// TODO: add detection for counter wrap / counter reset

func (c *Counter) CurRate() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
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

	// Triggered in two use cases
	// 1. counters have not been updated at all
	// 2. no updates to counters
	// TODO: return NaN in case of 1

	if a == 0 && b == 0 {
		return 0
	}

	if a >= b && b > 0  {
		return float64(a-b) / c.m.Step.Seconds()
	}

	return math.NaN()
}

// Gauges
// NewGauge initializes a Gauge and returns it
func (m *MetricContext) NewGauge(name string) *Gauge {
	g := new(Gauge)
	g.K = name
	g.v = math.NaN()
	g.m = m
	g.history = make([]float64, g.m.Nsamples)
	for i, _ := range g.history {
		g.history[i] = math.NaN()
	}
	m.Gauges[name] = g
	return g
}

// Set value of Gauge
func (g *Gauge) Set(v float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.v = v
	g.updateStats()
}

func (g *Gauge) Get() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.v
}

// Percentile
// should be in statistics package
func (g *Gauge) Percentile(percentile float64) float64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Nearest rank implementation
	// http://en.wikipedia.org/wiki/Percentile

	if percentile > 100 {
		panic(fmt.Sprintf("Percentile out of bounds (should be <100): %f",
			percentile))
	}

	// Since slices are zero-indexed, we are naturally rounded up
	nearest_rank := int((percentile / 100) * float64(g.m.Nsamples))

	if nearest_rank == g.m.Nsamples {
		nearest_rank = g.m.Nsamples - 1
	}

	in := make([]float64, g.m.Nsamples)
	copy(in, g.history)

	sort.Float64s(in)

	return in[nearest_rank]
}

// unexported

// updateStats() stores the current value in history
func (g *Gauge) updateStats() {
	g.history[g.idx] = g.v
	g.idx = (g.idx + 1) % g.m.Nsamples
}
