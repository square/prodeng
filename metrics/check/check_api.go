//Copyright (c) 2014 Square, Inc

package check

type Checker interface {
	//Returns Warnings using input function. e.g. OutputWarnings(formats.Basic)
	OutputWarnings(func(Checker, ...string) error, ...string) error

	//Check the metrics against their thresholds
	CheckMetrics() error

	//Return results of metric checks.
	// Result in the form of sectionName -> check results
	GetWarnings() map[string]metricResults
}
