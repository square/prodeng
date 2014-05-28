//Copyright (c) 2014 Square, Inc

package main

import (
	"fmt"
	"time"

	"github.com/square/prodeng/inspect/mysqlstat"
	"github.com/square/prodeng/metrics"
)

func main() {
	var stepSec int

	m := metrics.NewMetricContext("system")
	step := time.Millisecond * time.Duration(stepSec) * 1000
	s, err := mysqlstat.New(m, step, nil)

	s.get_slave_stats()
	fmt.Println("done.")
}
