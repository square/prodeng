###### Example API use

```go
// Initialize a metric context with step 1 second and maximum
// history of 3 samples
m := metrics.NewMetricContext("system", time.Millisecond*1000*1, 3)

// Create a new counter
c := metrics.NewCounter(m)

c.V++ // increment counter
c.Inc() // increment counter
c.UpdateStats() // store current sample in history
c.Set(20) // Set counter value
c.CurRate() // calculate rate of change with respect to time

// Create a new gauge
c := metrics.NewGauge(m)
c.Set(12.0) // Set Value
c.V // get Value
```


###### TODO

1. Cleanup API - there are bunch of inconsistencies.
2. Add support for basic statistics that are cheap to
   calculate at runtime
3. Remove locks where not needed
