// Copyright (c) 2014 Square, Inc

package metrics

import "testing"
import "time"
import "math"

// BUG: This test will most likely fail on a highly loaded
// system
func TestCounterRate(t *testing.T) {
	m := NewMetricContext("testing",time.Millisecond * 100,1)
	c := m.NewCounter("testcounter")
	// increment counter every millisecond in two goroutines
	// rate ~ 20/step
        tick1 := time.NewTicker(time.Millisecond)
        go func() {
                for _ = range tick1.C {
			c.Add(1)
                }
        }()
        tick2 := time.NewTicker(time.Millisecond)
        go func() {
                for _ = range tick2.C {
			c.Add(1)
                }
        }()
	time.Sleep(time.Millisecond*1000)
	want := 10.0
	out  := c.CurRate()
	if math.Abs(out - want) > math.SmallestNonzeroFloat64 {
		t.Errorf("c.CurRate() = %f, want %f", out,want)
	}
}

func TestCounterRateNoChange(t *testing.T) {
	m := NewMetricContext("testing",time.Millisecond * 1,1)
	c := m.NewCounter("testcounter")
	c.Set(0)
	c.Set(0)
	want := 0.0
	out  := c.CurRate()
	if math.Abs(out - want) > math.SmallestNonzeroFloat64 {
		t.Errorf("c.CurRate() = %v, want %v", out,want)
	}
}

func TestDefaultGaugeVal(t *testing.T) {
	m := NewMetricContext("testing",time.Millisecond * 1,1)
	c := m.NewGauge("stuff")
	if !math.IsNaN(c.Get()) {
		t.Errorf("c.Get() = %v, want %v", c.Get(),math.NaN())
	}
}
func TestGaugePercentile(t *testing.T) {
	m := NewMetricContext("testing",time.Millisecond * 1,1)
	c := m.NewGauge("stuff")
	if !math.IsNaN(c.Get()) {
		t.Errorf("c.Get() = %v, want %v", c.Get(),math.NaN())
	}
}
func TestDefaultCounterVal(t *testing.T) {
	m := NewMetricContext("testing",time.Millisecond * 1,1)
	c := m.NewCounter("stuff")
	if c.Get() != 0 {
		t.Errorf("c.Get() = %v, want %v", c.Get(),0)
	}
}
