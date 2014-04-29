#### inspect

inspect is a collection of libraries for gathering
system metrics.

inspect command line is a utility that gives a
brief overview on current state of system resource
usage.

inspect on linux gathers cpu,memory,io usage currently
both at system level, per pid/cgroup level for cpu/memory.

inspect aims to evolve to be an intelligent tool that
can spot problems.

examples: 

  * process X is throttled on CPU because of cgroup restrictions
  * process Y is causing IO contention
  * process X rejecting connections because TCP is out of memory
  * CPU spike at 14:00 UTC. Top users: A, B


For now it just dumps metrics like top,iostat

###### Installation

1. Get go
2. go get -v -u github.com/square/prodeng/inspect # fetches packages and builds

###### Documentation (WIP)

http://godoc.org/github.com/square/prodeng/inspect

http://godoc.org/github.com/square/prodeng/metrics


###### Usage

./bin/inspect

```
--------------------------
total: cpu: 15.8%, mem: 1.6% (4.51GB/283.79GB)
disk: sdd usage: 0.1
disk: sdk usage: 0.1
disk: sdo usage: 0.0
disk: sdi usage: 0.1
disk: sdj usage: 0.0
disk: sda usage: 0.0
disk: sdf usage: 0.0
disk: sdc usage: 2.4
disk: sdl usage: 0.0
iface: lo TX: 4.23Mb/s, RX: 4.23Mb/s
iface: em1 TX: 271.70Mb/s, RX: 116.27Mb/s
iface: em2 TX: 0.00b/s, RX: 1.46Kb/s
iface: em3 TX: NaNb/s, RX: NaNb/s
iface: em4 TX: NaNb/s, RX: NaNb/s
iface: bond0 TX: 271.70Mb/s, RX: 116.27Mb/s
cgroup:app  cpu: 20% cpu_throttling: 20% (1/24) mem: 0.0% (2.46GB/8.00GB)
Top processes by CPU usage:
usage: 394.7, command: (java)
usage: 3.0, command: (inspect)
usage: 1.0, command: (runner)
usage: 0.0, command: (svlogd)
usage: 0.0, command: (dsm_sa_datamgrd)
---
Top processes by Mem usage:
usage: 2.46GB, command: (java)
usage: 140.07MB, command: (dsm_sa_datamgrd)
usage: 28.45MB, command: (server)
usage: 28.40MB, command: (runner)
usage: 18.12MB, command: (ruby20)
```

###### Example API use 


```go
// collect CPU stats
import "github.com/square/prodeng/inspect/cpustat"
import "github.com/square/prodeng/metrics"

// Initialize a metric context with step 1 second and maximum
// history of 3 samples
m := metrics.NewMetricContext("system", time.Millisecond*1000*1, 3)
	
// Collect CPU metrics every m.Step seconds
cstat := cpustat.New(m)

// Allow two samples to be collected. Since most metrics are counters.
time.Sleep(time.Millisecond * 3)
fmt.Println(cstat.Usage())

```

###### Todo

  * Performance can be improved. PerProcessStat needs to have better heuristics
to backoff when the number of processes is > 1024
  * Add intelligence to find problems. Start with easy ones like CPU usage
  * Command line utility needs much nicer formatting and options to dig into per process/cgroup details
  * Add io metrics per process (need root priviliges)
  * Add caching support to reduce load when multiple invocations of inspect happen.
  * API to collect and expose historical/current statistics




