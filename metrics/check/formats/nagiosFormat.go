package formats

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"code.google.com/p/goconf/conf" // used for parsing config files
	"github.com/square/prodeng/metrics/check"
)

var (
	nagLevels = map[string]int{"OK": 0, "WARN": 1, "CRIT": 2, "UNKNOWN": 3}
)

type nagSender struct {
	server           string
	serviceType      string
	hostname         string
	NSCA_BINARY_PATH string
	NSCA_CONFIG_PATH string
}

//Nagios statement formatted as: host service state_code message
func Nagios(hc check.Checker, configFile ...string) error {
	ns := getNagiosInfo(configFile[0])
	res := []string{}
	critical := []string{}
	warning := []string{}
	ok := []string{}
	for sectionName, result := range hc.GetWarnings() {
		crit := false
		warn := false
		for checkName, res := range result.Checks {
			if strings.Contains(strings.ToLower(checkName), "crit") && res {
				crit = true
			} else if strings.Contains(strings.ToLower(checkName), "warn") && res {
				warn = true
			}
		}
		if crit {
			critical = append(critical, sectionName) //result.Message)
		} else if warn {
			warning = append(warning, sectionName) //result.Message)
		} else {
			ok = append(ok, sectionName) //result.Message)
		}
	}
	messages := map[string][]string{"CRIT": critical, "WARN": warning, "OK": ok}
	for level, msgs := range messages {
		if len(msgs) == 0 {
			continue
		}
		res = append(res, fmt.Sprintf("%s\t%s\t%d\t%s\n", ns.hostname, ns.serviceType, nagLevels[level], strings.Join(msgs, ", ")))
	}
	for _, m := range res {
		fmt.Println(m)
	}
	return nil
}

//Sends nagios server metrics warnings
func SendNagiosPassive(messages []string, configFile string) error {
	ns := getNagiosInfo(configFile)
	for _, message := range messages {
		printCmd := exec.Command("printf", fmt.Sprintf("\"%s\\n\"", message))
		sendCmd := exec.Command(ns.NSCA_BINARY_PATH, ns.server, "-c "+ns.NSCA_CONFIG_PATH)
		sendCmd.Stdin, _ = printCmd.StdoutPipe()
		sendCmd.Start()
		printCmd.Run()
		err := sendCmd.Wait()
		if err != nil {
			return err
		}
	}
	return nil
}

//grabs nagios info from config file
//TODO: can either grab this info from config file or give as input to send function
func getNagiosInfo(configFile string) nagSender {
	ns := &nagSender{}
	c, err := conf.ReadConfigFile(configFile)
	if !c.HasSection("nagios") || err != nil {
		return *ns
	}
	ns.hostname, _ = os.Hostname()
	ns.server, _ = c.GetString("nagios", "server")
	ns.NSCA_BINARY_PATH, _ = c.GetString("nagios", "nsca-binary-path")
	ns.NSCA_CONFIG_PATH, _ = c.GetString("nagios", "nsca-config-path")
	ns.serviceType, _ = c.GetString("nagios", "service")
	return *ns
}
