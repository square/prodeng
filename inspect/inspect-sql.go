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
	stepSec = 2
	m := metrics.NewMetricContext("system")

	step := time.Millisecond * time.Duration(stepSec) * 1000
	s, err := mysqlstat.New(m, step, "")

	fmt.Println("Done!")
}
