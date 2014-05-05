#### inspect

inspect is a collection of libraries for gathering
system metrics.

inspect command line is a utility that gives a
brief overview on current state of system resource
usage.

Supported platforms: linux, MacOSX 10.9

inspect on linux gathers cpu,memory,io usage currently
both at system level, per pid/cgroup level for cpu/memory.

inspect on MacOSX gathers cpu,memory usage currently
both at system level, per pid. inspect needs root privileges
on MacOSX for per-pid information.

inspect aims to evolve to be an intelligent tool that
can spot problems.

examples: 

  * process X is throttled on CPU because of cgroup restrictions
  * process Y is causing IO contention
  * process X rejecting connections because TCP is out of memory
  * CPU spike at 14:00 UTC. Top users: A, B


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
total: cpu: 34.0%, mem: 92.2% (43.45GB/47.13GB)
---
Top processes by CPU usage:
cpu: 99.3%  command: (carbon-cache.py) user: apache pid: 15986
cpu: 98.9%  command: (carbon-cache.py) user: apache pid: 31571
cpu: 54.9%  command: (carbon-relay.py) user: apache pid: 1860
cpu: 52.9%  command: (carbon-relay.py) user: apache pid: 1855
cpu: 49.8%  command: (carbon-relay.py) user: apache pid: 1882
---
Top processes by Mem usage:
mem: 26.27GB command: (carbon-cache.py) user: apache pid: 15986
mem: 15.91GB command: (carbon-cache.py) user: apache pid: 31571
mem: 49.41MB command: (node) user: apache pid: 1863
mem: 48.10MB command: (node) user: apache pid: 1852
mem: 26.34MB command: (carbon-relay.py) user: apache pid: 1884
--
disk: sda usage: 0.5%
---
iface: lo TX: 22.45Mb/s, RX: 22.45Mb/s
iface: eth0 TX: 0.00b/s, RX: 0.00b/s
iface: eth1 TX: 17.07Mb/s, RX: 24.46Mb/s
iface: bond0 TX: 17.07Mb/s, RX: 24.46Mb/s
--
cgroup:syam_test cpu: 9.9% cpu_throttling: 89.7% (0.1/16) mem: 0.0% (316.00KB/1.00GB)
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




