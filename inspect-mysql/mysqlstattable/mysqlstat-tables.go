// Copyright (c) 2014 Square, Inc
//

package mysqlstattable

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/square/prodeng/inspect-mysql/mysqltools"
	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
)

// Structs:
// MysqlStatTables - main struct that contains connection to database, metric context, and map to database stats struct
type MysqlStatTables struct {
	DBs map[string]*DBStats
	m   *metrics.MetricContext
	db  *mysqltools.MysqlDB
}

//DBStats - database stats struct
//  contains metrics for databases and map to tables stats struct
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
	// connect to database
	var err error
	s.db, err = mysqltools.New(user, password, config)
	s.DBs = make(map[string]*DBStats)
	if err != nil { //error in connecting to database
		return nil, err
	}

	s.Collect()

	ticker := time.NewTicker(Step)
	go func() {
		for _ = range ticker.C {
			s.Collect()
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

//collects metrics
func (s *MysqlStatTables) Collect() {
	collections := []error{
		s.getDBSizes(),
		s.getTableSizes(),
		s.getTableStatistics(),
	}
	for _, err := range collections {
		if err != nil {
			s.db.Logger.Println(err)
		}
	}
}

//instantiate database metrics struct
func (s *MysqlStatTables) initializeDB(dbname string) *DBStats {
	n := new(DBStats)
	n.Metrics = newMysqlStatPerDB(s.m, dbname)
	n.Tables = make(map[string]*MysqlStatPerTable)
	return n
}

//check if database struct is instantiated, and instantiate if not
func (s *MysqlStatTables) checkDB(dbname string) error {
	if _, ok := s.DBs[dbname]; !ok {
		s.DBs[dbname] = s.initializeDB(dbname)
	}
	return nil
}

//check if table struct is instantiated, and instantiate if not
func (s *MysqlStatTables) checkTable(dbname, tblname string) error {
	s.checkDB(dbname)
	if _, ok := s.DBs[dbname].Tables[tblname]; !ok {
		s.DBs[dbname].Tables[tblname] = newMysqlStatPerTable(s.m, dbname, tblname)
	}
	return nil
}

//gets sizes of databases
func (s *MysqlStatTables) getDBSizes() error {
	res, err := s.db.QueryReturnColumnDict("SELECT @@GLOBAL.innodb_stats_on_metadata;")
	if err != nil {
		return err
	}
	for _, val := range res {
		if v, _ := strconv.ParseInt(string(val[0]), 10, 64); v == 1 {
			fmt.Println("Not capturing db/tbl sizes because @@GLOBAL.innodb_stats_on_metadata = 1")
			return errors.New("not capturing sizes: innodb_stats_on_metadata = 1")
		}
		break
	}
	cmd := `
  SELECT table_schema AS db,
         SUM( data_length + index_length ) AS db_size_bytes
    FROM information_schema.TABLES
   WHERE table_schema NOT IN ('performance_schema', 'information_schema', 'mysql')
   GROUP BY 1;`

	res, err = s.db.QueryMapFirstColumnToRow(cmd)
	if err != nil {
		return err
	}
	for key, value := range res {
		//key being the name of the database, value being its size in bytes
		dbname := string(key)
		size, _ := strconv.ParseInt(string(value[0]), 10, 64)
		if size > 0 {
			s.checkDB(dbname)
			s.DBs[dbname].Metrics.SizeBytes.Set(float64(size))
		}
	}
	return nil
}

//gets sizes of tables within databases
func (s *MysqlStatTables) getTableSizes() error {
	res, err := s.db.QueryReturnColumnDict("SELECT @@GLOBAL.innodb_stats_on_metadata;")
	if err != nil {
		return err
	}
	for _, val := range res {
		if v, _ := strconv.ParseInt(string(val[0]), 10, 64); v == int64(1) {
			fmt.Println("Not capturing db/tbl sizes because @@GLOBAL.innodb_stats_on_metadata = 1")
			return errors.New("not capturing sizes: innodb_stats_on_metadata = 1")
		}
		break
	}
	cmd := `
    SELECT table_schema AS db, table_name as tbl,
           data_length + index_length AS tbl_size_bytes
      FROM information_schema.TABLES
     WHERE table_schema NOT IN ('performance_schema', 'information_schema', 'mysql');`
	res, err = s.db.QueryReturnColumnDict(cmd)
	if err != nil {
		return err
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
			s.db.Logger.Println(err)
		}
		if size > 0 {
			s.checkTable(dbname, tblname)
			s.DBs[dbname].Tables[tblname].SizeBytes.Set(float64(size))
		}
	}
	return nil
}

//get table statistics: rows read, rows changed, rows changed x indices
func (s *MysqlStatTables) getTableStatistics() error {
	cmd := `
SELECT table_schema AS db, table_name AS tbl, 
       rows_read, rows_changed, rows_changed_x_indexes  
  FROM INFORMATION_SCHEMA.TABLE_STATISTICS
 WHERE rows_read > 0;`
	res, err := s.db.QueryReturnColumnDict(cmd)
	if len(res) == 0 || err != nil {
		return err
	}
	for i, tblname := range res["tbl"] {
		dbname := res["db"][i]
		rows_read, err := strconv.ParseInt(res["rows_read"][i], 10, 64)
		if err != nil {
			s.db.Logger.Println(err)
		}
		rows_changed, err := strconv.ParseInt(res["rows_changed"][i], 10, 64)
		if err != nil {
			s.db.Logger.Println(err)
		}
		rows_changed_x_indexes, err := strconv.ParseInt(res["rows_changed_x_indexes"][i], 10, 64)
		if err != nil {
			s.db.Logger.Println(err)
		}
		if rows_read > 0 {
			s.checkDB(dbname)
			s.checkTable(dbname, tblname)
			s.DBs[dbname].Tables[tblname].RowsRead.Set(uint64(rows_read))
		}
		if rows_changed > 0 {
			s.checkDB(dbname)
			s.checkTable(dbname, tblname)
			s.DBs[dbname].Tables[tblname].RowsRead.Set(uint64(rows_changed))
		}
		if rows_changed_x_indexes > 0 {
			s.checkDB(dbname)
			s.checkTable(dbname, tblname)
			s.DBs[dbname].Tables[tblname].RowsRead.Set(uint64(rows_changed_x_indexes))
		}
	}
	return nil
}

func (s *MysqlStatTables) Close() {
	s.db.Close()
}
