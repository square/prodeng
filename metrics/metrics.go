// Copyright (c) 2014 Square, Inc

package metrics

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// package initialization code
// sets up a ticker to "cache" time

var TICKS int64

func init() {
	start := time.Now().UnixNano()
	ticker := time.NewTicker(time.Millisecond * jiffy)
	go func() {
		for t := range ticker.C {
			TICKS = t.UnixNano() - start
		}
	}()
}

type MetricContext struct {
	namespace     string
	Counters      map[string]*Counter
	Gauges        map[string]*Gauge
	BasicCounters map[string]*BasicCounter
	StatsTimers   map[string]*StatsTimer
}

type metric interface {
	GetJson(name string, allowNaN bool) []byte
}

// Creates a new metric context. A metric context specifies a namespace
// time duration that is used as step and number of samples to keep
// in-memory
// Arguments:
// namespace - namespace that all metrics in this context belong to

const jiffy = 100

const NS_IN_SEC = float64(time.Second) //nanoseconds in a second represented in float64

// default percentiles to compute when serializing statstimer type
// to stdout/json
var percentiles = []float64{50, 75, 95, 99, 99.9, 99.99, 99.999}

func NewMetricContext(namespace string) *MetricContext {
	m := new(MetricContext)
	m.namespace = namespace
	m.Counters = make(map[string]*Counter, 0)
	m.Gauges = make(map[string]*Gauge, 0)
	m.BasicCounters = make(map[string]*BasicCounter, 0)
	m.StatsTimers = make(map[string]*StatsTimer, 0)

	return m
}

// Register(v Metric) registers a metric with metric
// context
func (m *MetricContext) Register(v interface{}, name string) {
	switch v := v.(type) {
	case *BasicCounter:
		m.BasicCounters[name] = v
	case *Counter:
		m.Counters[name] = v
	case *Gauge:
		m.Gauges[name] = v
	case *StatsTimer:
		m.StatsTimers[name] = v
	}
}

// Unregister(v Metric) unregisters a metric with metric
// context
func (m *MetricContext) Unregister(v interface{}, name string) {
	switch v.(type) {
	case *BasicCounter:
		delete(m.BasicCounters, name)
	case *Counter:
		delete(m.Counters, name)
	case *Gauge:
		delete(m.Gauges, name)
	case *StatsTimer:
		delete(m.StatsTimers, name)
	}
}

// Print() prints ALL metrics to stdout
func (m *MetricContext) Print() {
	for name, c := range m.Counters {
		fmt.Printf("counter %s %d %.3f \n", name,
			c.Get(), c.ComputeRate())
	}
	for name, g := range m.Gauges {
		fmt.Printf("gauge %s %.3f \n", name, g.Get())
	}
	for name, c := range m.BasicCounters {
		fmt.Printf("basiccounter %s %d \n", name, c.Get())
	}

	for name, s := range m.StatsTimers {
		out := ""
		for _, p := range percentiles {
			percentile, err := s.Percentile(p)
			if err == nil {
				out += fmt.Sprintf(".3f", percentile)
			}
		}
		fmt.Printf("statstimer %s %s \n", name, out)
	}
}

// HttpJsonHandler metrics via json
func (m *MetricContext) HttpJsonHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		return
	}
	allowNaN := true // if allowNaN is set to false, filter out NaN metric values
	if n, ok := r.Form["allowNaN"]; ok && strings.ToLower(n[0]) == "false" {
		allowNaN = false
	}
	paths := ParseURL(r.URL.Path)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[\n"))
	WriteMetrics(m.FilterMetrics(paths...), allowNaN, w)
	w.Write([]byte("]"))
	w.Write([]byte("\n")) // Be nice to curl
}

func ParseURL(url string) []string {
	path := strings.SplitN(url, "metrics.json", 2)[1]
	levels := strings.Split(path, "/")
	return levels[1:]
}

//filter metrics
// return a map of metric name -> metric, return metrics that are of the given type and match the input regexp metricnames
func (m *MetricContext) FilterMetrics(metricnames ...string) map[interface{}]metric {
	types := metricnames[0]
	metricsToCollect := map[interface{}]metric{}
	if strings.Contains(types, "Gauges") {
		for k, v := range m.Gauges {
			metricsToCollect[k] = v
		}
	}
	if strings.Contains(types, "Counters") {
		for k, v := range m.Counters {
			metricsToCollect[k] = v
		}
	}
	if strings.Contains(types, "StatsTimers") {
		for k, v := range m.StatsTimers {
			metricsToCollect[k] = v
		}
	}
	if len(metricnames) == 1 {
		return metricsToCollect
	}
	for nameFound, _ := range metricsToCollect {
		for _, nameLookingFor := range metricnames[1:] {
			re := regexp.MustCompile(nameLookingFor)
			if !re.MatchString(nameFound.(string)) {
				delete(metricsToCollect, nameFound)
			}
		}
	}
	return metricsToCollect
}

//write metrics given
func WriteMetrics(m map[interface{}]metric, allowNaN bool, w io.Writer) error {
	prependcomma := false
	for name, metric := range m {
		if prependcomma {
			w.Write([]byte(",\n"))
			prependcomma = false
		}
		b := metric.GetJson(name.(string), allowNaN)
		if b == nil {
			continue
		}
		w.Write(b)
		prependcomma = true
	}
	return nil
}
