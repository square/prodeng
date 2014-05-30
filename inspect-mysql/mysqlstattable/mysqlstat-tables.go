// Copyright (c) 2014 Square, Inc
//
// Must download driver for mysql use. Run the following command:
//      go get github.com/go-sql-driver/mysql
// in order to successfully build/install

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

type MysqlStats struct {
	DBs map[string]*DBStats
	m   *metrics.MetricContext
	db  *mysqltools.MysqlDB
}

type DBStats struct {
	Tables  map[string]*MysqlStatPerTable
	Metrics *MysqlStatPerDB
}

type MysqlStatPerTable struct {
	Size *metrics.Gauge
	// These add a lot of metrics, so commented out for easier testing
	//   potentially consolidate into a per database metric for sake of volume
	//    RowsRead *metrics.Counter
	//    RowsChanged *metrics.Counter
	//    RowsChangedXIndexes *metrics.Counter
	//TODO: add the other metrics
}

type MysqlStatPerDB struct {
	Size *metrics.Gauge
	//TODO: add the other metrics
}

//initializes mysqlstat
// starts off collect
func New(m *metrics.MetricContext, Step time.Duration, user string, password string) (*MysqlStats, error) {
	s := new(MysqlStats)
	s.m = m
	// connect to database
	var err error
	s.db, err = mysqltools.New(user, password)
	s.DBs = make(map[string]*DBStats)
	if err != nil { //error in connecting to database
		return nil, err
	}

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

//initialize  per database metrics
func newMysqlStatPerDB(m *metrics.MetricContext, dbname string) *MysqlStatPerDB {
	o := new(MysqlStatPerDB)
	misc.InitializeMetrics(o, m, "sqldbstat."+dbname, true)
	return o
}

//initialize per table metrics
func newMysqlStatPerTable(m *metrics.MetricContext, dbname, tblname string) *MysqlStatPerTable {
	o := new(MysqlStatPerTable)

	misc.InitializeMetrics(o, m, "sqltablestat."+dbname+"."+tblname, true)
	return o
}

func (s *MysqlStats) Collect() {
	s.getDBSizes()
	s.getTableSizes()
	//    s.getTableStatistics()
}

//instantiate database metrics struct
func (s *MysqlStats) initializeDB(dbname string) *DBStats {
	n := new(DBStats)
	n.Metrics = newMysqlStatPerDB(s.m, dbname)
	n.Tables = make(map[string]*MysqlStatPerTable)
	return n
}

//check if database struct is instantiated, and instantiate if not
func (s *MysqlStats) checkDB(dbname string) error {
	if _, ok := s.DBs[dbname]; !ok {
		s.DBs[dbname] = s.initializeDB(dbname)
	}
	return nil
}

//check if table struct is instantiated, and instantiate if not
func (s *MysqlStats) checkTable(dbname, tblname string) error {
	s.checkDB(dbname)
	if _, ok := s.DBs[dbname].Tables[tblname]; !ok {
		s.DBs[dbname].Tables[tblname] = newMysqlStatPerTable(s.m, dbname, tblname)
	}
	return nil
}

//gets sizes of databases
func (s *MysqlStats) getDBSizes() error {
	res, _ := s.db.QueryReturnColumnDict("SELECT @@GLOBAL.innodb_stats_on_metadata;")
	for _, val := range res {
		if v, _ := strconv.Atoi(string(val[0])); v == 1 {
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

	res, _ = s.db.QueryMapFirstColumnToRow(cmd)
	for key, value := range res {
		//key being the name of the database, value being its size in bytes
		dbname := string(key)
		size, _ := strconv.Atoi(string(value[0]))
		if size > 0 { //50*1024*1024 { //don't bother with small guys
			s.checkDB(dbname)
			s.DBs[dbname].Metrics.Size.Set(float64(size)) //this is getting condoluded
		}
	}
	return nil
}

//gets sizes of tables within databases
func (s *MysqlStats) getTableSizes() error {
	res, _ := s.db.QueryReturnColumnDict("SELECT @@GLOBAL.innodb_stats_on_metadata;")
	for _, val := range res {
		if v, _ := strconv.Atoi(string(val[0])); v == 1 {
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
	res, _ = s.db.QueryReturnColumnDict(cmd)
	tbl_count := len(res["tbl"])
	for i := 0; i < tbl_count; i++ {
		dbname := string(res["db"][i])
		tblname := string(res["tbl"][i])
		size, _ := strconv.Atoi(string(res["tbl_size_bytes"][i]))
		if size > 0 {
			s.checkTable(dbname, tblname)
			s.DBs[dbname].Tables[tblname].Size.Set(float64(size)) //this is looking way too complex.
		}

	}
	return nil
}

/*
func (s *MysqlStats) getTableStatistics() {
	cmd := `
  SELECT table_schema AS db, table_name AS tbl,
         rows_read, rows_changed, rows_changed_x_indexes,
    FROM INFORMATION_SCHEMA.TABLE_STATISTICS
   WHERE rows_read > 0;`
	res, _ := s.db.QueryReturnColumnDict(cmd)
	//TODO: implement
    if len(res) == 0 {
        return
    }
    for i, tblname := range res["tbl"] {
        dbname := res["db"][i]
        rows_read, _ := strconv.Atoi(res["rows_read"][i])
        rows_changed, _ := strconv.Atoi(res["rows_changed"][i])
        rows_changed_x_indexes, _ := strconv.Atoi(res["rows_changed_x_indexes"][i])
        s.DBs[dbname].Tables[tblname].RowsRead.Set(uint64(rows_read))
        s.DBs[dbname].Tables[tblname].RowsChanged.Set(uint64(rows_changed))
        s.DBs[dbname].Tables[tblname].RowsChangedXIndexes.Set(uint64(rows_changed_x_indexes))
    }
} */