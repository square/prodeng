// Copyright (c) 2014 Square, Inc
//

package mysqlstat

import (
	"errors"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/square/prodeng/inspect-mysql/mysqltools"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
)

type MysqlStat struct {
	Metrics *MysqlStatMetrics //collection of metrics
	m       *metrics.MetricContext
	db      *mysqltools.MysqlDB //mysql connection
}

//metrics being collected about the server/database
type MysqlStatMetrics struct {
	ASFsynce                      *metrics.Gauge
	ActiveLongRunQueries          *metrics.Gauge
	ActiveSessions                *metrics.Gauge
	AdaptiveHash                  *metrics.Gauge
	AvgBytesPerRead               *metrics.Gauge
	BinlogCacheDiskUse            *metrics.Counter
	BinlogCacheUse                *metrics.Counter
	BinlogFiles                   *metrics.Gauge
	BinlogPosition                *metrics.Counter
	BinlogSeqFile                 *metrics.Gauge
	BinlogSize                    *metrics.Gauge
	BufferPoolHitRate             *metrics.Gauge
	BufferPoolSize                *metrics.Gauge
	BusySessionPct                *metrics.Gauge
	CacheHitPct                   *metrics.Gauge
	ComAlterTable                 *metrics.Counter
	ComBegin                      *metrics.Counter
	ComCommit                     *metrics.Counter
	ComCreateTable                *metrics.Counter
	ComDelete                     *metrics.Counter
	ComDeleteMulti                *metrics.Counter
	ComDropTable                  *metrics.Counter
	ComInsert                     *metrics.Counter
	ComInsertSelect               *metrics.Counter
	ComReplace                    *metrics.Counter
	ComReplaceSelect              *metrics.Counter
	ComRollback                   *metrics.Counter
	ComSelect                     *metrics.Counter
	ComUpdate                     *metrics.Counter
	ComUpdateMulti                *metrics.Counter
	CopyingToTable                *metrics.Gauge
	CreatedTmpDiskTables          *metrics.Counter
	CreatedTmpFiles               *metrics.Counter
	CreatedTmpTables              *metrics.Counter
	CurrentConnectionsPct         *metrics.Gauge
	CurrentSessions               *metrics.Gauge
	DatabasePages                 *metrics.Gauge
	DictionaryCache               *metrics.Gauge
	DictionaryMemoryAllocated     *metrics.Gauge
	FileSystem                    *metrics.Gauge
	FreeBuffers                   *metrics.Gauge
	FsyncsPerSec                  *metrics.Gauge
	GlobalReadLocks               *metrics.Gauge
	IdenticalQueriesMaxAge        *metrics.Gauge
	IdenticalQueriesStacked       *metrics.Gauge
	InnodbBufpoolLRUMutexOSWait   *metrics.Counter
	InnodbBufpoolZipMutexOSWait   *metrics.Counter
	InnodbCheckpointAge           *metrics.Gauge
	InnodbCheckpointAgeTarget     *metrics.Gauge
	InnodbCurrentRowLocks         *metrics.Gauge
	InnodbHistoryLinkList         *metrics.Gauge
	InnodbLastCheckpointAt        *metrics.Gauge
	InnodbLogFlushedUpTo          *metrics.Gauge
	InnodbLogOsWaits              *metrics.Gauge
	InnodbLogSequenceNumber       *metrics.Counter
	InnodbLogWriteRatio           *metrics.Gauge
	InnodbMaxCheckpointAge        *metrics.Gauge
	InnodbModifiedAge             *metrics.Gauge
	InnodbPendingCheckpointWrites *metrics.Gauge
	InnodbPendingLogWrites        *metrics.Gauge
	InnodbRowLockCurrentWaits     *metrics.Gauge
	InnodbRowLockTimeAvg          *metrics.Gauge
	InnodbRowLockTimeMax          *metrics.Counter
	InnodbTransactionsNotStarted  *metrics.Gauge
	InnodbUndo                    *metrics.Counter
	LockSystem                    *metrics.Gauge
	LockedSessions                *metrics.Gauge
	LogIOPerSec                   *metrics.Gauge
	MaxConnections                *metrics.Gauge
	ModifiedDBPages               *metrics.Gauge
	OSFileReads                   *metrics.Gauge
	OSFileWrites                  *metrics.Gauge
	OldDatabasePages              *metrics.Gauge
	OldestQuery                   *metrics.Gauge
	OpenTables                    *metrics.Gauge
	PageHash                      *metrics.Gauge
	PagesFlushedUpTo              *metrics.Gauge
	PagesMadeYoung                *metrics.Gauge
	PagesRead                     *metrics.Gauge
	PendingReads                  *metrics.Gauge
	PendingWritesLRU              *metrics.Gauge
	Queries                       *metrics.Counter
	ReadsPerSec                   *metrics.Gauge
	RecoverySystem                *metrics.Gauge
	SlavePosition                 *metrics.Counter
	SlaveSecondsBehindMaster      *metrics.Gauge
	SlaveSeqFile                  *metrics.Gauge
	SortMergePasses               *metrics.Counter
	Statistics                    *metrics.Gauge
	TablesLocks                   *metrics.Gauge
	ThreadsConnected              *metrics.Gauge
	ThreadsRunning                *metrics.Gauge
	TotalMem                      *metrics.Gauge
	TotalMemByReadViews           *metrics.Gauge
	TransactionID                 *metrics.Gauge
	UnauthenticatedSessions       *metrics.Gauge
	Uptime                        *metrics.Counter
	UptimeSinceFlushStatus        *metrics.Counter
	Version                       *metrics.Gauge
	WritesPerSec                  *metrics.Gauge
	//Query response time metrics
	QueryResponseSec_000001  *metrics.Counter
	QueryResponseSec_00001   *metrics.Counter
	QueryResponseSec_0001    *metrics.Counter
	QueryResponseSec_001     *metrics.Counter
	QueryResponseSec_01      *metrics.Counter
	QueryResponseSec_1       *metrics.Counter
	QueryResponseSec1_       *metrics.Counter
	QueryResponseSec10_      *metrics.Counter
	QueryResponseSec100_     *metrics.Counter
	QueryResponseSec1000_    *metrics.Counter
	QueryResponseSec10000_   *metrics.Counter
	QueryResponseSec100000_  *metrics.Counter
	QueryResponseSec1000000_ *metrics.Counter
}

//initializes mysqlstat
// starts off collect
func New(m *metrics.MetricContext, Step time.Duration, user, password, config string) (*MysqlStat, error) {
	s := new(MysqlStat)

	// connect to database
	var err error
	s.db, err = mysqltools.New(user, password, config)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	s.Metrics = MysqlStatMetricsNew(m, Step)

	defer s.db.Close()

	s.Collect(0)

	ticker := time.NewTicker(Step)
	go func() {
		i := 0
		for _ = range ticker.C {
			s.Collect(i)
			i = (i + 1) % (40 / int(Step.Seconds())) // set i to 0 every 40 seconds
		}
	}()
	return s, nil
}

//initializes metrics
func MysqlStatMetricsNew(m *metrics.MetricContext, Step time.Duration) *MysqlStatMetrics {
	c := new(MysqlStatMetrics)
	misc.InitializeMetrics(c, m, "mysqlstat", true)
	return c
}

func (s *MysqlStat) Collect(i int) {
	collections := []error{
		s.getVersion(),
		s.getSlaveStats(),
		s.getGlobalStatus(),
		s.getBinlogStats(),
		s.getStackedQueries(),
		s.getSessions(),
		s.getInnodbStats(),
		s.getNumLongRunQueries(),
		s.getInnodbBufferpoolMutexWaits(i),
		s.getQueryResponseTime(),
	}
	for _, err := range collections {
		if err != nil {
			log.Print(err)
		}
	}
}

// get_slave_stats gets slave statistics
func (s *MysqlStat) getSlaveStats() error {
	res, err := s.db.QueryReturnColumnDict("SHOW SLAVE STATUS;")
	if err != nil {
		return err
	}

	if len(res["Seconds_Behind_Master"]) > 0 {
		seconds_behind_master, _ := strconv.ParseFloat(string(res["Seconds_Behind_Master"][0]), 64)
		s.Metrics.SlaveSecondsBehindMaster.Set(float64(seconds_behind_master))
	}

	relay_master_log_file, _ := res["Relay_Master_Log_File"]
	if len(relay_master_log_file) > 0 {
		slave_seqfile, err := strconv.Atoi(strings.Split(string(relay_master_log_file[0]), ".")[1])
		s.Metrics.SlaveSeqFile.Set(float64(slave_seqfile))
		if err != nil {
			log.Print(err)
		}
	}

	if len(res["Exec_Master_Log_Pos"]) > 0 {
		slave_position, err := strconv.ParseFloat(string(res["Exec_Master_Log_Pos"][0]), 64)
		if err != nil {
			return err
		}
		s.Metrics.SlavePosition.Set(uint64(slave_position))
	}
	return nil
}

//gets global statuses
func (s *MysqlStat) getGlobalStatus() error {
	res, err := s.db.QueryMapFirstColumnToRow("SHOW GLOBAL STATUS;")
	if err != nil {
		return err
	}
	vars := map[string]interface{}{
		"Binlog_cache_disk_use":         s.Metrics.BinlogCacheDiskUse,
		"Binlog_cache_use":              s.Metrics.BinlogCacheUse,
		"Com_alter_table":               s.Metrics.ComAlterTable,
		"Com_begin":                     s.Metrics.ComBegin,
		"Com_commit":                    s.Metrics.ComCommit,
		"Com_create_table":              s.Metrics.ComCreateTable,
		"Com_delete":                    s.Metrics.ComDelete,
		"Com_delete_multi":              s.Metrics.ComDeleteMulti,
		"Com_drop_table":                s.Metrics.ComDropTable,
		"Com_insert":                    s.Metrics.ComInsert,
		"Com_insert_select":             s.Metrics.ComInsertSelect,
		"Com_replace":                   s.Metrics.ComReplace,
		"Com_replace_select":            s.Metrics.ComReplaceSelect,
		"Com_rollback":                  s.Metrics.ComRollback,
		"Com_select":                    s.Metrics.ComSelect,
		"Com_update":                    s.Metrics.ComUpdate,
		"Com_update_multi":              s.Metrics.ComUpdateMulti,
		"Created_tmp_disk_tables":       s.Metrics.CreatedTmpDiskTables,
		"Created_tmp_files":             s.Metrics.CreatedTmpFiles,
		"Created_tmp_tables":            s.Metrics.CreatedTmpTables,
		"Innodb_current_row_locks":      s.Metrics.InnodbCurrentRowLocks,
		"Innodb_log_os_waits":           s.Metrics.InnodbLogOsWaits,
		"Innodb_row_lock_current_waits": s.Metrics.InnodbRowLockCurrentWaits,
		"Innodb_row_lock_time_avg":      s.Metrics.InnodbRowLockTimeAvg,
		"Innodb_row_lock_time_max":      s.Metrics.InnodbRowLockTimeMax,
		"Queries":                       s.Metrics.Queries,
		"Sort_merge_passes":             s.Metrics.SortMergePasses,
		"Threads_connected":             s.Metrics.ThreadsConnected,
		"Uptime":                        s.Metrics.Uptime,
		"Threads_running":               s.Metrics.ThreadsRunning,
	}

	//range through expected metrics and grab from data
	for name, metric := range vars {
		v, ok := res[name]
		if ok && len(v) > 0 {
			val, err := strconv.ParseFloat(string(v[0]), 64)
			if err != nil {
				log.Print(err)
			}
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

func (s *MysqlStat) getInnodbBufferpoolMutexWaits(i int) error {
	//this collects less frequently as this operation can be expensive
	if i != 0 {
		return nil
	}
	res, err := s.db.QueryReturnColumnDict("SHOW ENGINE INNODB MUTEX;")
	if err != nil {
		return err
	}
	mets := map[string]*metrics.Counter{"&buf_pool->LRU_list_mutex": s.Metrics.InnodbBufpoolLRUMutexOSWait,
		"&buf_pool->zip_mutex": s.Metrics.InnodbBufpoolZipMutexOSWait}
	for i, name := range res["Name"] {
		status := res["Status"][i]
		metric, ok := mets[name]
		if ok {
			if !strings.Contains(status, "os_waits=") {
				return errors.New("mutex status did not contain 'os_waits=': " + status)
			}
			os_waits, err := strconv.Atoi(status[9:])
			if err != nil {
				return err
			}
			metric.Set(uint64(os_waits))
		}
	}
	return nil
}

//get time of oldest query in seconds
func (s *MysqlStat) getOldest() error {
	cmd := `
 SELECT time FROM information_schema.processlist
  WHERE command NOT IN ('Sleep','Connect','Binlog Dump')
  ORDER BY time DESC LIMIT 1;`

	res, err := s.db.QueryReturnColumnDict(cmd)
	if err != nil {
		return err
	}
	t := 0
	if time, ok := res["time"]; ok && len(time) > 0 {
		t, err = strconv.Atoi(time[0])
		if err != nil {
			log.Print(err)
		}
	}
	s.Metrics.OldestQuery.Set(float64(t))
	return nil
}

//calculate query response times
func (s *MysqlStat) getQueryResponseTime() error {
	timers := map[string]*metrics.Counter{
		".000001":  s.Metrics.QueryResponseSec_000001,
		".00001":   s.Metrics.QueryResponseSec_00001,
		".0001":    s.Metrics.QueryResponseSec_0001,
		".001":     s.Metrics.QueryResponseSec_001,
		".01":      s.Metrics.QueryResponseSec_01,
		".1":       s.Metrics.QueryResponseSec_1,
		"1.":       s.Metrics.QueryResponseSec1_,
		"10.":      s.Metrics.QueryResponseSec10_,
		"100.":     s.Metrics.QueryResponseSec100_,
		"1000.":    s.Metrics.QueryResponseSec1000_,
		"10000.":   s.Metrics.QueryResponseSec10000_,
		"100000.":  s.Metrics.QueryResponseSec100000_,
		"1000000.": s.Metrics.QueryResponseSec100000_,
	}

	res, err := s.db.QueryReturnColumnDict("SELECT time, count FROM INFORMATION_SCHEMA.QUERY_RESPONSE_TIME;")
	if err != nil {
		return err
	}

	for i, time := range res["time"] {
		count, err := strconv.Atoi(res["count"][i])
		if err != nil {
			log.Print(err)
		}
		if count < 1 {
			continue
		}
		key := strings.Trim(time, " 0")
		if timer, ok := timers[key]; ok {
			timer.Set(uint64(count))
		}
	}
	return nil
}

//gets status on binary logs
func (s *MysqlStat) getBinlogFiles() error {
	res, err := s.db.QueryReturnColumnDict("SHOW MASTER LOGS;")
	if err != nil {
		return err
	}
	s.Metrics.BinlogFiles.Set(float64(len(res["File_size"])))
	binlog_total_size := 0
	for _, size := range res["File_size"] {
		s, err := strconv.Atoi(size)
		if err != nil {
			log.Print(err) //don't return err so we can continue with more values
		}
		binlog_total_size += s
	}
	s.Metrics.BinlogSize.Set(float64(binlog_total_size))
	return nil
}

//get number of long running queries
func (s *MysqlStat) getNumLongRunQueries() error {
	cmd := `
    SELECT * FROM information_schema.processlist
     WHERE command NOT IN ('Sleep', 'Connect', 'Binlog Dump')
       AND time > 30;`
	res, err := s.db.QueryReturnColumnDict(cmd)
	if err != nil {
		return err
	}
	found_sql := len(res["ID"])
	s.Metrics.ActiveLongRunQueries.Set(float64(found_sql))
	return nil
}

//get version
//version is of the form '1.2.34-56.7' or '9.8.76a-54.3-log'
// want to represent version in form '1.234567' or '9.876543'
func (s *MysqlStat) getVersion() error {
	res, err := s.db.QueryReturnColumnDict("SELECT VERSION();")
	if err != nil {
		return err
	}
	version := res["VERSION()"][0]
	//filter out letters
	f := func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			return 'A'
		}
		return r
	}
	version = strings.Replace(strings.Map(f, version), "A", "", -1)
	version = strings.Replace(version, "_", ".", -1)
	leading := float64(len(strings.Split(version, ".")[0]))
	version = strings.Replace(version, ".", "", -1)
	ver, err := strconv.ParseFloat(version, 64)
	ver /= math.Pow(10.0, (float64(len(version)) - leading))
	s.Metrics.Version.Set(ver)
	return err
}

// get binlog statistics
func (s *MysqlStat) getBinlogStats() error {
	res, err := s.db.QueryReturnColumnDict("SHOW MASTER STATUS;")
	if err != nil {
		return err
	}
	if len(res["File"]) == 0 || len(res["Position"]) == 0 {
		return nil
	}

	v, err := strconv.ParseFloat(strings.Split(string(res["File"][0]), ".")[1], 64)
	if err != nil {
		log.Print(err)
	}
	s.Metrics.BinlogSeqFile.Set(float64(v))
	v, err = strconv.ParseFloat(string(res["Position"][0]), 64)
	if err != nil {
		log.Print(err)
	}
	s.Metrics.BinlogPosition.Set(uint64(v))
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
	res, err := s.db.QueryReturnColumnDict(cmd)
	if err != nil {
		return err
	}
	if len(res["identical_queries_stacked"]) > 0 {
		count, err := strconv.ParseFloat(string(res["identical_queries_stacked"][0]), 64)
		if err != nil {
			log.Print(err)
		}
		s.Metrics.IdenticalQueriesStacked.Set(float64(count))
		age, err := strconv.ParseFloat(string(res["max_age"][0]), 64)
		if err != nil {
			log.Print(err)
		}
		s.Metrics.IdenticalQueriesMaxAge.Set(float64(age))
	}
	return nil
}

//get session stats
func (s *MysqlStat) getSessions() error {
	res, err := s.db.QueryReturnColumnDict("SELECT @@GLOBAL.max_connections;")
	if err != nil {
		return err
	}
	var max_sessions int
	for _, val := range res {
		max_sessions, err = strconv.Atoi(val[0])
		if err != nil {
			log.Print(err)
		}
		s.Metrics.MaxConnections.Set(float64(max_sessions))
	}
	cmd := `
    SELECT IF(command LIKE 'Sleep',1,0) +
           IF(state LIKE '%master%' OR state LIKE '%slave%',1,0) AS sort_col,
           processlist.*
      FROM information_schema.processlist
     ORDER BY 1, time DESC;`
	res, err = s.db.QueryReturnColumnDict(cmd)
	if err != nil {
		return err
	}
	if len(res["COMMAND"]) == 0 {
		return nil
	}
	current_total := len(res["COMMAND"])
	s.Metrics.CurrentSessions.Set(float64(current_total))
	pct := (float64(current_total) / float64(max_sessions)) * 100
	s.Metrics.CurrentConnectionsPct.Set(pct)

	active := 0.0
	unauthenticated := 0
	locked := 0
	table_lock_wait := 0
	global_read_lock_wait := 0
	copy_to_table := 0
	statistics := 0
	for i, val := range res["COMMAND"] {
		if val != "Sleep" && val != "Connect" && val != "Binlog Dump" {
			active += 1
		}
		if matched, err := regexp.MatchString("unauthenticated", res["USER"][i]); err == nil && matched {
			unauthenticated += 1
		}
		if matched, err := regexp.MatchString("Locked", res["STATE"][i]); err == nil && matched {
			locked += 1
		} else if matched, err := regexp.MatchString("Table Lock", res["STATE"][i]); err == nil && matched {
			table_lock_wait += 1
		} else if matched, err := regexp.MatchString("Waiting for global read lock", res["STATE"][i]); err == nil && matched {
			global_read_lock_wait += 1
		} else if matched, err := regexp.MatchString("opy.*table", res["STATE"][i]); err == nil && matched {
			copy_to_table += 1
		} else if matched, err := regexp.MatchString("statistics", res["STATE"][i]); err == nil && matched {
			statistics += 1
		}
	}
	s.Metrics.ActiveSessions.Set(active)
	s.Metrics.BusySessionPct.Set(active / float64(current_total))
	s.Metrics.UnauthenticatedSessions.Set(float64(unauthenticated))
	s.Metrics.LockedSessions.Set(float64(locked))
	s.Metrics.TablesLocks.Set(float64(table_lock_wait))
	s.Metrics.GlobalReadLocks.Set(float64(global_read_lock_wait))
	s.Metrics.CopyingToTable.Set(float64(copy_to_table))
	s.Metrics.Statistics.Set(float64(statistics))

	return nil
}

//metrics from innodb
func (s *MysqlStat) getInnodbStats() error {
	res, err := s.db.QueryReturnColumnDict("SHOW GLOBAL VARIABLES LIKE 'innodb_log_file_size';")
	if err != nil {
		return err
	}
	var innodb_log_file_size int
	if err == nil && len(res["Value"]) > 0 {
		innodb_log_file_size, err = strconv.Atoi(res["Value"][0])
		if err != nil {
			log.Print(err)
		}
	}

	res, err = s.db.QueryReturnColumnDict("SHOW ENGINE INNODB STATUS")
	if err != nil {
		return err
	}

	//parse the result
	var idb *mysqltools.InnodbStats
	for _, val := range res {
		idb, _ = mysqltools.ParseInnodbStats(val[0])
	}
	vars := map[string]interface{}{
		"OS_file_reads":               s.Metrics.OSFileReads,
		"OS_file_writes":              s.Metrics.OSFileWrites,
		"adaptive_hash":               s.Metrics.AdaptiveHash,
		"avg_bytes_per_read":          s.Metrics.AvgBytesPerRead,
		"buffer_pool_hit_rate":        s.Metrics.BufferPoolHitRate,
		"buffer_pool_size":            s.Metrics.BufferPoolSize,
		"cache_hit_pct":               s.Metrics.CacheHitPct,
		"checkpoint_age":              s.Metrics.InnodbCheckpointAge,
		"checkpoint_age_target":       s.Metrics.InnodbCheckpointAgeTarget,
		"database_pages":              s.Metrics.DatabasePages,
		"dictionary_cache":            s.Metrics.DictionaryCache,
		"dictionary_memory_allocated": s.Metrics.DictionaryMemoryAllocated,
		"file_system":                 s.Metrics.FileSystem,
		"free_buffers":                s.Metrics.FreeBuffers,
		"fsyncs_per_s":                s.Metrics.FsyncsPerSec,
		"history_list":                s.Metrics.InnodbHistoryLinkList,
		"last_checkpoint_at":          s.Metrics.InnodbLastCheckpointAt,
		"lock_system":                 s.Metrics.LockSystem,
		"log_flushed_up_to":           s.Metrics.InnodbLogFlushedUpTo,
		"log_io_per_sec":              s.Metrics.LogIOPerSec,
		"log_sequence_number":         s.Metrics.InnodbLogSequenceNumber,
		"max_checkpoint_age":          s.Metrics.InnodbMaxCheckpointAge,
		"modified_age":                s.Metrics.InnodbModifiedAge,
		"modified_db_pages":           s.Metrics.ModifiedDBPages,
		"old_database_pages":          s.Metrics.OldDatabasePages,
		"page_hash":                   s.Metrics.PageHash,
		"pages_flushed_up_to":         s.Metrics.PagesFlushedUpTo,
		"pages_made_young":            s.Metrics.PagesMadeYoung,
		"pages_read":                  s.Metrics.PagesRead,
		"pending_chkp_writes":         s.Metrics.InnodbPendingCheckpointWrites,
		"pending_log_writes":          s.Metrics.InnodbPendingLogWrites,
		"pending_reads":               s.Metrics.PendingReads,
		"pending_writes_lru":          s.Metrics.PendingWritesLRU,
		"reads_per_s":                 s.Metrics.ReadsPerSec,
		"recovery_system":             s.Metrics.RecoverySystem,
		"total_mem":                   s.Metrics.TotalMem,
		"total_mem_by_read_views":     s.Metrics.TotalMemByReadViews,
		"trx_id":                      s.Metrics.TransactionID,
		"trxes_not_started":           s.Metrics.InnodbTransactionsNotStarted,
		"undo":                        s.Metrics.InnodbUndo,
		"writes_per_s":                s.Metrics.WritesPerSec,
	}
	//store the result in the appropriate metrics
	for name, metric := range vars {
		v, ok := idb.Metrics[name]
		if ok {
			val, err := strconv.ParseFloat(string(v), 64)
			if err != nil {
				log.Print(err)
			}
			switch met := metric.(type) {
			case *metrics.Counter:
				met.Set(uint64(val))
			case *metrics.Gauge:
				met.Set(float64(val))
			}
		}
	}
	if lsn, ok := idb.Metrics["log_sequence_number"]; ok && innodb_log_file_size != 0 {
		lsn_s, _ := strconv.ParseFloat(lsn, 64)
		s.Metrics.InnodbLogWriteRatio.Set((lsn_s * 3600.0) / float64(innodb_log_file_size))
	}
	return nil
}

func (s *MysqlStat) Queries() uint64 {
	return s.Metrics.Queries.Get()
}

func (s *MysqlStat) Uptime() uint64 {
	return s.Metrics.Uptime.Get()
}
