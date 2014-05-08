// Copyright (c) 2014 Square, Inc

package metrics

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type MetricContext struct {
	namespace string
	Counters  map[string]*Counter
	Gauges    map[string]*Gauge
	ticks     uint64
}

type Counter struct {
	v       uint64
	p       uint64
	rate    float64
	ticks_p uint64
	ticks_v uint64
	K       string
	m       *MetricContext
	mu      sync.RWMutex
}

type Gauge struct {
	v  float64
	K  string
	m  *MetricContext
	mu sync.RWMutex
}

// Creates a new metric context. A metric context specifies a namespace
// time duration that is used as step and number of samples to keep
// in-memory
// Arguments:
// namespace - namespace that all metrics in this context belong to

const jiffy = 100

func NewMetricContext(namespace string) *MetricContext {
	m := new(MetricContext)
	m.namespace = namespace
	m.Counters = make(map[string]*Counter)
	m.Gauges = make(map[string]*Gauge)

	// TODO: make this configurable
	ticker := time.NewTicker(time.Millisecond * jiffy)
	go func() {
		for _ = range ticker.C {
			m.ticks++
		}
	}()

	return m
}

// print ALL metrics to stdout
func (m *MetricContext) Print() {
	for key, value := range m.Counters {
		fmt.Printf("counter: %s value: %d previous: %v \n", key,
			value.v, value.p)
	}
	for key, value := range m.Gauges {
		fmt.Printf("gauge: %s , value: %f \n", key, value.v)
	}
}

// Counters
func (m *MetricContext) NewCounter(name string) *Counter {
	c := new(Counter)
	c.K = name
	c.m = m
	c.rate = 0.0
	m.Counters[name] = c
	return c
}

// Set Counter value. This is useful if you are reading a metric
// that is already a counter
func (c *Counter) Set(v uint64) {
	if c.ticks_p == 0 {
		c.p = c.v
		c.ticks_p = c.ticks_v
	}
	c.ticks_v = c.m.ticks
	atomic.StoreUint64(&c.v, v)
}

// Add value to counter
func (c *Counter) Add(delta uint64) {
	if c.ticks_p == 0 {
		c.p = c.v
		c.ticks_p = c.ticks_v
	}
	c.ticks_v = c.m.ticks
	atomic.AddUint64(&c.v, delta)
}

// Get value of counter
func (c *Counter) Get() uint64 {
	return c.v
}

// ComputeRate() calculates the rate of change of counter per
// second.
// Since we avoid locking on Set/Add operations, rate can be
// inaccurate on highly contended threads
func (c *Counter) ComputeRate() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	rate := math.NaN()
	delta_t := c.ticks_v - c.ticks_p
	delta_v := c.v - c.p

	// handle special cases

	// no updates yet
	if c.ticks_v == 0 {
		c.rate = math.NaN()
		return c.rate
	}

	// check if our counter stays at zero
	if c.p == 0 && c.v == 0 {
		c.rate = 0.0
		return c.rate
	}

	// return cached rate if no new samples
	// are seen
	if c.p == c.v {
		return c.rate
	}

	// we have two samples, compute rate and
	// cache it away
	if delta_t > 0 && c.v >= c.p {
		rate = (float64(delta_v) / float64(delta_t)) * (1000 / jiffy)
		c.p = c.v
		c.ticks_p = c.ticks_v
		c.rate = rate
	}

	return c.rate
}

// Gauges

// NewGauge initializes a Gauge and returns it
func (m *MetricContext) NewGauge(name string) *Gauge {
	g := new(Gauge)
	g.K = name
	g.v = math.NaN()
	g.m = m
	m.Gauges[name] = g
	return g
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
