// Copyright (c) 2014 Square, Inc

package misc

import (
	"github.com/square/prodeng/metrics"
)

type DarwinMetrics struct {
}

func RegisterOsDependent(m *metrics.MetricContext) *DarwinMetrics {
	x := new(DarwinMetrics)
	return x
}

func PrintOsDependent(d *DarwinMetrics) {
}
