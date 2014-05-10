// Copyright (c) 2014 Square, Inc

package metrics

import "testing"
import "time"
import "math"
import "sync"
import "fmt"

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

	time.Sleep(time.Millisecond * 5000)
	tick1.Stop()
	tick2.Stop()

	want := 200.0
	out := c.ComputeRate()

	if math.Abs(want-out) > 1 {
		t.Errorf("c.ComputeRate() = %v, want %v", out, want)
	}
}

func TestCounterRateNoChange(t *testing.T) {
	m := NewMetricContext("testing")
	c := m.NewCounter("testcounter")
	c.Set(0)
	time.Sleep(time.Millisecond * 100)
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

func TestStatsTimer(t *testing.T) {
	m := NewMetricContext("testing")
	s := m.NewStatsTimer("stuff", time.Millisecond, 100) // keep 100 samples
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		x := i + 1
		go func() {
			defer wg.Done()
			fmt.Println(x, time.Now())
			stopWatch := s.Start()
			time.Sleep(time.Millisecond * time.Duration(x*100))
			s.Stop(stopWatch)
		}()
	}

	// block till all goroutines finish
	wg.Wait()

	pctile, err := s.Percentile(100)
	if math.Abs(pctile - 100) > 5 || err != nil {
		t.Errorf("Percentile expected: 100 got: %v", pctile)
	}

	pctile, err = s.Percentile(75)
	if math.Abs(pctile - 75) > 5 || err != nil {
		t.Errorf("Percentile expected: 75 got: %v", pctile)
	}
}
