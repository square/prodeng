package check

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"code.google.com/p/goconf/conf" // used for parsing config files
	"github.com/square/prodeng/metrics"
)

func initChecker(t testing.TB) checker {
	hc := checker{
		hostport: "localhost:12345",
		Metrics:  make(map[string]metric),
		Warnings: make(map[string]metricResults),
		Logger:   log.New(os.Stderr, "LOG: ", log.Lshortfile),
	}
	return hc
}

func initConfigFile() *conf.ConfigFile {
	c := conf.NewConfigFile()
	c.AddSection("constants")
	c.AddOption("constants", "const1", "1")
	c.AddOption("constants", "const2", "2")
	c.AddSection("section1")
	c.AddOption("section1", "check1", "value1")
	c.AddOption("section1", "check2", "value2")
	c.AddSection("section2")
	c.AddOption("section2", "check1", "valueA")
	c.AddOption("section2", "check2", "valueB")
	c.AddOption("section2", "check3", "valueC")
	return c
}

var (
	expectedValues = map[string]float64{
		"testGauge2": float64(200),
		"testGauge3": float64(300),
		"testGauge4": float64(400),
		"testGauge5": float64(500)}
)

func initMetricsJson() {
	_, err := http.Get("http://localhost:12345/api/v1/metrics.json")
	if err == nil {
		return
	}
	m := metrics.NewMetricContext("test")
	g1 := metrics.NewGauge()
	m.Register(g1, "testGauge1")
	g2 := metrics.NewGauge()
	m.Register(g2, "testGauge2")
	g3 := metrics.NewGauge()
	m.Register(g3, "testGauge3")
	g4 := metrics.NewGauge()
	m.Register(g4, "testGauge4")
	g5 := metrics.NewGauge()
	m.Register(g5, "testGauge5")
	g2.Set(float64(200))
	g3.Set(float64(300))
	g4.Set(float64(400))
	g5.Set(float64(500))
	go func() {
		http.HandleFunc("/api/v1/metrics.json", m.HttpJsonHandler)
		http.ListenAndServe("localhost:12345", nil)
	}()
}

//Tests get metrics json correctly
func TestGetMetrics(t *testing.T) {
	//initialize checkers
	hc := initChecker(t)
	initMetricsJson()
	//get metrics here
	err := hc.getMetrics()
	if err != nil {
		t.Fatal(err)
	}
	//now check we collected the right metrics
	for name, metric := range hc.Metrics {
		v, ok := expectedValues[name]
		if !ok {
			t.Errorf("Unexpected metric collected: " + name)
			continue
		}
		if metric.Value != v {
			t.Errorf(fmt.Sprintf("Unexpected value in %s. Expected %f, got %f", name, v, metric.Value))
		}
	}
}

//tests replacement of names in expressions correctly
func TestReplaceNames1(t *testing.T) {
	expr := "testGauge2.Value > 100"
	hc := initChecker(t)
	initMetricsJson()
	hc.getMetrics()
	result, err := hc.replaceNames(expr)
	if err != nil {
		t.Fatal(err)
	}
	expected := "200.00000 > 100"
	if result != expected {
		t.Error(fmt.Sprintf("Expected %s, but got %s", expected, result))
	}
}

func TestReplaceNames2(t *testing.T) {
	expr := "testGauge2.Rate > 100"
	hc := initChecker(t)
	initMetricsJson()
	hc.getMetrics()
	result, err := hc.replaceNames(expr)
	if err != nil {
		t.Fatal(err)
	}
	expected := "0.00000 > 100"
	if result != expected {
		t.Error(fmt.Sprintf("Expected %s, but got %s", expected, result))
	}
}

func TestReplaceNames3(t *testing.T) {
	expr := "testGauge2.Value > testGauge2.Rate"
	hc := initChecker(t)
	initMetricsJson()
	hc.getMetrics()
	result, err := hc.replaceNames(expr)
	if err != nil {
		t.Fatal(err)
	}
	expected := "200.00000 > 0.00000"
	if result != expected {
		t.Error(fmt.Sprintf("Expected %s, but got %s", expected, result))
	}
}

//tests correctly.checks metrics against thresholds correctly
func TestCheckMetrics1(t *testing.T) {
	hc := initChecker(t)
	initMetricsJson()
	hc.getMetrics()
	m := metricThresholds{
		checks: map[string]string{
			"1": "testGauge2.Value > 199",
			"2": "testGauge2.Value == 200 ",
			"3": "testGauge2.Value <= 205",
		},
	}
	result := hc.checkMetric(m)
	if result.Checks["1"] != true {
		t.Errorf("Did not make check 1 correctly")
	}
	if result.Checks["2"] != true {
		t.Errorf("Did not make check 2 correctly")
	}
	if result.Checks["3"] != true {
		t.Errorf("Did not make check 3 correctly")
	}
}

func TestCheckMetrics2(t *testing.T) {
	hc := initChecker(t)
	initMetricsJson()
	hc.getMetrics()
	m := metricThresholds{
		checks: map[string]string{
			"1": "testGauge2.Value < 199",
			"2": "testGauge2.Value != 200 ",
			"3": "testGauge2.Value >= 205",
		},
	}
	result := hc.checkMetric(m)
	if result.Checks["1"] != false {
		t.Errorf("Did not make check 1 correctly")
	}
	if result.Checks["2"] != false {
		t.Errorf("Did not make check 2 correctly")
	}
	if result.Checks["3"] != false {
		t.Errorf("Did not make check 3 correctly")
	}
}

func TestCheckMetrics3(t *testing.T) {
	hc := initChecker(t)
	initMetricsJson()
	hc.getMetrics()
	m := metricThresholds{
		checks: map[string]string{
			"1": "testGauge2.Value < testGauge3.Value",
			"2": "testGauge2.Value == testGauge4.Value ",
			"3": "testGauge4.Value >= testGauge3.Value",
		},
	}
	result := hc.checkMetric(m)
	if result.Checks["1"] != true {
		t.Errorf("Did not make check 1 correctly")
	}
	if result.Checks["2"] != false {
		t.Errorf("Did not make check 2 correctly")
	}
	if result.Checks["3"] != true {
		t.Errorf("Did not make check 3 correctly")
	}
}

func TestCheckConstants1(t *testing.T) {
	hc := initChecker(t)
	hc.c = conf.NewConfigFile()
	hc.c.AddSection("constants")
	hc.c.AddOption("constants", "const1", "1")

	m := metricThresholds{
		checks: map[string]string{
			"1": "const1 == 1",
			"2": "const1 != 1",
			"3": "const1 >= 0",
		},
	}

	hc.setupConstants()
	result := hc.checkMetric(m)

	if result.Checks["1"] != true {
		t.Errorf("Did not make check 1 correctly")
	}
	if result.Checks["2"] != false {
		t.Errorf("Did not make check 2 correctly")
	}
	if result.Checks["3"] != true {
		t.Errorf("Did not make check 3 correctly")
	}
}

func TestCheckConstants2(t *testing.T) {
	hc := initChecker(t)
	hc.c = conf.NewConfigFile()
	hc.c.AddSection("constants")
	hc.c.AddOption("constants", "const1", "1")
	hc.c.AddOption("constants", "const2", "2")

	m := metricThresholds{
		checks: map[string]string{
			"1": "const1 == const2",
			"2": "const1 != const2",
			"3": "const1 >= const2",
		},
	}

	hc.setupConstants()
	result := hc.checkMetric(m)

	if result.Checks["1"] != false {
		t.Errorf("Did not make check 1 correctly")
	}
	if result.Checks["2"] != true {
		t.Errorf("Did not make check 2 correctly")
	}
	if result.Checks["3"] != false {
		t.Errorf("Did not make check 3 correctly")
	}
}

func TestReadConfigFile(t *testing.T) {
	c := initConfigFile()
	m1 := getConfigChecks(c, "section1")
	if m1.checks["check1"] != "value1" {
		t.Error("did not get section1, check1 correct")
	}
	if m1.checks["check2"] != "value2" {
		t.Error("did not get section1, check2 correct")
	}
	m2 := getConfigChecks(c, "section2")
	if m2.checks["check1"] != "valueA" {
		t.Error("did not get section2, check1 correct")
	}
	if m2.checks["check2"] != "valueB" {
		t.Error("did not get section2, check2 correct")
	}
	if m2.checks["check3"] != "valueC" {
		t.Error("did not get section2, check3 correct")
	}
}
