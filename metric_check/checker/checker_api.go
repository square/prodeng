//Copyright (c) 2014 Square, Inc

package checker

type Checker interface {
	//Returns Warnings specified in config file; e.g. to nagios, commandline
	OutputWarnings() error

	//Check the metrics against their thresholds
	CheckMetrics() error

	//Group the warnings by their levels, i.e. CRIT, WARN, OK
	GetWarnings() map[string]metricResults

	OutputNagiosFormat() []string
}
