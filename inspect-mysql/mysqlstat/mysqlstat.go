// Copyright (c) 2014 Square, Inc
//
// Must download driver for mysql use. Run the following command:
//      go get github.com/go-sql-driver/mysql
// in order to successfully build/install

package mysqlstat

import (
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
	ThreadsRunning            *metrics.Gauge
	ThreadsConnected          *metrics.Gauge
	UptimeSinceFlushStatus    *metrics.Counter
	OpenTables                *metrics.Gauge
	Uptime                    *metrics.Counter
	InnodbRowLockCurrentWaits *metrics.Gauge
	InnodbCurrentRowLocks     *metrics.Gauge
	InnodbRowLockTimeAvg      *metrics.Gauge
	InnodbRowLockTimeMax      *metrics.Counter
	InnodbLogOsWaits          *metrics.Gauge
	ComSelect                 *metrics.Gauge
	Queries                   *metrics.Counter
	BinlogSeqFile             *metrics.Gauge
	BinlogPosition            *metrics.Gauge
	IdenticalQueriesStacked   *metrics.Gauge
	IdenticalQueriesMaxAge    *metrics.Gauge
	MaxConnections            *metrics.Gauge
	CurrentConnections        *metrics.Gauge
	CurrentConnectionsPercent *metrics.Gauge
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
	go func() {
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
	misc.InitializeMetrics(c, m, "mysqlstat", true)
	return c
}

func (s *MysqlStat) Collect() {
	s.getSlaveStats()
	s.getGlobalStatus()
	s.getBinlogStats()
	s.getStackedQueries()
	s.getSessions()
}

// get_slave_stats gets slave statistics
func (s *MysqlStat) getSlaveStats() error {
	res, _ := s.db.QueryReturnColumnDict("SHOW SLAVE STATUS;")
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
func (s *MysqlStat) getGlobalStatus() error {
	res, _ := s.db.QueryMapFirstColumnToRow("SHOW GLOBAL STATUS;")
	vars := map[string]interface{}{"Threads_running": s.Metrics.ThreadsRunning,
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
			val, _ := strconv.Atoi(string(v[0]))
			switch met := metric.(type) {
			case *metrics.Counter:
				met.Set(uint64(val))
			case *metrics.Gauge:
				met.Set(float64(val))
			}
		}
	}
	return nil
}

// get binlog statistics
func (s *MysqlStat) getBinlogStats() error {
	res, _ := s.db.QueryReturnColumnDict("SHOW MASTER STATUS;")
	if len(res["File"]) == 0 || len(res["Position"]) == 0 {
		return nil
	}

	v, _ := strconv.Atoi(strings.Split(string(res["File"][0]), ".")[1])
	s.Metrics.BinlogSeqFile.Set(float64(v))
	v, _ = strconv.Atoi(string(res["Position"][0]))
	s.Metrics.BinlogPosition.Set(float64(v))
	return nil
}

//detect application bugs which result in multiple instance of the same
// query "stacking up"/ executing at the same time
func (s *MysqlStat) getStackedQueries() error {
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
	res, _ := s.db.QueryReturnColumnDict(cmd)
	if len(res) > 0 && len(res["identical_queries_stacked"]) > 0 {
		count, _ := strconv.Atoi(string(res["identical_queries_stacked"][0]))
		s.Metrics.IdenticalQueriesStacked.Set(float64(count))
		age, _ := strconv.Atoi(string(res["max_age"][0]))
		s.Metrics.IdenticalQueriesMaxAge.Set(float64(age))
	}
	return nil
}

func (s *MysqlStat) getSessions() error {
	res, _ := s.db.QueryReturnColumnDict("SELECT @@GLOBAL.max_connections;")
	var max_sessions int
	for _, val := range res {
		max_sessions, _ = strconv.Atoi(val[0])
		s.Metrics.MaxConnections.Set(float64(max_sessions))
	}
	cmd := `
    SELECT IF(command LIKE 'Sleep',1,0) +
           IF(state LIKE '%master%' OR state LIKE '%slave%',1,0) AS sort_col,
           processlist.*
      FROM information_schema.processlist
     ORDER BY 1, time DESC;`
	res, _ = s.db.QueryReturnColumnDict(cmd)
	if len(res) == 0 || len(res["COMMAND"]) == 0 {
		return nil
	}
	current_total := len(res["COMMAND"])
	s.Metrics.CurrentConnections.Set(float64(current_total))
	pct := (float64(current_total) / float64(max_sessions)) * 100
	s.Metrics.CurrentConnectionsPercent.Set(pct)
	return nil
}

func (s *MysqlStat) Queries() uint64 {
	return s.Metrics.Queries.Get()
}

func (s *MysqlStat) Uptime() uint64 {
	return s.Metrics.Uptime.Get()
}
