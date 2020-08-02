package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/vipinvkmenon/awsalarms/plugins/inputs/awsalarms"

	"github.com/influxdata/telegraf/plugins/common/shim"
)

var pollInterval = flag.Duration("poll_interval", 2*time.Minute, "how often to send metrics")
var pollIntervalDisabled = flag.Bool("poll_interval_disabled", false, "how often to send metrics")
var configFile = flag.String("config", "", "path to the config file for this plugin")
var err error

func main() {
	// parse command line options
	flag.Parse()
	if *pollIntervalDisabled {
		*pollInterval = shim.PollIntervalDisabled
	}

	shim := shim.New()

	err = shim.LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Err loading input: %s\n", err)
		os.Exit(1)
	}

	if err := shim.Run(*pollInterval); err != nil {
		fmt.Fprintf(os.Stderr, "Err: %s\n", err)
		os.Exit(1)
	}
}
