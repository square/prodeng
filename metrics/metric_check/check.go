//Copyright (c) 2014 Square, Inc

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/square/prodeng/metrics/check"
	"github.com/square/prodeng/metrics/check/formats"
)

var (
	testconfigurationfile = "./test.config"
)

// basic checker
// starts loop and prints checks against config file
func main() {
	var hostport, configFile string
	var stepSec int

	flag.StringVar(&hostport, "hostport", "localhost:12345", "hostport to grab metrics")
	flag.StringVar(&configFile, "conf", "", "config file to read metric thresholds")
	flag.IntVar(&stepSec, "step", 2, "time step in between sending messages to nagios")
	flag.Parse()
	if configFile == "" {
		configFile = testconfigurationfile
	}

	fmt.Println("starting metrics checker on: ", hostport)

	hc, err := check.New(hostport, configFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	step := time.Millisecond * time.Duration(stepSec) * 1000
	ticker := time.NewTicker(step)
	for _ = range ticker.C {
		hc.CheckMetrics()
		hc.OutputWarnings(formats.Nagios, "./test_nagios.config")
	}
}
