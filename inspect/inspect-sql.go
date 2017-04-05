//Copyright (c) 2014 Square, Inc

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/square/prodeng/inspect/mysqlstat"
	"github.com/square/prodeng/metrics"
)

func main() {
	var stepSec int
	stepSec = 2
	m := metrics.NewMetricContext("system")
	var user, password string

	flag.StringVar(&user, "u", "", "user using database")
	flag.StringVar(&password, "p", "", "password for database")

	//to test:
	servermode := true
	address := ":12345"
	if servermode {
		fmt.Println("starting json handler")
		go func() {
			http.HandleFunc("/metrics.json", m.HttpJsonHandler)
			log.Fatal(http.ListenAndServe(address, nil))
		}()
	}
	step := time.Millisecond * time.Duration(stepSec) * 1000
	_, err := mysqlstat.New(m, step, user, password)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Done!")
}
