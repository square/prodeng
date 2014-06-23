//Copyright (c) 2014 Square, Inc

package check

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"code.google.com/p/go.tools/go/types"
	"code.google.com/p/goconf/conf" // used for parsing config files
)

//Used for nagios formatted warnings
const (
	OK = iota
	WARN
	CRIT
)

type checker struct {
	hostport string
	Metrics  map[string]metric
	Warnings map[string]metricResults
	c        *conf.ConfigFile
	/*
		routers          map[string]string //nagios use: maps service name to a regexp string that matches metrics collected for that service
		nagServer        string            //used for nagios messages
		serviceType      string            //mysql, postgres, etc.
		hostname         string            //used for nagios messages
		NSCA_BINARY_PATH string            //used for nagios messages
		NSCA_CONFIG_PATH string            //used for nagios messages
	*/
}

type metricThresholds struct {
	metricblob string
	checks     map[string]string
}

type metricResults struct {
	Message string
	Checks  map[string]bool // maps metric name to result
}

type metric struct {
	Type  string
	Name  string
	Value float64
	Rate  float64
}

//Creates new checker
//hostport is address to listen on for metrics json
func New(hostport, configFile string) (Checker, error) {
	c, err := conf.ReadConfigFile(configFile)
	if err != nil {
		return nil, err
	}
	hc := &checker{
		hostport: hostport, //hostport to listen on for metrics json
		Metrics:  make(map[string]metric),
		Warnings: make(map[string]metricResults),
		c:        c,
	}
	return hc, nil
}

func (hc *checker) OutputWarnings(printer func(Checker, ...string) error, s ...string) error {
	err := printer(hc, s[0])
	return err
}

//gets metrics and unmarshals from JSON
func (hc *checker) getMetrics() error {
	//get metrics from metrics collector
	resp, err := http.Get("http://" + hc.hostport + "/api/v1/metrics.json?allowNaN=false")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	//unmarshal metrics
	var metrics []metric
	err = json.Unmarshal(body, &metrics)
	if err != nil {
		return err
	}
	//store metrics in map, so they can be found easily by name
	for _, m := range metrics {
		hc.Metrics[m.Name] = m
	}
	return nil
}

//Checks all metrics metrics
// iterates through checks in config file and checks against collected metrics
func (hc *checker) CheckMetrics() error {
	err := hc.getMetrics()
	if err != nil {
		fmt.Println(err)
		return err
	}
	//iterate through all sections of tests
	for _, sectionName := range hc.c.GetSections() {
		if sectionName == "default" || sectionName == "nagios" {
			continue
		}
		m := getConfigChecks(hc.c, sectionName)
		hc.Warnings[sectionName] = hc.checkMetric(m)
	}
	return nil
}

//Check single section against its tests
func (hc *checker) checkMetric(m metricThresholds) metricResults {
	res := &metricResults{}
	res.Checks = make(map[string]bool)
	for name, check := range m.checks {
		checkVal, err := hc.replaceNames(check)
		if err != nil {
			fmt.Println(err)
		}
		_, result, err := types.Eval(checkVal, nil, nil)
		if err != nil {
			continue
		}
		res.Checks[name], _ = strconv.ParseBool(result.String())
	}
	return *res
}

//finds and replaces names of other metrics inside expression
func (hc *checker) replaceNames(expr string) (string, error) {
	words := strings.Split(expr, " ")
	for _, word := range words {
		if strings.Contains(word, ".") {
			parts := strings.Split(word, ".")
			metricName := strings.Join(parts[:len(parts)-1], ".")
			m, ok := hc.Metrics[metricName]
			if !ok {
				continue
			}
			if parts[len(parts)-1] == "Value" {
				expr = strings.Replace(expr, word, strconv.FormatFloat(m.Value, 'f', 5, 64), -1)
			} else if parts[len(parts)-1] == "Rate" {
				expr = strings.Replace(expr, word, strconv.FormatFloat(m.Rate, 'f', 5, 64), -1)
			}
		}
	}
	return expr, nil
}

//Reads the thresholds and messages from the config file
func getConfigChecks(c *conf.ConfigFile, test string) metricThresholds {
	m := &metricThresholds{}
	m.checks = make(map[string]string)
	checks, _ := c.GetOptions(test)
	for _, checkName := range checks {
		if checkName == "metric-name" {
			continue
		}
		m.checks[checkName], _ = c.GetString(test, checkName)
	}
	return *m
}

func (hc *checker) GetWarnings() map[string]metricResults {
	return hc.Warnings
}
