// Copyright (c) 2014 Square, Inc

package metrics

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

type StatsTimer struct {
	history  []int64
	idx      int
	K        string
	m        *MetricContext
	mu       sync.RWMutex
	timeUnit time.Duration
}

const NOT_INITIALIZED = -1

// StatsTimer
func (m *MetricContext) NewStatsTimer(name string, timeUnit time.Duration, nsamples int) *StatsTimer {
	s := new(StatsTimer)
	s.K = name
	s.m = m
	s.timeUnit = timeUnit
	s.history = make([]int64, nsamples)

	for i := range s.history {
		s.history[i] = NOT_INITIALIZED
	}
	return s
}

func (s *StatsTimer) Start() *Timer {
	t := s.m.NewTimer()
	t.Start()
	return t
}

func (s *StatsTimer) Stop(t *Timer) {
	delta := t.Stop()

	// Store current value in history
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history[s.idx] = delta
	s.idx++
	if s.idx == len(s.history) {
		s.idx = 0
	}
}

// TODO: move stats implementation to a dedicated package

type Int64Slice []int64

func (a Int64Slice) Len() int           { return len(a) }
func (a Int64Slice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Int64Slice) Less(i, j int) bool { return a[i] < a[j] }

func (s *StatsTimer) Percentile(percentile float64) (float64, error) {
	// Nearest rank implementation
	// http://en.wikipedia.org/wiki/Percentile
	histLen := len(s.history)

	if percentile > 100 {
		return math.NaN(), errors.New("Invalid argument")
	}

	in := make([]int64, 0, histLen)
	for i := range s.history {
		if s.history[i] != NOT_INITIALIZED {
			in = append(in, s.history[i])
		}
	}

	filtLen := len(in)

	// Since slices are zero-indexed, we are naturally rounded up
	nearest_rank := int((percentile / 100) * float64(filtLen))

	if nearest_rank == filtLen {
		nearest_rank = filtLen - 1
	}

	sort.Sort(Int64Slice(in))
	ret := float64(in[nearest_rank]) / float64(s.timeUnit.Nanoseconds())

	return ret, nil
}
