// Copyright (c) 2014 Square, Inc
//
// Must download driver for mysql use. Run the following command:
//      go get github.com/go-sql-driver/mysql
// in order to successfully build/install

package mysqlstat

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/square/prodeng/inspect/misc"
	"github.com/square/prodeng/metrics"
)

import "database/sql"
import _ "github.com/go-sql-driver/mysql"

const (
	DEFAULT_MYSQL_USER = "root"
	MAX_RETRIES        = 5
)

type Configuration struct {
	password []string
}

type MysqlStat struct {
	Metrics    *MysqlStatMetrics
	m          *metrics.MetricContext
	db         *sql.DB
	dsn_string string
}

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

//wrapper for make_query, where if there is an error querying the database
// retry connecting to the db and make the query
func (s *MysqlStat) query_db(query string) ([]string, [][]sql.RawBytes, error) {
	var err error
	for attempts := 0; attempts <= MAX_RETRIES; attempts++ {
		err = s.db.Ping()
		if err == nil {
			if cols, data, err := s.make_query(query); err == nil {
				return cols, data, nil
			} else {
				return nil, nil, err
			}
		}
		s.db.Close()
		s.db, err = sql.Open("mysql", s.dsn_string)
	}
	return nil, nil, err
}

//makes a query to the database
// returns array of column names and arrays of data stored as sql.RawBytes
// sql.RawBytes equivalent to []byte
// data stored as 2d array with each subarray containing a single column's data
func (s *MysqlStat) make_query(query string) ([]string, [][]sql.RawBytes, error) {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, nil, err
	}

	column_names, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	columns := len(column_names)
	values := make([][]sql.RawBytes, columns)
	tmp_values := make([]sql.RawBytes, columns)

	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &tmp_values[i]
	}

	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, nil, err
		}
		for i, col := range tmp_values {
			values[i] = append(values[i], col)
		}
	}
	err = rows.Err()

	return column_names, values, nil
}

//return values of query in a mapping of column_name -> column
func (s *MysqlStat) query_return_columns_dict(query string) (map[string][]sql.RawBytes, error) {
	column_names, values, err := s.query_db(query)
	result := make(map[string][]sql.RawBytes)
	for i, col := range column_names {
		result[col] = values[i]
	}
	return result, err
}

//return values of query in a mapping of first columns entry -> row
func (s *MysqlStat) query_map_first_column_to_row(query string) (map[string][]sql.RawBytes, error) {
	_, values, err := s.query_db(query)
	result := make(map[string][]sql.RawBytes)
	for i, name := range values[0] {
		for j, vals := range values {
			if j != 0 {
				result[string(name)] = append(result[string(name)], vals[i])
			}
		}
	}
	return result, err
}

//initializes mysqlstat
// starts off collect
func New(m *metrics.MetricContext, Step time.Duration, user string, password string) (*MysqlStat, error) {
	fmt.Println("starting")
	s := new(MysqlStat)
	// connect to database
	err := s.connect(user, password)
	if err != nil { //error in connecting to database
		return nil, err
	}
	s.Metrics = MysqlStatMetricsNew(m, Step)

	err = s.db.Ping()
	if err != nil {
		fmt.Println("Cant ping database")
		fmt.Println(err)
	}

	defer s.db.Close()
	fmt.Println("starting collect")
	s.Collect()

	ticker := time.NewTicker(Step)
	//go func() {
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
	misc.InitializeMetrics(c, m, "mysqlstat", true)
	return c
}

//makes dsn to open up connection
//dsn is made up of the format:
//     [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
func make_dsn(dsn map[string]string) string {
	var dsn_string string
	user, ok := dsn["user"]
	if ok {
		dsn_string = user
	}
	password, ok := dsn["password"]
	if ok {
		dsn_string = dsn_string + ":" + password
	}
	dsn_string = dsn_string + "@"
	dsn_string = dsn_string + dsn["unix_socket"]
	dsn_string = dsn_string + "/" + dsn["db"]
	fmt.Println("dsn string: " + dsn_string)
	return dsn_string
}

//attempts connecting to database using given information
// if failed on first attempt, try getting password from ini file
func (s *MysqlStat) connect(user, password string) error {
	dsn := map[string]string{"db": "information_schema"}
	creds := map[string]string{"root": "/root/.my.cnf", "nrpe": "/etc/my_nrpe.cnf"}
	if user == "" {
		user = DEFAULT_MYSQL_USER
		dsn["user"] = DEFAULT_MYSQL_USER
	} else {
		dsn["user"] = user
	}
	if password != "" {
		dsn["password"] = password
	}
	socket_file := "/var/lib/mysql/mysql.sock"
	if _, err := os.Stat(socket_file); err == nil {
		dsn["unix_socket"] = socket_file
	}
	s.dsn_string = make_dsn(dsn)
	db, err := sql.Open("mysql", s.dsn_string)
	if err == nil {
		fmt.Println("opened database without password")
		s.db = db
		return nil
	}
	ini_file := creds[user]
	if _, err := os.Stat(ini_file); err != nil {
		return errors.New("'" + ini_file + "' does not exist")
	}
	// read ini file to get password
	file, _ := os.Open(ini_file)
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err = decoder.Decode(&configuration)
	dsn["password"] = configuration.password[0]
	s.dsn_string = make_dsn(dsn)
	db, err = sql.Open("mysql", s.dsn_string)
	if err == nil {
		s.db = db
		return nil
	}
	return err
}

func (s *MysqlStat) Collect() {
	s.get_slave_stats()
	s.get_global_status()
}

// get_slave_stats gets slave statistics
func (s *MysqlStat) get_slave_stats() error {
	res, _ := s.query_return_columns_dict("SHOW SLAVE STATUS;")
	//	fmt.Println("Result when querying 'SHOW SLAVE STATUS;'")
	//	fmt.Println(res)

	if len(res["Seconds_Behind_Master"]) > 0 {
		seconds_behind_master, _ := strconv.Atoi(string(res["Seconds_Behind_Master"][0]))
		s.Metrics.SecondsBehindMaster.Set(float64(seconds_behind_master))
	} else {
		fmt.Println("no seconds behind master data")
	}

	relay_master_log_file, _ := res["Relay_Master_Log_File"]
	if len(relay_master_log_file) > 0 {
		slave_seqfile, err := strconv.Atoi(strings.Split(string(relay_master_log_file[0]), ".")[1])
		s.Metrics.SlaveSeqFile.Set(float64(slave_seqfile))

		if err != nil {
			return err
		}
	} else {
		fmt.Println("no relay master log file")
	}

	if len(res["Exec_Master_Log_Pos"]) > 0 {
		slave_position, err := strconv.Atoi(string(res["Exec_Master_Log_Pos"][0]))
		if err != nil {
			return err
		}
		s.Metrics.SlavePosition.Set(float64(slave_position))
	} else {
		fmt.Println("no Exec_Master_Log_Pos")
	}
	return nil
}

//gets global statuses
func (s *MysqlStat) get_global_status() error {
	res, _ := s.query_map_first_column_to_row("SHOW GLOBAL STATUS;")
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
	res, _ := s.query_return_columns_dict("SHOW MASTER STATUS;")
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
	res, _ := s.query_return_columns_dict(cmd)
	if len(res) > 0 {
		count, _ := strconv.Atoi(string(res["identical_queries_stacked"][0]))
		s.Metrics.IdenticalQueriesStacked.Set(uint64(count))
		age, _ := strconv.Atoi(string(res["max_age"][0]))
		s.Metrics.IdenticalQueriesMaxAge.Set(uint64(age))
	}
	return nil
}
