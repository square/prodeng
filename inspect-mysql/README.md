#inspect-mysql


inspect-mysql is a collection of libraries for gathering metrics of mysql databases.

inspect command line is a utility that gives a brief overview on the databases: version, uptime, queries made, and database sizes.

inspect gathers the following metrics:
- Version number
- Slave Stats
- Global Stats
- Binlog Stats
- Stacked Query info
- Session Info
- Innodb stats
- Long Run Query info
- Query Response Times

##Installation

1. Get Go
2. `go get -v -u github.com/square/prodeng/inspect-mysql`

##Usage

###Command Line

./bin/inspect-mysql

```
--------------------------
Version: 5.1234
Queries made: 123456
Uptime: 543210
Database sizes:
    database_name: 0.54 GB
    other_database_name: 12.31 GB

```

###Server

_inspect-mysql_ can be run in server mode to run continuously and expose all metrics via HTTP JSON api

./bin/inspect-mysql -server -address :12345

```
[
{"type": "counter", "name": "mysqlstat.Queries", "value": 9342251, "rate": 31.003152},
{"type": "counter", "name": "mysqltablestat.database_name.table_name.RowsRead", "value": 0, "rate": 0.000000},
{"type": "counter", "name": "mysqltablestat.database_name.table_name.RowsChanged", "value": 0, "rate": 0.000000},
{"type": "counter", "name": "mysqltablestat.database_name.other_table_name.RowsChanged", "value": 0, "rate": 0.000000},
{"type": "counter", "name": "mysqltablestat.database_name.table_name.RowsChangedXIndexes", "value": 0, "rate": 0.000000},
... truncated
{"type": "counter", "name": "mysqlstat.SortMergePasses", "value": 0, "rate": 0.000000}]
```

###Example API Use


```
// Import packages
import "github.com/square/prodeng/inspect-mysql"
import "github.com/square/prodeng/metrics"

// Initialize a metric context
m := metrics.NewMetricContext("system")

// Collect mysql metrics every m.Step seconds
sqlstats := mysqlstat.New(m, time.Millisecond*2000)

// Collects mysql metrics for specific databases and tables
sqltablestats := mysqlstattable.New(m, time.Millisecond*2000)
```

All metrics collected are exported, so any metric may be accessed using Get():
```
// Print the number of queries accessed
fmt.Println(sqlstats.Metrics.Queries.Get())

// Print the size of table t1 in databse db1
fmt.Println(sqltablestats.DBs["db1"].Tables["t1"].Metrics.SizeBytes.Get())
```

##Testing 

Packages are tested using Go's testing package.
To test:
1. cd to the directory containing the .go and _test.go files
2. Run `go test`. You can also run with the `-v` option for a verbose output. For these tests, many logs are expected so stderr is redirected to a file `test.log` 

Tests for each metric may be added to `mysqlstat_test.go` and `mysqlstat-tables_test.go`. These tests do not connect to a database. Instead, the desired test input is hard coded into each test. Testing for the parser for the Innodb metrics are located in `mysqltools_test.go`. 






