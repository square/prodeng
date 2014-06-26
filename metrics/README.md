###### Example API use

```go
// Initialize a metric context
m := metrics.NewMetricContext("system")

// Create a new counter
// Add/Set operations are atomic
// No locks are held for counter operations
c := metrics.NewCounter()

c.Add(n)    // increment counter by delta n
c.Set(n)    // Set counter value to n

r := c.ComputeRate() // compute rate of change/sec

// Create a new gauge
// Set/Get acquire a mutex
c := metrics.NewGauge()
c.Set(12.0) // Set Value
c.Get() // get Value

// StatsTimer - useful for computing statistics on timed operations
s := metrics.NewStatsTimer()

t := s.Start() // returns a timer
s.Stop(t) // stop the timer

// Example
func (* Webapp) ServeRequest(uri string) error {
	t := s.Start()

	// do something
	s.Stop(t)
}
pctile_75th, err := s.Percentile(75)
if err == nil {
	fmt.Println("Percentile latency for 75 pctile: ", pctile_75th)
}


// Launch a goroutine to serve metrics via http json
go func() {
	http.HandleFunc("/metrics.json", m.HttpJsonHandler)
	http.ListenAndServe("localhost:12345", nil)
}

// Get metrics via http json.
resp, err := http.Get("http://localhost:12345/metrics.json/Counters|Gauges|StatsTimers")

// Get metrics via http json and filter out metrics with NaN values
resp, err := http.Get("http://localhost:12345/metrics.json/Counters|Gauges|StatsTimers?allowNaN=false")

// Get counter metrics
resp, err := http.Get("http://localhost:12345/metrics.json/Counters")

// Get gauges
resp, err := http.Get("http://localhost:12345/metrics.json/Gauges")

// Get stats timers
resp, err := http.Get("http://localhost:12345/metrics.json/StatsTimers")

// Get counters and gauges
resp, err := http.Get("http://localhost:12345/metrics.json/Gauges|Counters")

// Get gauges and limit to metrics matching memstat.Active
resp, err := http.Get("http://localhost:12345/metrics.json/Gauges/memstat.Active")

```
