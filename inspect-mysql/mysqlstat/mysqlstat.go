// Copyright (c) 2014 Square, Inc
//
// Must download driver for mysql use. Run the following command:
//      go get github.com/go-sql-driver/mysql
// in order to successfully build/install

package mysqlstat

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/square/prodeng/inspect-mysql/mysqltools"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
)

type MysqlStat struct {
	Metrics *MysqlStatMetrics
	m       *metrics.MetricContext
	db      *mysqltools.MysqlDB
}

//metrics being collected about the server/database
type MysqlStatMetrics struct {
	SecondsBehindMaster       *metrics.Gauge
	SlaveSeqFile              *metrics.Gauge
	SlavePosition             *metrics.Gauge
	ThreadsRunning            *metrics.Counter
	ThreadsConnected          *metrics.Counter
	UptimeSinceFlushStatus    *metrics.Counter
	OpenTables                *metrics.Counter
	Uptime                    *metrics.Counter
	InnodbRowLockCurrentWaits *metrics.Counter
	InnodbCurrentRowLocks     *metrics.Counter
	InnodbRowLockTimeAvg      *metrics.Counter
	InnodbRowLockTimeMax      *metrics.Counter
	InnodbLogOsWaits          *metrics.Counter
	ComSelect                 *metrics.Counter
	Queries                   *metrics.Counter
	BinlogSeqFile             *metrics.Counter
	BinlogPosition            *metrics.Counter
	IdenticalQueriesStacked   *metrics.Counter
	IdenticalQueriesMaxAge    *metrics.Counter
}

//initializes mysqlstat
// starts off collect
func New(m *metrics.MetricContext, Step time.Duration, user string, password string) (*MysqlStat, error) {
	s := new(MysqlStat)

	// connect to database
	db, err := mysqltools.New(user, password)
	s.db = db
	if err != nil { //error in connecting to database
		return nil, err
	}
	s.Metrics = MysqlStatMetricsNew(m, Step)

	defer s.db.Close()
	s.Collect()

	ticker := time.NewTicker(Step)
	func() {
		for _ = range ticker.C {
			s.Collect()
		}
	}()
	return s, nil
}

//initializes metrics
func MysqlStatMetricsNew(m *metrics.MetricContext, Step time.Duration) *MysqlStatMetrics {
	//fmt.Println("starting")
	c := new(MysqlStatMetrics)
	misc.InitializeMetrics(c, m, "mysqlstat") //, true)
	return c
}

func (s *MysqlStat) Collect() {
	s.get_slave_stats()
	s.get_global_status()
}

// get_slave_stats gets slave statistics
func (s *MysqlStat) get_slave_stats() error {
	res, _ := s.db.Query_return_columns_dict("SHOW SLAVE STATUS;")
	//	fmt.Println("Result when querying 'SHOW SLAVE STATUS;'")
	//	fmt.Println(res)

	if len(res["Seconds_Behind_Master"]) > 0 {
		seconds_behind_master, _ := strconv.Atoi(string(res["Seconds_Behind_Master"][0]))
		s.Metrics.SecondsBehindMaster.Set(float64(seconds_behind_master))
	}

	relay_master_log_file, _ := res["Relay_Master_Log_File"]
	if len(relay_master_log_file) > 0 {
		slave_seqfile, err := strconv.Atoi(strings.Split(string(relay_master_log_file[0]), ".")[1])
		s.Metrics.SlaveSeqFile.Set(float64(slave_seqfile))

		if err != nil {
			return err
		}
	}

	if len(res["Exec_Master_Log_Pos"]) > 0 {
		slave_position, err := strconv.Atoi(string(res["Exec_Master_Log_Pos"][0]))
		if err != nil {
			return err
		}
		s.Metrics.SlavePosition.Set(float64(slave_position))
	}
	return nil
}

//gets global statuses
func (s *MysqlStat) get_global_status() error {
	res, _ := s.db.Query_map_first_column_to_row("SHOW GLOBAL STATUS;")
	vars := map[string]*metrics.Counter{"Threads_running": s.Metrics.ThreadsRunning,
		"Threads_connected":             s.Metrics.ThreadsConnected,
		"Uptime":                        s.Metrics.Uptime,
		"Innodb_row_lock_current_waits": s.Metrics.InnodbRowLockCurrentWaits,
		"Innodb_current_row_locks":      s.Metrics.InnodbCurrentRowLocks,
		"Innodb_row_lock_time_avg":      s.Metrics.InnodbRowLockTimeAvg,
		"Innodb_row_lock_time_max":      s.Metrics.InnodbRowLockTimeMax,
		"Queries":                       s.Metrics.Queries}

	//range through expected metrics and grab from data
	for name, metric := range vars {
		v, ok := res[name]
		if ok && len(v) > 0 {
			fmt.Println(name + ": " + string(v[0]))
			val, _ := strconv.Atoi(string(v[0]))
			metric.Set(uint64(val))
		} else {
			fmt.Println("cannot find " + name)
		}
	}
	return nil
}

// get binlog statistics
func (s *MysqlStat) get_binlog_stats() error {
	res, _ := s.db.Query_return_columns_dict("SHOW MASTER STATUS;")
	v, _ := strconv.Atoi(strings.Split(string(res["File"][0]), ".")[1])
	s.Metrics.BinlogSeqFile.Set(uint64(v))
	v, _ = strconv.Atoi(string(res["Position"][0]))
	s.Metrics.BinlogPosition.Set(uint64(v))
	return nil
}

//detect application bugs which result in multiple instance of the same
// query "stacking up"/ executing at the same time
func (s *MysqlStat) get_stacked_queries() error {
	cmd := `
  SELECT COUNT(*) AS identical_queries_stacked, 
         MAX(time) AS max_age, 
         GROUP_CONCAT(id SEPARATOR ' ') AS thread_ids, 
         info as query 
    FROM information_schema.processlist 
   WHERE user != 'system user'
     AND user NOT LIKE 'repl%'
     AND info IS NOT NULL
   GROUP BY 4
  HAVING COUNT(*) > 1
     AND MAX(time) > 300
   ORDER BY 2 DESC;`
	res, _ := s.db.Query_return_columns_dict(cmd)
	if len(res) > 0 {
		count, _ := strconv.Atoi(string(res["identical_queries_stacked"][0]))
		s.Metrics.IdenticalQueriesStacked.Set(uint64(count))
		age, _ := strconv.Atoi(string(res["max_age"][0]))
		s.Metrics.IdenticalQueriesMaxAge.Set(uint64(age))
	}
	return nil
}
