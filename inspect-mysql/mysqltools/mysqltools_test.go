//Copyright (c) 2014 Square, Inc
//
// Testing for mysqltools

package mysqltools

import (
	"os"
	"testing"
)

var (
	testInnodbStats = InnodbStats{}
)

const (
	prefix     = "./testfiles/"
	fakeDBName = "foobar"
)

//Basic parse. should run correctly
func TestParseInnodbStats(t *testing.T) {
	file, err := os.Open(prefix + "innodb_output.txt")
	if err != nil {
		t.Error("Couldn't open test file")
	}
	defer file.Close()
	x := 10000
	data := make([]byte, x)
	_, err = file.Read(data)
	blob := string(data)
	test, err := ParseInnodbStats(blob)
	if err != nil || test == nil {
		t.Error(err)
	}
	tests := map[string]string{"page_hash": "139112",
		"dictionary_cache":    "5771169",
		"file_system":         "1053936",
		"pages_read":          "534",
		"trxes_not_started":   "6",
		"log_sequence_number": "139401310",
		"log_flushed_up_to":   "139401310",
		"log_io_done":         "277124",
		"modified_age":        "0",
		"OS_file_reads":       "1597",
		"avg_bytes_per_read":  "0",
		"fsyncs_per_s":        "0.89",
	}

	for key, val := range tests {
		if test.Metrics[key] != val {
			t.Error(key + " not parsed correctly. Expected: " + val + ", Got: " + test.Metrics[key])
		}
	}
}

//input that totally does not match regular expressions,
// should not return an error, but will return an empty result
func TestParseMalformedInput(t *testing.T) {
	file, err := os.Open(prefix + "innodb_giberish.txt")
	if err != nil {
		t.Error("Couldn't open test file")
	}
	defer file.Close()
	x := 10000
	data := make([]byte, x)
	_, err = file.Read(data)
	blob := string(data)
	test1, err := ParseInnodbStats(blob)
	if err != nil || test1 == nil {
		t.Error(err)
	}
	//expecting 0 entries into metrics
	for key, _ := range test1.Metrics {
		t.Error("All metrics should be nil, but found:" + key)
	}
}

//input text has missing fields, but should still parse the remaining file
func TestParseMissingFields(t *testing.T) {
	testFiles := []string{"innodb_missing1.txt",
		"innodb_missing2.txt",
		"innodb_missing3.txt"}
	for _, testFile := range testFiles {
		file, err := os.Open(prefix + testFile)
		if err != nil {
			t.Error("Couldn't Open test file: " + testFile)
		}
		data := make([]byte, 10000)
		_, err = file.Read(data)
		blob := string(data)
		test, err := ParseInnodbStats(blob)
		if err != nil {
			t.Error(err)
		}
		if len(test.Metrics) == 0 {
			t.Error("Could not collect Metrics")
		}
		file.Close()
	}
}

//Playing with the regular expression matching
func TestParseDifferentRegexps(t *testing.T) {
	testFiles := []string{"innodb_regexp1.txt",
		"innodb_regexp2.txt",
		"innodb_regexp3.txt"}
	for _, testFile := range testFiles {
		file, err := os.Open(prefix + testFile)
		if err != nil {
			t.Error("Couldn't Open test file: " + testFile)
		}
		data := make([]byte, 10000)
		_, err = file.Read(data)
		blob := string(data)
		test, err := ParseInnodbStats(blob)
		if err != nil {
			t.Error(err)
		}
		if len(test.Metrics) == 0 {
			t.Error("Could not collect Metrics")
		}
		tests := map[string]string{"trxes_not_started": "6",
			"undo":                    "123",
			"OS_file_reads":           "1597",
			"OS_fsyncs":               "367474",
			"avg_bytes_per_read":      "10",
			"fsyncs_per_s":            "0.89",
			"log_sequence_number":     "139401310",
			"checkpoint_age_target":   "78300347",
			"log_io_per_sec":          "0.41",
			"page_hash":               "139112",
			"lock_system":             "335128",
			"buffer_pool_hit_rate":    "0.42",
			"cache_hit_pct":           "42",
			"total_mem_by_read_views": "472",
			"total_mem":               "137363456",
			"adaptive_hash":           "2250352",
		}
		for key, val := range tests {
			if test.Metrics[key] != val {
				t.Error(key + " not parsed correctly. Expected: " + val + ", Got: " + test.Metrics[key])
			}
		}
		file.Close()
	}
}

func BenchmarkRead(b *testing.B) {
	file, err := os.Open(prefix + "innodb_output.txt")
	if err != nil {
		b.Error("Couldn't open test file")
	}
	defer file.Close()
	x := 10000
	data := make([]byte, x)
	_, err = file.Read(data)
	blob := string(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		test1, err := ParseInnodbStats(blob)
		if err != nil || test1 == nil {
			b.Error(err)
		}
	}
}
