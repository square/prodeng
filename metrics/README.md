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
```


###### TODO
1. Add support for Timers; see netflix servo for examples
