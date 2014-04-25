// Copyright (c) 2014 Square, Inc

package metrics

import "testing"
import "time"
import "math"

func TestCounterRate(t *testing.T) {
	m := NewMetricContext("testing",time.Millisecond * 1,1)
	c := m.NewCounter("testcounter")
	c.Set(10)
	time.Sleep(time.Millisecond)
	c.Set(100)
	want := 90.0 * 1000
	out  := c.CurRate()
	// TODO: find better ways to compare floats
	if math.Abs(out - want) > 0.0001 {
		t.Errorf("c.CurRate() = %v, want %v", out,want)
	}
}

func TestDefaultGaugeVal(t *testing.T) {
	m := NewMetricContext("testing",time.Millisecond * 1,1)
	c := m.NewGauge("stuff")
	if !math.IsNaN(c.V) {
		t.Errorf("c.V = %v, want %v", c.V,math.NaN())
	}
}
func TestGaugePercentile(t *testing.T) {
	m := NewMetricContext("testing",time.Millisecond * 1,1)
	c := m.NewGauge("stuff")
	if !math.IsNaN(c.V) {
		t.Errorf("c.V = %v, want %v", c.V,math.NaN())
	}
}
func TestDefaultCounterVal(t *testing.T) {
	m := NewMetricContext("testing",time.Millisecond * 1,1)
	c := m.NewCounter("stuff")
	if c.V != 0 {
		t.Errorf("c.V = %v, want %v", c.V,0)
	}
}
