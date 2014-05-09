// Copyright (c) 2014 Square, Inc

package osmain

import (
	"github.com/square/prodeng/metrics"
	"time"
)

type DarwinStats struct {
}

func RegisterOsDependent(
	m *metrics.MetricContext, step time.Duration,
	d *OsIndependentStats) *DarwinStats {

	x := new(DarwinStats)
	return x
}

func PrintOsDependent(d *DarwinStats, batchmode bool) {
}
