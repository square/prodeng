//Copyright (c) 2014 Square, Inc

package main

import (
	"fmt"
	"time"

	"./mysqlstat"
	"github.com/square/prodeng/metrics"
)

func main() {
	var stepSec int
	stepSec = 2
	m := metrics.NewMetricContext("system")
	step := time.Millisecond * time.Duration(stepSec) * 1000
	mysqlstat.New(m, step, "")

	fmt.Println("done.")
}
