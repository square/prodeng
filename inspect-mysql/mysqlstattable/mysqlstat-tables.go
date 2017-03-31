// Copyright (c) 2014 Square, Inc
//

package mysqlstattable

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/square/prodeng/inspect-mysql/mysqltools"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
)

const (
	innodbMetadataCheck = "SELECT @@GLOBAL.innodb_stats_on_metadata;"
	dbSizesQuery        = `
  SELECT table_schema AS db,
         SUM( data_length + index_length ) AS db_size_bytes
    FROM information_schema.TABLES
   WHERE table_schema NOT IN ('performance_schema', 'information_schema', 'mysql')
   GROUP BY 1;`
	tblSizesQuery = `
    SELECT table_schema AS db, table_name as tbl,
           data_length + index_length AS tbl_size_bytes
      FROM information_schema.TABLES
     WHERE table_schema NOT IN ('performance_schema', 'information_schema', 'mysql');`
	tblStatisticsQuery = `
SELECT table_schema AS db, table_name AS tbl, 
       rows_read, rows_changed, rows_changed_x_indexes  
  FROM INFORMATION_SCHEMA.TABLE_STATISTICS
 WHERE rows_read > 0;`
)

// MysqlStatTables - main struct that contains connection to database, metric context, and map to database stats struct
type MysqlStatTables struct {
	DBs   map[string]*DBStats
	m     *metrics.MetricContext
	db    mysqltools.MysqlDB
	nLock *sync.Mutex
}

//database stats struct
//contains metrics for databases and map to tables stats struct
type DBStats struct {
	Tables  map[string]*MysqlStatPerTable
	Metrics *MysqlStatPerDB
}

//  MysqlStatPerTable - metrics for each table
type MysqlStatPerTable struct {
	SizeBytes           *metrics.Gauge
	RowsRead            *metrics.Counter
	RowsChanged         *metrics.Counter
	RowsChangedXIndexes *metrics.Counter
}

// MysqlStatPerDB - metrics for each database
type MysqlStatPerDB struct {
	SizeBytes *metrics.Gauge
}

//initializes mysqlstat
// starts off collect
func New(m *metrics.MetricContext, Step time.Duration, user, password, config string) (*MysqlStatTables, error) {
	s := new(MysqlStatTables)
	s.m = m
	s.nLock = &sync.Mutex{}
	// connect to database
	var err error
	s.db, err = mysqltools.New(user, password, config)
	s.nLock.Lock()
	s.DBs = make(map[string]*DBStats)
	s.nLock.Unlock()
	if err != nil { //error in connecting to database
		return nil, err
	}

	s.Collect()

	ticker := time.NewTicker(Step)
	go func() {
		for _ = range ticker.C {
			go s.Collect()
		}
	}()
	return s, nil
}

//initialize  per database metrics
func newMysqlStatPerDB(m *metrics.MetricContext, dbname string) *MysqlStatPerDB {
	o := new(MysqlStatPerDB)
	misc.InitializeMetrics(o, m, "mysqldbstat."+dbname, true)
	return o
}

//initialize per table metrics
func newMysqlStatPerTable(m *metrics.MetricContext, dbname, tblname string) *MysqlStatPerTable {
	o := new(MysqlStatPerTable)

	misc.InitializeMetrics(o, m, "mysqltablestat."+dbname+"."+tblname, true)
	return o
}

//collects metrics.
// sql.DB is thread safe so launching metrics collectors
// in their own goroutines is safe
func (s *MysqlStatTables) Collect() {
	go s.getDBSizes()
	go s.getTableSizes()
	go s.getTableStatistics()
}

//instantiate database metrics struct
func (s *MysqlStatTables) initializeDB(dbname string) *DBStats {
	n := new(DBStats)
	n.Metrics = newMysqlStatPerDB(s.m, dbname)
	n.Tables = make(map[string]*MysqlStatPerTable)
	return n
}

//check if database struct is instantiated, and instantiate if not
func (s *MysqlStatTables) checkDB(dbname string) {
	s.nLock.Lock()
	if _, ok := s.DBs[dbname]; !ok {
		s.DBs[dbname] = s.initializeDB(dbname)
	}
	s.nLock.Unlock()
	return
}

//check if table struct is instantiated, and instantiate if not
func (s *MysqlStatTables) checkTable(dbname, tblname string) {
	s.checkDB(dbname)
	s.nLock.Lock()
	if _, ok := s.DBs[dbname].Tables[tblname]; !ok {
		s.DBs[dbname].Tables[tblname] = newMysqlStatPerTable(s.m, dbname, tblname)
	}
	s.nLock.Unlock()
	return
}

//gets sizes of databases
func (s *MysqlStatTables) getDBSizes() {
	res, err := s.db.QueryReturnColumnDict(innodbMetadataCheck)
	if err != nil {
		s.db.Log(err)
		return
	}
	for _, val := range res {
		if v, _ := strconv.ParseInt(string(val[0]), 10, 64); v == 1 {
			fmt.Println("Not capturing db/tbl sizes because @@GLOBAL.innodb_stats_on_metadata = 1")
			s.db.Log(errors.New("not capturing sizes: innodb_stats_on_metadata = 1"))
			return
		}
		break
	}

	res, err = s.db.QueryMapFirstColumnToRow(dbSizesQuery)
	if err != nil {
		s.db.Log(err)
		return
	}
	for key, value := range res {
		//key being the name of the database, value being its size in bytes
		dbname := string(key)
		size, _ := strconv.ParseInt(string(value[0]), 10, 64)
		if size > 0 {
			s.checkDB(dbname)
			s.nLock.Lock()
			s.DBs[dbname].Metrics.SizeBytes.Set(float64(size))
			s.nLock.Unlock()
		}
	}
	return
}

//gets sizes of tables within databases
func (s *MysqlStatTables) getTableSizes() {
	res, err := s.db.QueryReturnColumnDict(innodbMetadataCheck)
	if err != nil {
		s.db.Log(err)
		return
	}
	for _, val := range res {
		if v, _ := strconv.ParseInt(string(val[0]), 10, 64); v == int64(1) {
			fmt.Println("Not capturing db/tbl sizes because @@GLOBAL.innodb_stats_on_metadata = 1")
			s.db.Log(errors.New("not capturing sizes: innodb_stats_on_metadata = 1"))
			return
		}
		break
	}
	res, err = s.db.QueryReturnColumnDict(tblSizesQuery)
	if err != nil {
		s.db.Log(err)
		return
	}
	tbl_count := len(res["tbl"])
	for i := 0; i < tbl_count; i++ {
		dbname := string(res["db"][i])
		tblname := string(res["tbl"][i])
		if res["tbl_size_bytes"][i] == "" {
			continue
		}
		s.checkDB(dbname)
		size, err := strconv.ParseInt(string(res["tbl_size_bytes"][i]), 10, 64)
		if err != nil {
			s.db.Log(err)
		}
		if size > 0 {
			s.checkTable(dbname, tblname)
			s.nLock.Lock()
			s.DBs[dbname].Tables[tblname].SizeBytes.Set(float64(size))
			s.nLock.Unlock()
		}
	}
	return
}

//get table statistics: rows read, rows changed, rows changed x indices
func (s *MysqlStatTables) getTableStatistics() {
	res, err := s.db.QueryReturnColumnDict(tblStatisticsQuery)
	if len(res) == 0 || err != nil {
		s.db.Log(err)
		return
	}
	for i, tblname := range res["tbl"] {
		dbname := res["db"][i]
		rows_read, err := strconv.ParseInt(res["rows_read"][i], 10, 64)
		if err != nil {
			s.db.Log(err)
		}
		rows_changed, err := strconv.ParseInt(res["rows_changed"][i], 10, 64)
		if err != nil {
			s.db.Log(err)
		}
		rows_changed_x_indexes, err := strconv.ParseInt(res["rows_changed_x_indexes"][i], 10, 64)
		if err != nil {
			s.db.Log(err)
		}
		if rows_read > 0 {
			s.checkDB(dbname)
			s.checkTable(dbname, tblname)
			s.nLock.Lock()
			s.DBs[dbname].Tables[tblname].RowsRead.Set(uint64(rows_read))
			s.nLock.Unlock()
		}
		if rows_changed > 0 {
			s.checkDB(dbname)
			s.checkTable(dbname, tblname)
			s.nLock.Lock()
			s.DBs[dbname].Tables[tblname].RowsChanged.Set(uint64(rows_changed))
			s.nLock.Unlock()
		}
		if rows_changed_x_indexes > 0 {
			s.checkDB(dbname)
			s.checkTable(dbname, tblname)
			s.nLock.Lock()
			s.DBs[dbname].Tables[tblname].RowsChangedXIndexes.Set(uint64(rows_changed_x_indexes))
			s.nLock.Unlock()
		}
	}
	return
}

//Closes connection with database
func (s *MysqlStatTables) Close() {
	s.db.Close()
}
