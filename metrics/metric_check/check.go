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
	testnagiosconfigfile  = "./test_nagios.config"
)

// basic checker
// starts loop and prints checks against config file
func main() {
	var hostport, configFile, nagConfigFile string
	var basic, nagios bool
	var stepSec int

	flag.StringVar(&hostport, "hostport", "localhost:12345", "hostport to grab metrics")
	flag.StringVar(&configFile, "conf", "", "config file to read metric thresholds")
	flag.StringVar(&nagConfigFile, "nagConf", "", "config file to send nagios messages")
	flag.IntVar(&stepSec, "step", 2, "time step in between sending messages to nagios")
	flag.BoolVar(&basic, "basic", true, "output check results in basic format")
	flag.BoolVar(&nagios, "nagios", false, "output check results in nagios format")
	flag.Parse()
	if configFile == "" {
		configFile = testconfigurationfile
	}
	if nagConfigFile == "" {
		nagConfigFile = testconfigurationfile
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
		if basic {
			hc.OutputWarnings(formats.Basic)
		}
		if nagios {
			hc.OutputWarnings(formats.Nagios, nagConfigFile)
		}
	}
}
