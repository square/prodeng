package mysqlstat

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/square/prodeng/metrics"
)

type testMysqlDB struct {
	Logger *log.Logger
}

var (
	//testquerycol and testqueryrow map a query string to the desired test result
	//simulates QueryReturnColumnDict()
	testquerycol = map[string]map[string][]string{}

	//Simulates QueryMapFirstColumnToRow
	testqueryrow = map[string]map[string][]string{}

	//Mapping of metric and its expected value
	// defined as map of interface{}->interface{} so
	// can switch between metrics.Gauge and metrics.Counter
	// and between float64 and uint64 easily
	expectedValues = map[interface{}]interface{}{}
)

//functions that behave like mysqltools but we can make it return whatever
func (s *testMysqlDB) QueryReturnColumnDict(query string) (map[string][]string, error) {
	if query == "SHOW ENGINE INNODB STATUS" {
		return nil, errors.New(" not checking innodb parser")
	}
	return testquerycol[query], nil
}

func (s *testMysqlDB) QueryMapFirstColumnToRow(query string) (map[string][]string, error) {
	return testquerycol[query], nil
}

func (s *testMysqlDB) Log(in interface{}) {
	s.Logger.Println(in)
}

func (s *testMysqlDB) Close() {
	return
}

//initializes a test instance of MysqlStat
// instance does not connect with a db
func initMysqlStat() *MysqlStat {
	s := new(MysqlStat)
	s.db = &testMysqlDB{
		Logger: log.New(os.Stderr, "TESTING LOG: ", log.Lshortfile),
	}

	s.Metrics = MysqlStatMetricsNew(metrics.NewMetricContext("system"),
		time.Millisecond*time.Duration(1)*1000)

	return s
}

//checkResults checks the results between
func checkResults() string {
	for metric, expected := range expectedValues {
		switch m := metric.(type) {
		case *metrics.Counter:
			val, ok := expected.(uint64)
			if !ok {
				return "unexpected type"
			}
			if m.Get() != val {
				return ("unexpected value - got: " +
					strconv.FormatInt(int64(m.Get()), 10) + " but wanted " + strconv.FormatInt(int64(val), 10))
			}
		case *metrics.Gauge:
			val, ok := expected.(float64)
			if !ok {
				return "unexpected type"
			}
			if m.Get() != val {
				return ("unexpected value - got: " +
					strconv.FormatFloat(float64(m.Get()), 'f', 5, 64) + " but wanted " +
					strconv.FormatFloat(float64(val), 'f', 5, 64))
			}
		}
	}
	return ""
}

// Test basic parsing of all fields
func TestBasic(t *testing.T) {
	fmt.Println("Basic Test")
	//intitialize MysqlStat
	s := initMysqlStat()
	//set desired test output
	testquerycol = map[string]map[string][]string{

		//getSlaveStats()
		slaveQuery: map[string][]string{
			"Seconds_Behind_Master": []string{"8"},
			"Relay_Master_Log_File": []string{"some-name-bin.010"},
			"Exec_Master_Log_Pos":   []string{"79"},
		},
		// getInnodbBufferpoolMutexWaits
		mutexQuery: map[string][]string{
			"Name":   []string{"&buf_pool->LRU_list_mutex", "&buf_pool->zip_mutex"},
			"Status": []string{"os_waits=54321", "os_waits=4321"},
		},
		//getOldest
		oldestQuery: map[string][]string{
			"time": []string{"12345"},
		},
		//getQueryResponseTime
		responseTimeQuery: map[string][]string{
			"time": []string{"0.000001", "00.00001", "000.0001",
				"0000.001", "00000.01", "00000.1", "1.00000", "10.0000",
				"100.000", "1000.00", "10000.0"},
			"count": []string{"1", "2", "300", "4", "5", "6", "7", "8", "9", "10", "11"},
		},
		// getBinlogFiles
		binlogQuery: map[string][]string{
			"Log_name":  []string{"binlog.00001", "binlog.00002", "binlog.00003", "binlog.00004"},
			"File_size": []string{"1", "10", "100", "1000"}, // sum = 1111
		},
		//getNumLongRunQueries
		longQuery: map[string][]string{
			"ID": []string{"1", "2", "3", "4", "5", "6", "7"},
		},
		//getVersion
		versionQuery: map[string][]string{
			"VERSION()": []string{"1.2.34"},
		},
		// getBinlogStats
		binlogStatsQuery: map[string][]string{
			"File":     []string{"mysql-bin.003"},
			"Position": []string{"73"},
		},
		// getStackedQueries
		stackedQuery: map[string][]string{
			"identical_queries_stacked": []string{"5", "4", "3"},
			"max_age":                   []string{"10", "9", "8"},
		},
		//getSessions
		sessionQuery1: map[string][]string{
			"max_connections": []string{"10"},
		},
		sessionQuery2: map[string][]string{
			"COMMAND": []string{"Sleep", "Connect", "Binlog Dump", "something else", "database stuff"},
			"USER":    []string{"unauthenticated", "jackdorsey", "santaclaus", "easterbunny", "me"},
			"STATE":   []string{"statistics", "copying table", "Table Lock", "Waiting for global read lock", "else"},
		},
		innodbQuery: map[string][]string{
			"Value": []string{"100"},
		},
		//not going to include every metric since the parsing function is the same for each
		// missing metrics should not break metrics collector
		globalStatsQuery: map[string][]string{
			"Queries":         []string{"8"},
			"Uptime":          []string{"100"},
			"Threads_running": []string{"5"},
		},
	}
	//expected results
	expectedValues = map[interface{}]interface{}{
		s.Metrics.SlaveSecondsBehindMaster:    float64(8),
		s.Metrics.SlaveSeqFile:                float64(10),
		s.Metrics.SlavePosition:               uint64(79),
		s.Metrics.Queries:                     uint64(8),
		s.Metrics.Uptime:                      uint64(100),
		s.Metrics.ThreadsRunning:              float64(5),
		s.Metrics.MaxConnections:              float64(10),
		s.Metrics.CurrentSessions:             float64(5),
		s.Metrics.ActiveSessions:              float64(2),
		s.Metrics.UnauthenticatedSessions:     float64(1),
		s.Metrics.LockedSessions:              float64(0),
		s.Metrics.TablesLocks:                 float64(1),
		s.Metrics.CopyingToTable:              float64(1),
		s.Metrics.Statistics:                  float64(1),
		s.Metrics.IdenticalQueriesStacked:     float64(5),
		s.Metrics.IdenticalQueriesMaxAge:      float64(10),
		s.Metrics.BinlogSeqFile:               float64(3),
		s.Metrics.BinlogPosition:              uint64(73),
		s.Metrics.Version:                     float64(1.234),
		s.Metrics.ActiveLongRunQueries:        float64(7),
		s.Metrics.BinlogSize:                  float64(1111),
		s.Metrics.QueryResponseSec_0001:       uint64(300),
		s.Metrics.OldestQuery:                 float64(12345),
		s.Metrics.InnodbBufpoolLRUMutexOSWait: uint64(54321),
		s.Metrics.InnodbBufpoolZipMutexOSWait: uint64(4321),
	}
	s.Collect(0)
	time.Sleep(time.Millisecond * 1000 * 1)
	err := checkResults()
	if err != "" {
		t.Error(err)
	}
	fmt.Println("PASS")
}

//test parsing of version
func TestVersion(t *testing.T) {
	fmt.Println("Test Version")
	//intialize MysqlStat
	s := initMysqlStat()

	//set desired test result
	testquerycol = map[string]map[string][]string{
		versionQuery: map[string][]string{
			"VERSION()": []string{"123-456-789.0987"},
		},
	}
	//set expected result
	expectedValues = map[interface{}]interface{}{
		s.Metrics.Version: float64(123.4567890987),
	}
	//make sure to sleep for ~1 second before checking results
	// otherwise no metrics will be collected in time
	s.Collect(0)
	time.Sleep(time.Millisecond * 1000 * 1)
	//check results
	err := checkResults()
	if err != "" {
		t.Error(err)
	}
	//repeat for different test results
	testquerycol = map[string]map[string][]string{
		versionQuery: map[string][]string{
			"VERSION()": []string{"123ABC456-abc-987"},
		},
	}
	expectedValues = map[interface{}]interface{}{
		s.Metrics.Version: float64(123456.987),
	}
	s.Collect(0)
	time.Sleep(time.Millisecond * 1000 * 1)
	err = checkResults()
	if err != "" {
		t.Error(err)
	}

	testquerycol = map[string]map[string][]string{
		versionQuery: map[string][]string{
			"VERSION()": []string{"abcdefg-123-456-qwerty"},
		},
	}
	expectedValues = map[interface{}]interface{}{
		s.Metrics.Version: float64(0.123456),
	}
	s.Collect(0)
	time.Sleep(time.Millisecond * 1000 * 1)
	err = checkResults()
	if err != "" {
		t.Error(err)
	}
	fmt.Println("PASS")
}

func TestMutexes(t *testing.T) {
	fmt.Println("Test Mutexes")
	//intialize MysqlStat
	s := initMysqlStat()

	//set desired test result
	testquerycol = map[string]map[string][]string{
		mutexQuery: map[string][]string{
			"Name":   []string{"&buf_pool->LRU_list_mutex", "&buf_pool->zip_mutex"},
			"Status": []string{"os_waits=2", "os_waits=3"},
		},
	}

	//set expected result
	expectedValues = map[interface{}]interface{}{
		s.Metrics.InnodbBufpoolLRUMutexOSWait: uint64(2),
		s.Metrics.InnodbBufpoolZipMutexOSWait: uint64(3),
	}
	//make sure to sleep for ~1 second before checking results
	// otherwise no metrics will be collected in time
	s.Collect(0)
	time.Sleep(time.Millisecond * 1000 * 1)
	//check results
	err := checkResults()
	if err != "" {
		t.Error(err)
	}

	testquerycol = map[string]map[string][]string{
		mutexQuery: map[string][]string{
			"Name": []string{"some other string", "&buf_pool->LRU_list_mutex",
				"something else", "&buf_pool->zip_mutex"},
			"Status": []string{"os_waits=1", "os_waits=2", "os_waits=5", "os_waits=3"},
		},
	}

	expectedValues = map[interface{}]interface{}{
		s.Metrics.InnodbBufpoolLRUMutexOSWait: uint64(2),
		s.Metrics.InnodbBufpoolZipMutexOSWait: uint64(3),
	}
	s.Collect(0)
	time.Sleep(time.Millisecond * 1000 * 1)
	err = checkResults()
	if err != "" {
		t.Error(err)
	}
	fmt.Println("PASS")
}

func TestSessions(t *testing.T) {
	fmt.Println("Testing Sessions")
	s := initMysqlStat()
	testquerycol = map[string]map[string][]string{
		sessionQuery1: map[string][]string{
			"max_connections": []string{"100"},
		},
		sessionQuery2: map[string][]string{
			"COMMAND": []string{"Sleep", "Connect", "Binlog Dump", "something else", "database stuff",
				"Sleep", "Sleep", "database stuff", "other things", "square"},
			"USER": []string{"unauthenticated", "jackdorsey", "santaclaus", "easterbunny", "me",
				"also unauthenticated", "unauthenticated", "root", "root", "root"},
			"STATE": []string{"statistics", "copying another table", "Table Lock",
				"Waiting for global read lock", "else", "Table Lock", "Locked", "statistics",
				"statistics", "copying table also"},
		},
	}
	expectedValues = map[interface{}]interface{}{
		s.Metrics.MaxConnections:          float64(100),
		s.Metrics.CurrentSessions:         float64(10),
		s.Metrics.CurrentConnectionsPct:   float64(10),
		s.Metrics.ActiveSessions:          float64(5),
		s.Metrics.BusySessionPct:          float64(50),
		s.Metrics.UnauthenticatedSessions: float64(3),
		s.Metrics.LockedSessions:          float64(1),
		s.Metrics.TablesLocks:             float64(2),
		s.Metrics.GlobalReadLocks:         float64(1),
		s.Metrics.CopyingToTable:          float64(2),
		s.Metrics.Statistics:              float64(3),
	}
	s.Collect(0)
	time.Sleep(time.Millisecond * 1000 * 1)
	err := checkResults()
	if err != "" {
		t.Error(err)
	}
	fmt.Println("PASS")
}

// Test basic parsing of slave info query
func TestSlave(t *testing.T) {
	fmt.Println("Slave Stats Test")
	//intitialize MysqlStat
	s := initMysqlStat()
	//set desired test output
	testquerycol = map[string]map[string][]string{
		//getSlaveStats()
		slaveQuery: map[string][]string{
			"Seconds_Behind_Master": []string{"80"},
			"Relay_Master_Log_File": []string{"some-name-bin.01345"},
			"Exec_Master_Log_Pos":   []string{"7"},
		},
	}
	expectedValues = map[interface{}]interface{}{
		s.Metrics.SlaveSecondsBehindMaster: float64(80),
		s.Metrics.SlaveSeqFile:             float64(1345),
		s.Metrics.SlavePosition:            uint64(7),
	}
	s.Collect(0)
	time.Sleep(time.Millisecond * 1000 * 1)
	err := checkResults()
	if err != "" {
		t.Error(err)
	}

	//set desired test output
	testquerycol = map[string]map[string][]string{
		//getSlaveStats()
		slaveQuery: map[string][]string{
			"Seconds_Behind_Master": []string{"80"},
			"Relay_Master_Log_File": []string{"some.name.bin.01345"},
			"Exec_Master_Log_Pos":   []string{"7"},
		},
	}
	expectedValues = map[interface{}]interface{}{
		s.Metrics.SlaveSecondsBehindMaster: float64(80),
		s.Metrics.SlaveSeqFile:             float64(1345),
		s.Metrics.SlavePosition:            uint64(7),
	}
	s.Collect(0)
	time.Sleep(time.Millisecond * 1000 * 1)
	err = checkResults()
	if err != "" {
		t.Error(err)
	}

	fmt.Println("PASS")
}
