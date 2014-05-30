//Copyright (c) 2014 Square, Inc

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/square/prodeng/inspect-mysql/mysqlstat"
	"github.com/square/prodeng/inspect-mysql/mysqlstattable"
	"github.com/square/prodeng/metrics"
)

func main() {
	var user, password, address string
	var stepSec int
	var servermode bool

	m := metrics.NewMetricContext("system")

	flag.StringVar(&user, "u", "root", "user using database")
	flag.StringVar(&password, "p", "", "password for database")
	flag.BoolVar(&servermode, "server", false, "Runs continously and exposes metrics as JSON on HTTP")
	flag.StringVar(&address, "address", ":6969", "address to listen on for http if running in server mode")
	flag.IntVar(&stepSec, "step", 2, "metrics are collected every step seconds")
	flag.Parse()

	if servermode {
		go func() {
			http.HandleFunc("/metrics.json", m.HttpJsonHandler)
			log.Fatal(http.ListenAndServe(address, nil))
		}()
	}
	step := time.Millisecond * time.Duration(stepSec) * 1000
	sqlstat, err := mysqlstat.New(m, step, user, password)
	if err != nil {
		fmt.Println(err)
		return
	}
	sqlstatTables, err := mysqlstattable.New(m, step, user, password)
	if err != nil {
		fmt.Println(err)
		return
	}
	ticker := time.NewTicker(step * 2)
	for _ = range ticker.C {
		//Print stats here
		fmt.Println("--------------------------")
		fmt.Println("Queries made: " + strconv.Itoa(int(sqlstat.Queries())))
		fmt.Println("Uptime: " + strconv.Itoa(int(sqlstat.Uptime())))
		fmt.Println("Database sizes: ")
		for dbname, db := range sqlstatTables.DBs {
			fmt.Println("    " + dbname + ": " + strconv.Itoa(int(db.Metrics.Size.Get())))
		}
	}

}
