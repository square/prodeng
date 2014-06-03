// Copyright (c) 2014 Square, Inc
//

package mysqlstat

import (
	"errors"
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
	SlaveSecondsBehindMaster      *metrics.Gauge
	SlaveSeqFile                  *metrics.Gauge
	SlavePosition                 *metrics.Counter
	ThreadsRunning                *metrics.Gauge
	ThreadsConnected              *metrics.Gauge
	UptimeSinceFlushStatus        *metrics.Counter
	OpenTables                    *metrics.Gauge
	Uptime                        *metrics.Counter
	InnodbRowLockCurrentWaits     *metrics.Gauge
	InnodbCurrentRowLocks         *metrics.Gauge
	InnodbRowLockTimeAvg          *metrics.Gauge
	InnodbRowLockTimeMax          *metrics.Counter
	InnodbLogOsWaits              *metrics.Gauge
	ComSelect                     *metrics.Counter
	Queries                       *metrics.Counter
	BinlogSeqFile                 *metrics.Gauge
	BinlogPosition                *metrics.Counter
	IdenticalQueriesStacked       *metrics.Gauge
	IdenticalQueriesMaxAge        *metrics.Gauge
	MaxConnections                *metrics.Gauge
	CurrentSessions               *metrics.Gauge
	CurrentConnectionsPct         *metrics.Gauge
	ActiveSessions                *metrics.Gauge
	UnauthenticatedSessions       *metrics.Gauge
	LockedSessions                *metrics.Gauge
	TablesLocks                   *metrics.Gauge
	GlobalReadLocks               *metrics.Gauge
	CopyingToTable                *metrics.Gauge
	Statistics                    *metrics.Gauge
	ComCreateTable                *metrics.Counter
	ComAlterTable                 *metrics.Counter
	ComDropTable                  *metrics.Counter
	ComBegin                      *metrics.Counter
	ComCommit                     *metrics.Counter
	ComRollback                   *metrics.Counter
	ComInsert                     *metrics.Counter
	ComInsertSelect               *metrics.Counter
	ComUpdate                     *metrics.Counter
	ComUpdateMulti                *metrics.Counter
	ComDelete                     *metrics.Counter
	ComDeleteMulti                *metrics.Counter
	ComReplace                    *metrics.Counter
	ComReplaceSelect              *metrics.Counter
	BinlogCacheUse                *metrics.Counter
	BinlogCacheDiskUse            *metrics.Counter
	SortMergePasses               *metrics.Counter
	CreatedTmpDiskTables          *metrics.Counter
	CreatedTmpFiles               *metrics.Counter
	CreatedTmpTables              *metrics.Counter
	TransactionID                 *metrics.Gauge
	InnodbTransactionsNotStarted  *metrics.Gauge
	InnodbHistoryLinkList         *metrics.Gauge
	InnodbUndo                    *metrics.Counter
	OSFileReads                   *metrics.Gauge
	OSFileWrites                  *metrics.Gauge
	ASFsynce                      *metrics.Gauge
	ReadsPerSec                   *metrics.Gauge
	AvgBytesPerRead               *metrics.Gauge
	WritesPerSec                  *metrics.Gauge
	FsyncsPerSec                  *metrics.Gauge
	InnodbLogSequenceNumber       *metrics.Counter
	InnodbLastCheckpointAt        *metrics.Gauge
	InnodbMaxCheckpointAge        *metrics.Gauge
	InnodbModifiedAge             *metrics.Gauge
	InnodbCheckpointAge           *metrics.Gauge
	InnodbPendingLogWrites        *metrics.Gauge
	InnodbLogFlushedUpTo          *metrics.Gauge
	PagesFlushedUpTo              *metrics.Gauge
	InnodbCheckpointAgeTarget     *metrics.Gauge
	InnodbPendingCheckpointWrites *metrics.Gauge
	TotalMemByReadViews           *metrics.Gauge
	PageHash                      *metrics.Gauge
	FileSystem                    *metrics.Gauge
	LockSystem                    *metrics.Gauge
	RecoverySystem                *metrics.Gauge
	DatabasePages                 *metrics.Gauge
	PendingReads                  *metrics.Gauge
	BufferPoolSize                *metrics.Gauge
	TotalMem                      *metrics.Gauge
	AdaptiveHash                  *metrics.Gauge
	DictionaryCache               *metrics.Gauge
	OldDatabasePages              *metrics.Gauge
	PagesRead                     *metrics.Gauge
	DictionaryMemoryAllocated     *metrics.Gauge
	FreeBuffers                   *metrics.Gauge
	ModifiedDBPages               *metrics.Gauge
	LRU                           *metrics.Gauge
	PagesMadeYoung                *metrics.Gauge
	BufferPoolHitRate             *metrics.Gauge
	Version                       *metrics.Gauge
	ActiveLongRunQueries          *metrics.Gauge
	BinlogFiles                   *metrics.Gauge
	BinlogSize                    *metrics.Gauge
	CacheHitPct                   *metrics.Gauge
	InnodbLogWriteRatio           *metrics.Gauge
	LogIOPerSec                   *metrics.Gauge
	BusySessionPct                *metrics.Gauge
	InnodbBufpoolLRUMutexOSWait   *metrics.Counter
	InnodbBufpoolZipMutexOSWait   *metrics.Counter
	OldestQuery                   *metrics.Gauge
	//Query response time metrics
}

//initializes mysqlstat
// starts off collect
func New(m *metrics.MetricContext, Step time.Duration, user, password, config string) (*MysqlStat, error) {
	s := new(MysqlStat)

	// connect to database
	var err error
	s.db, err = mysqltools.New(user, password, config)
	if err != nil {
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
	c := new(MysqlStatMetrics)
	misc.InitializeMetrics(c, m, "mysqlstat", true)
	return c
}

func (s *MysqlStat) Collect() {
	s.getVersion()
	s.getSlaveStats()
	s.getGlobalStatus()
	s.getBinlogStats()
	s.getStackedQueries()
	s.getSessions()
	s.getInnodbStats()
	s.getNumLongRunQueries()
	s.getInnondbBufferpoolMutexWaits()
}

// get_slave_stats gets slave statistics
func (s *MysqlStat) getSlaveStats() error {
	res, _ := s.db.QueryReturnColumnDict("SHOW SLAVE STATUS;")

	if len(res["Seconds_Behind_Master"]) > 0 {
		seconds_behind_master, _ := strconv.ParseFloat(string(res["Seconds_Behind_Master"][0]), 64)
		s.Metrics.SlaveSecondsBehindMaster.Set(float64(seconds_behind_master))
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
	res, _ := s.db.QueryMapFirstColumnToRow("SHOW GLOBAL STATUS;")
	vars := map[string]interface{}{"Threads_running": s.Metrics.ThreadsRunning,
		"Threads_connected":             s.Metrics.ThreadsConnected,
		"Uptime":                        s.Metrics.Uptime,
		"Innodb_row_lock_current_waits": s.Metrics.InnodbRowLockCurrentWaits,
		"Innodb_current_row_locks":      s.Metrics.InnodbCurrentRowLocks,
		"Innodb_row_lock_time_avg":      s.Metrics.InnodbRowLockTimeAvg,
		"Innodb_row_lock_time_max":      s.Metrics.InnodbRowLockTimeMax,
		"Queries":                       s.Metrics.Queries,
		"Innodb_log_os_waits":           s.Metrics.InnodbLogOsWaits,
		"Com_select":                    s.Metrics.ComSelect,
		"Com_create_table":              s.Metrics.ComCreateTable,
		"Com_alter_table":               s.Metrics.ComAlterTable,
		"Com_drop_table":                s.Metrics.ComDropTable,
		"Com_begin":                     s.Metrics.ComBegin,
		"Com_commit":                    s.Metrics.ComCommit,
		"Com_rollback":                  s.Metrics.ComRollback,
		"Com_insert":                    s.Metrics.ComInsert,
		"Com_insert_select":             s.Metrics.ComInsertSelect,
		"Com_update":                    s.Metrics.ComUpdate,
		"Com_update_multi":              s.Metrics.ComUpdateMulti,
		"Com_delete":                    s.Metrics.ComDelete,
		"Com_delete_multi":              s.Metrics.ComDeleteMulti,
		"Com_replace":                   s.Metrics.ComReplace,
		"Com_replace_select":            s.Metrics.ComReplaceSelect,
		"Binlog_cache_use":              s.Metrics.BinlogCacheUse,
		"Binlog_cache_disk_use":         s.Metrics.BinlogCacheDiskUse,
		"Sort_merge_passes":             s.Metrics.SortMergePasses,
		"Created_tmp_disk_tables":       s.Metrics.CreatedTmpDiskTables,
		"Created_tmp_files":             s.Metrics.CreatedTmpFiles,
		"Created_tmp_tables":            s.Metrics.CreatedTmpTables}

	//range through expected metrics and grab from data
	for name, metric := range vars {
		v, ok := res[name]
		if ok && len(v) > 0 {
			val, _ := strconv.ParseFloat(string(v[0]), 64)
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

func (s *MysqlStat) getInnondbBufferpoolMutexWaits() error {
	//TODO: make this collect less frequently as this operation can be expensive
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
			if !strings.Contains(name, "os_waits=") {
				return errors.New("mutex status did not contain 'os_waits='")
			}
			os_waits, err := strconv.Atoi(status[9:])
			if err != nil {
				return nil
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
		t, _ = strconv.Atoi(time[0])
	}
	s.Metrics.OldestQuery.Set(float64(t))
	return nil
}

//calculate query response times
//TODO: finish
func (s *MysqlStat) getQueryResponseTime() error {
	res, err := s.db.QueryReturnColumnDict("SELECT time, count FROM INFORMATION_SCHEMA.QUERY_RESPONSE_TIME;")
	if err != nil {
		return err
	}

	for i, time := range res["time"] {
		count, _ := strconv.Atoi(res["count"][i])
		if count < 1 {
			continue
		}
		key := "query_response_secs." + strings.Replace(strings.Trim(time, " 0"), ".", "_", -1)
		//TODO: finish here
		return errors.New(key)
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
		s, _ := strconv.Atoi(size)
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
	res, _ := s.db.QueryReturnColumnDict(cmd)
	found_sql := len(res["ID"])
	s.Metrics.ActiveLongRunQueries.Set(float64(found_sql))
	return nil
}

//get version
//version is of the form '1.2.34-56.7' or '9.8.76a-54.3-log'
// want to represent version in form '1.234567' or '9.876543'
func (s *MysqlStat) getVersion() error {
	res, _ := s.db.QueryReturnColumnDict("SELECT VERSION();")
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
	ver, _ := strconv.ParseFloat(version, 64)
	ver /= math.Pow(10.0, (float64(len(version)) - leading))
	s.Metrics.Version.Set(ver)
	return nil
}

// get binlog statistics
func (s *MysqlStat) getBinlogStats() error {
	res, _ := s.db.QueryReturnColumnDict("SHOW MASTER STATUS;")
	if len(res["File"]) == 0 || len(res["Position"]) == 0 {
		return nil
	}

	v, _ := strconv.ParseFloat(strings.Split(string(res["File"][0]), ".")[1], 64)
	s.Metrics.BinlogSeqFile.Set(float64(v))
	v, _ = strconv.ParseFloat(string(res["Position"][0]), 64)
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
	res, _ := s.db.QueryReturnColumnDict(cmd)
	if len(res) > 0 && len(res["identical_queries_stacked"]) > 0 {
		count, _ := strconv.ParseFloat(string(res["identical_queries_stacked"][0]), 64)
		s.Metrics.IdenticalQueriesStacked.Set(float64(count))
		age, _ := strconv.ParseFloat(string(res["max_age"][0]), 64)
		s.Metrics.IdenticalQueriesMaxAge.Set(float64(age))
	}
	return nil
}

//get session stats
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
func (s *MysqlStat) getInnodbStats() {
	res, err := s.db.QueryReturnColumnDict("SHOW GLOBAL VARIABLES LIKE 'innodb_log_file_size';")
	var innodb_log_file_size int
	if err == nil && len(res["Value"]) > 0 {
		innodb_log_file_size, _ = strconv.Atoi(res["Value"][0])
	}
	res, _ = s.db.QueryReturnColumnDict("SHOW ENGINE INNODB STATUS")
	var idb *mysqltools.InnodbStats
	for _, val := range res {
		idb, _ = mysqltools.ParseInnodbStats(val[0])
	}
	vars := map[string]interface{}{
		"trx_id":                      s.Metrics.TransactionID,
		"trxes_not_started":           s.Metrics.InnodbTransactionsNotStarted,
		"history_list":                s.Metrics.InnodbHistoryLinkList,
		"undo":                        s.Metrics.InnodbUndo,
		"OS_file_reads":               s.Metrics.OSFileReads,
		"OS_file_writes":              s.Metrics.OSFileWrites,
		"reads_per_s":                 s.Metrics.ReadsPerSec,
		"avg_bytes_per_read":          s.Metrics.AvgBytesPerRead,
		"writes_per_s":                s.Metrics.WritesPerSec,
		"fsyncs_per_s":                s.Metrics.FsyncsPerSec,
		"log_sequence_number":         s.Metrics.InnodbLogSequenceNumber,
		"last_checkpoint_at":          s.Metrics.InnodbLastCheckpointAt,
		"max_checkpoint_age":          s.Metrics.InnodbMaxCheckpointAge,
		"modified_age":                s.Metrics.InnodbModifiedAge,
		"checkpoint_age":              s.Metrics.InnodbCheckpointAge,
		"pending_log_writes":          s.Metrics.InnodbPendingLogWrites,
		"log_flushed_up_to":           s.Metrics.InnodbLogFlushedUpTo,
		"pages_flushed_up_to":         s.Metrics.PagesFlushedUpTo,
		"checkpoint_age_target":       s.Metrics.InnodbCheckpointAgeTarget,
		"pending_chkp_writes":         s.Metrics.InnodbPendingCheckpointWrites,
		"total_mem_by_read_views":     s.Metrics.TotalMemByReadViews,
		"page_hash":                   s.Metrics.PageHash,
		"file_system":                 s.Metrics.FileSystem,
		"lock_system":                 s.Metrics.LockSystem,
		"recovery_system":             s.Metrics.RecoverySystem,
		"database_pages":              s.Metrics.DatabasePages,
		"pending_reads":               s.Metrics.PendingReads,
		"buffer_pool_size":            s.Metrics.BufferPoolSize,
		"total_mem":                   s.Metrics.TotalMem,
		"adaptive_hash":               s.Metrics.AdaptiveHash,
		"dictionary_cache":            s.Metrics.DictionaryCache,
		"old_database_pages":          s.Metrics.OldDatabasePages,
		"pages_read":                  s.Metrics.PagesRead,
		"dictionary_memory_allocated": s.Metrics.DictionaryMemoryAllocated,
		"free_buffers":                s.Metrics.FreeBuffers,
		"modified_db_pages":           s.Metrics.ModifiedDBPages,
		"pending_writes_lru":          s.Metrics.LRU,
		"pages_made_young":            s.Metrics.PagesMadeYoung,
		"buffer_pool_hit_rate":        s.Metrics.BufferPoolHitRate,
		"cache_hit_pct":               s.Metrics.CacheHitPct,
		"log_io_per_sec":              s.Metrics.LogIOPerSec,
	}
	for name, metric := range vars {
		v, ok := idb.Metrics[name]
		if ok {
			val, _ := strconv.ParseFloat(string(v), 64)
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
	return
}

func (s *MysqlStat) Queries() uint64 {
	return s.Metrics.Queries.Get()
}

func (s *MysqlStat) Uptime() uint64 {
	return s.Metrics.Uptime.Get()
}
