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
	ticks     int64
}

type Counter struct {
	v       uint64
	p       uint64
	rate    float64
	ticks_p int64
	ticks_v int64
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
const NS_IN_SEC = 1 * 1000 * 1000 * 1000

func NewMetricContext(namespace string) *MetricContext {
	m := new(MetricContext)
	m.namespace = namespace
	m.Counters = make(map[string]*Counter)
	m.Gauges = make(map[string]*Gauge)

	start := time.Now().UnixNano()
	m.ticks = 0

	ticker := time.NewTicker(time.Millisecond * jiffy)
	go func() {
		for t := range ticker.C {
			m.ticks = t.UnixNano() - start
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

// BasicCounter is a minimal counter - all operations are atomic
func (m *MetricContext) NewBasicCounter(name string) *BasicCounter {
	c := new(BasicCounter)
	c.K = name
	c.m = m
	c.Reset()
	return c
}

type BasicCounter struct {
	v uint64
	K string
	m *MetricContext
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

// Counters
// Counters differ from BasicCounter by having additional
// fields for computing rate
// All basic counter operations are atomic and no locks are held

func (m *MetricContext) NewCounter(name string) *Counter {
	c := new(Counter)
	c.K = name
	c.m = m
	c.Reset()
	m.Counters[name] = c
	return c
}

func (c *Counter) Reset() {
	c.rate = 0.0
	c.ticks_p = 0
	c.ticks_v = 0
	c.v = 0
	c.p = 0
}

// Set Counter value. This is useful if you are reading a metric
// that is already a counter
func (c *Counter) Set(v uint64) {
	c.ticks_v = c.m.ticks
	atomic.StoreUint64(&c.v, v)

	// baseline for rate calculation
	if c.ticks_p == 0 {
		c.p = c.v
		c.ticks_p = c.ticks_v
	}
}

// Add value to counter
func (c *Counter) Add(delta uint64) {
	c.ticks_v = c.m.ticks
	atomic.AddUint64(&c.v, delta)

	// baseline for rate calculation
	if c.ticks_p == 0 {
		c.p = c.v
		c.ticks_p = c.ticks_v
	}
}

// Get value of counter
func (c *Counter) Get() uint64 {
	return c.v
}

// ComputeRate() calculates the rate of change of counter per
// second. (acquires a lock)
// Since we avoid locking on Set/Add operations, rate can be
// inaccurate on highly contended threads

func (c *Counter) ComputeRate() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	rate := 0.0

	delta_t := c.ticks_v - c.ticks_p
	delta_v := c.v - c.p

	// we have two samples, compute rate and
	// cache it away
	if delta_t > 0 && c.v >= c.p {
		rate = (float64(delta_v) / float64(delta_t)) * NS_IN_SEC
		// update baseline
		c.p = c.v
		c.ticks_p = c.ticks_v
		// cache rate calculated
		c.rate = rate
	}

	return c.rate
}

// Gauges

// NewGauge initializes a Gauge and returns it
func (m *MetricContext) NewGauge(name string) *Gauge {
	g := new(Gauge)
	g.K = name
	g.m = m
	g.Reset()
	m.Gauges[name] = g
	return g
}

//
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
