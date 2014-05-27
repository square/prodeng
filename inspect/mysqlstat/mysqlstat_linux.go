// Copyright (c) 2014 Square, Inc

package mysqlstat

import (
	"container/list"
	"errors"
	"fmt"
	//for testing only. remove later
	"math"

	"github.com/square/prodeng/metrics"

	"strconv"
	"strings"
	"time"
)

import "database/sql"
import _ "github.com/go-sql-driver/mysql"

const (
	DEFAULT_MYSQL_USER        = "root"
	CRIT                      = "CRIT"
	WARN                      = "WARN"
	OK                        = "OK"
	SIP_RW_HOST               = "SIP_RW_HOST"
	SIP_RO_HOST               = "SIP_RO_HOST"
	BACKUP                    = "MYSQL_BACKUP_HOSTS"
	AUTO_DEBUG_RETENTION_MINS = 20
)

type MYSQLStat struct {
	host          string
	roles         map[string]map[string]bool // mapping of strings to maps (that map strings to bools)
	db            sql.DB
	nag_msg       map[string]*list.List
	nag           map[string]map[string]string // mapping of strings to maps (that map strings to strings)
	params        map[string]int
	cluster       map[string]bool
	state_current map[string]int
	state_last    map[string]int
}

//returns an array of pointers to strings of specified length
func create_address_Array(n int) []*string {
	pointers := [n]*string{}
	for i := 0; i < n; i++ {
		var s string
		pointers[i] = &s
	}
	return pointers
}

// returns value of columns in an array
func (s *MYSQLStat) query_return_columns_arrays(query) ([]string, [][]string, error) {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, nil, err
	}
	column_names, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	columns := len(column_names) // number of columns in rows
	result := [columns][]string{}
	pointers := create_address_array(columns)
	for rows.Next() {
		//scan each column of the row into separate space
		err = rows.Scan(pointers...)
		//TODO: determine how to handle error
		for i := range pointers {
			//append each element into corresponding column array
			result[i] = append(result[i], *pointers[i])
		}
	}
	err = rows.Err() //get any error encountered during iteration

	return column_names, result, err
}

//TODO: query correctly
func (s *MYSQLStat) query_return_columns_dict(query) (map[string][]string, error) {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	column_names, err := rows.Columns()
	result := make(map[string][]string)

}

func (s *MYSQLStat) calc_rate(key, current_value) int {
	key = strconv.Itoa(key)
	stamp := key + "_stamp"
	s.state_current[key] = current_value
	s.state_current[stamp] = int(time.Unix())
	_, key_ok := s.state_last[key]
	_, stamp_ok := s.state_last[stamnp]
	if !key_ok && !stamp_ok {
		return -1
	}
	return math.Abs((s.state_current[key] - s.state_last[key]) / (s.state_current[stamp] - s.state_last[stamp]))
}

func New(m *metrics.MetricContext, Step time.Duration) *MYSQLStat {
	s := new(MYSQLStat)
	//TODO: connect mysql db here, or in collect
	s.host = ""
	s.roles = make(map[string]map[string]bool)
	s.roles[SIP_RW_HOST] = make(map[string]bool)
	s.roles[SIP_RO_HOST] = make(map[string]bool)
	s.roles[BACKUP] = make(map[string]bool)
	s.nag_msg = make(map[string]*list.List)
	s.nag = make(map[string]map[string]string)
	s.params = make(map[string]int)
	s.cluster = make(map[string]bool)
	s.state_current = make(map[string]int)
	s.state_last = make(map[string]int)

	ticker := time.NewTicker(Step)
	go func() {
		for _ = range ticker.C {
			m.Collect()
		}
	}()

	return m
}

func (s *MYSQLStat) Collect() {
	//TODO: implement
	fmt.Println("Collect")
}

// get_slave_stats modeled from function in mysql_health.py line 546
func (s *MYSQLStat) get_slave_stats() error {
	var msg string
	if _, ok := s.roles[BACKUP][s.host]; ok {
		cmd := `SELECT COUNT(*) FROM information_schema.processlist WHERE user LIKE '%backup%';`
		column_names, res, err := query_return_columns_arrays(cmd)
		//TODO: finish this
	}

	res, err := query_return_columns_dict("SHOW SLAVE STATUS;")
	if len(res) == 0 {
		s.nag_msg[CRIT].PushBack("Slave is not configured")
		return nil
	}
	row = res[0]
	r, ok := row["Seconds_Behind_Master"]
	params["slave_seconds_behind_master"] = r
	if !ok {
		io_error, io_ok := row["Last_IO_Error"]
		sql_error, sql_ok := row["Last_SQL_Error"]
		if !io_ok && !sql_ok {
			msg = "Slave is NOT running"
		}
		msg = io_error + ' ' + sql_error
		s.nag_msg[CRIT].PushBack(msg)
		s.params["slave_seconds_behind_master"] = -1
		s.nag["slave_seconds_behind_master"] = map[string]string{CRIT: "<0"}
	} else if _, ok := s.cluster["unity"]; ok { //if "unity" is in s.cluster
		s.nag["slave_seconds_behind_cluster"] = map[string]string{CRIT: ">= 3600", WARN: ">= 300"}
	} else if _, ok := s.roles[BACKUP][s.host]; ok { //if host is in roles[BACKUP]
		s.nag["slave_seconds_behind_master"] = map[string]string{CRIT: ">= 3600", WARN: ">= 1800"}
	} else {
		s.nag["slave_seconds_behind_master"] = map[string]string{CRIT: ">= 600", WARN: ">= 300"}
	}
	relay_master_log_file, ok := row["Relay_Master_Log_File"]
	if !ok {
		return errors.New("Error, expected RelayMasterLogFile")
	}
	slave_seqfile, err := strconv.Atoi(strings.Split(relay_master_log_file, ".")[1])
	if err != nil {
		return err
	}
	s.state_current["slave_seqfile"] = slave_seqfile
	s.params["slave_seqfile"] = slave_seqfile
	slave_position, err := strconv.Atoi(row["Exec_Master_Log_Pos"])
	if err != nil {
		return err
	}
	s.params["slave_position"] = slave_position
	if state_last, ok := s.state_last["slave_seqfile"]; ok && state_last != s.state_currrent["slave_seqfile"] {
		return nil
	}
	s.params["slave_commit_Bps"] = s.calc_rate("slave_position", slave_position)
	return nil

}
