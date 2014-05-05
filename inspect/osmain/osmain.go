package osmain

import (
	"github.com/square/prodeng/inspect/cpustat"
	"github.com/square/prodeng/inspect/memstat"
	"github.com/square/prodeng/inspect/pidstat"
)

// these are implemented by all supported platforms
type OsIndependentStats struct {
	Cstat *cpustat.CPUStat
	Mstat *memstat.MemStat
	Procs *pidstat.ProcessStat
}
