// Copyright (c) 2014 Square, Inc
//
// Must download driver for mysql use. Run the following command:
//      go get github.com/go-sql-driver/mysql
// in order to successfully build/install

package mysqltools

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

import "database/sql"
import _ "github.com/go-sql-driver/mysql"

type MysqlDB struct {
	db        *sql.DB
	dsnString string
}

const (
	DEFAULT_MYSQL_USER = "root"
	MAX_RETRIES        = 5
)

type Configuration struct {
	password []string
}

//wrapper for make_query, where if there is an error querying the database
// retry connecting to the db and make the query
func (database *MysqlDB) queryDb(query string) ([]string, [][]string, error) {
	var err error
	for attempts := 0; attempts <= MAX_RETRIES; attempts++ {
		err = database.db.Ping()
		if err == nil {
			if cols, data, err := database.makeQuery(query); err == nil {
				return cols, data, nil
			} else {
				return nil, nil, err
			}
		}
		database.db.Close()
		database.db, err = sql.Open("mysql", database.dsnString)
	}
	return nil, nil, err
}

//makes a query to the database
// returns array of column names and arrays of data stored as string
// string equivalent to []byte
// data stored as 2d array with each subarray containing a single column's data
func (database *MysqlDB) makeQuery(query string) ([]string, [][]string, error) {
	rows, err := database.db.Query(query)
	if err != nil {
		return nil, nil, err
	}

	column_names, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	columns := len(column_names)
	values := make([][]string, columns)
	tmp_values := make([]string, columns)

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
func (database *MysqlDB) QueryReturnColumnDict(query string) (map[string][]string, error) {
	column_names, values, err := database.queryDb(query)
	result := make(map[string][]string)
	for i, col := range column_names {
		result[col] = values[i]
	}
	return result, err
}

//return values of query in a mapping of first columns entry -> row
func (database *MysqlDB) QueryMapFirstColumnToRow(query string) (map[string][]string, error) {
	_, values, err := database.queryDb(query)
	result := make(map[string][]string)
	for i, name := range values[0] {
		for j, vals := range values {
			if j != 0 {
				result[string(name)] = append(result[string(name)], vals[i])
			}
		}
	}
	return result, err
}

//makes dsn to open up connection
//dsn is made up of the format:
//     [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
func makeDsn(dsn map[string]string) string {
	var dsnString string
	user, ok := dsn["user"]
	if ok {
		dsnString = user
	}
	password, ok := dsn["password"]
	if ok {
		dsnString = dsnString + ":" + password
	}
	dsnString = dsnString + "@"
	dsnString = dsnString + dsn["unix_socket"]
	dsnString = dsnString + "/" + dsn["db"]
	fmt.Println("dsn string: " + dsnString)
	return dsnString
}

func New(user, password string) (*MysqlDB, error) {
	fmt.Println("connecting to database")
	database := new(MysqlDB)
	// build dsn info here
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
	database.dsnString = makeDsn(dsn)

	//first attempt to connect here
	db, err := sql.Open("mysql", database.dsnString)
	if err == nil {
		fmt.Println("opened database without password")
		database.db = db
		return database, nil
	}
	ini_file := creds[user]
	if _, err := os.Stat(ini_file); err != nil {
		return nil, errors.New("'" + ini_file + "' does not exist")
	}
	// read ini file to get password
	file, _ := os.Open(ini_file)
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err = decoder.Decode(&configuration)
	dsn["password"] = configuration.password[0]
	database.dsnString = makeDsn(dsn)

	//second attempt to connect here
	db, err = sql.Open("mysql", database.dsnString)
	if err != nil {
		return nil, err
	}
	database.db = db

	err = database.db.Ping()
	if err != nil {
		return nil, err
	}
	return database, nil
}

func (database *MysqlDB) Close() {
	database.db.Close()
}
