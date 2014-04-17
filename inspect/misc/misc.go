// Copyright (c) 2014 Square, Inc

package misc

import (
	"bufio"
	"os"
	"reflect"
	"strconv"
	"regexp"
	"strings"
	"errors"
	"path/filepath"
	"io/ioutil"
	"github.com/square/prodeng/metrics"
)

type Interface interface{}

func ParseUint(in string) uint64 {
	out, err := strconv.ParseUint(in, 10, 64) // decimal, 64bit
	if err != nil {
		return 0
	}
	return out
}

func ReadUintFromFile(path string) uint64 {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return 0
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		return ParseUint(scanner.Text())
	}
	return 0
}

func InitializeMetrics(c Interface, m *metrics.MetricContext) {
	s := reflect.ValueOf(c).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if f.Kind().String() != "ptr" {
			continue
		}
		if f.Type().Elem() == reflect.TypeOf(metrics.Gauge{}) {
			f.Set(reflect.ValueOf(m.NewGauge(typeOfT.Field(i).Name)))
		}
		if f.Type().Elem() == reflect.TypeOf(metrics.Counter{}) {
			f.Set(reflect.ValueOf(m.NewGauge(typeOfT.Field(i).Name)))
		}
	}
	return
}

// move these to cgroup library
// discover where memory subsystem is mounted

func FindCgroupMount(subsystem string) (string, error) {

	file, err := os.Open("/proc/mounts")
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		f := regexp.MustCompile("[\\s]+").Split(scanner.Text(), 6)
		if f[2] == "cgroup" {
			for _, o := range strings.Split(f[3], ",") {
				if o == subsystem {
					return f[1], nil
				}
			}
		}
	}

	return "", errors.New("no cgroup mount found")
}

func FindCgroups(mountpoint string) ([]string, error) {
	cgroups := make([]string, 0, 128)

	_ = filepath.Walk(
		mountpoint,
		func(path string, f os.FileInfo, _ error) error {
			if f.IsDir() && path != mountpoint {
				// skip cgroups with no tasks
				dat, err := ioutil.ReadFile(path + "/" + "tasks")
				if err == nil && len(dat) > 0 {
					cgroups = append(cgroups, path)
				}
			}
			return nil
		})

	return cgroups, nil
}
