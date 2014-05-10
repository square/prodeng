// Copyright (c) 2014 Square, Inc

package metrics

import "testing"
import "time"
import "math"

// BUG: This test will most likely fail on a highly loaded
// system
func TestCounterRate(t *testing.T) {
	m := NewMetricContext("testing")
	c := m.NewCounter("testcounter")
	// increment counter every 10ms in two goroutines
	// rate ~ 200/sec
	tick1 := time.NewTicker(time.Millisecond * 10)
	go func() {
		for _ = range tick1.C {
			c.Add(1)
		}
	}()
	tick2 := time.NewTicker(time.Millisecond * 10)
	go func() {
		for _ = range tick2.C {
			c.Add(1)
		}
	}()
<<<<<<< HEAD

	time.Sleep(time.Millisecond * 5000)
=======
	time.Sleep(time.Millisecond * 1000)
>>>>>>> b2d4c0b3b73b470bc201a70433a35539ca975be8
	tick1.Stop()
	tick2.Stop()

	want := 200.0
	out := c.ComputeRate()

	if math.Abs(want - out) > 1 {
		t.Errorf("c.ComputeRate() = %v, want %v", out, want)
	}
}

func TestCounterRateNoChange(t *testing.T) {
	m := NewMetricContext("testing")
	c := m.NewCounter("testcounter")
	c.Set(0)
	time.Sleep(time.Millisecond*100)
	c.Set(0)
	want := 0.0
	out := c.ComputeRate()
	if math.IsNaN(out) || (math.Abs(out-want) > math.SmallestNonzeroFloat64) {
		t.Errorf("c.ComputeRate() = %v, want %v", out, want)
	}
}

func TestDefaultGaugeVal(t *testing.T) {
	m := NewMetricContext("testing")
	c := m.NewGauge("stuff")
	if !math.IsNaN(c.Get()) {
		t.Errorf("c.Get() = %v, want %v", c.Get(), math.NaN())
	}
}
func TestGaugePercentile(t *testing.T) {
	m := NewMetricContext("testing")
	c := m.NewGauge("stuff")
	if !math.IsNaN(c.Get()) {
		t.Errorf("c.Get() = %v, want %v", c.Get(), math.NaN())
	}
}
func TestDefaultCounterVal(t *testing.T) {
	m := NewMetricContext("testing")
	c := m.NewCounter("stuff")
	if c.Get() != 0 {
		t.Errorf("c.Get() = %v, want %v", c.Get(), 0)
	}
}
