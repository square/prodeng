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

// StatsTimer - useful for computing statistics on timed operations
s := metrics.NewStatsTimer(m)

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
```
