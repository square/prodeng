#### inspect

inspect is a collection of libraries for gathering
system metrics.

inspect command line is an utility that gives a
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
total: cpu: 10.2%, mem: 6.5% (248.16MB/3.74GB)
disk: sr0 usage: NaN
disk: sda usage: 0.000000
disk: sdb usage: 0.000000
cgroup:CG1 cpu_throttling: 18.1% (0.1/1) mem: 0.1% (316.00KB/409.60MB)
Top processes by CPU usage:
usage: 10.1, command: (perl)
usage: 1.0, command: (inspect)
usage: 0.0, command: (zsh)
usage: 0.0, command: (abrt-dump-oops)
usage: 0.0, command: (zsh)
---
Top processes by Mem usage:
usage: 8.10MB, command: (zsh)
usage: 7.67MB, command: (zsh)
usage: 7.43MB, command: (zsh)
usage: 7.38MB, command: (zsh)
usage: 7.33MB, command: (zsh)
```
