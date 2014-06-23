package formats

import (
	"fmt"
	"strconv"

	"github.com/square/prodeng/metrics/check"
)

func Basic(hc check.Checker, s ...string) error {
	for metric, result := range hc.GetWarnings() {
		fmt.Println(metric + ": " + result.Message)
		for checkName, val := range result.Checks {
			fmt.Println("    " + checkName + ": " + strconv.FormatBool(val))
		}
	}
	return nil
}
