###### Example API use

```go
// Initialize a metric context
m := metrics.NewMetricContext("system")

// Create a new counter
// Add/Set operations are atomic
// No locks are held for counter operations
c := metrics.NewCounter(m)

c.Add(n)    // increment counter by delta n
c.Set(n)    // Set counter value to n

r := c.ComputeRate() // compute rate of change/sec

// Create a new gauge
// Set/Get acquire a mutex
c := metrics.NewGauge(m)
c.Set(12.0) // Set Value
c.Get() // get Value

// StatsTimer for measuring things like latencies

s := metrics.NewStatsTimer(m)

t := s.Start() // returns a timer
s.Stop(t) // stop the timer

// Example
func (* Webapp) ServeRequest(uri string) error {
	t := s.Start()

	// do something
	s.Stop(t)
}

fmt.Println("Percentile latency for 75 pctile: ", s.Percentile(75))
```
