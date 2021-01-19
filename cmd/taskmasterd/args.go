package main

import (
	"flag"
)

var daemonArg bool
var configPathArg string
var portArg int
var logPathArg string

func argsParse() {
	flag.BoolVar(&daemonArg, "d", false, "Launched as daemon")
	flag.StringVar(&configPathArg, "c", configDefaultPath, "Config file location path")
	flag.IntVar(&portArg, "p", 8080, "HTTP API Port")
	flag.StringVar(&logPathArg, "l", logDefaultPath, "Log file location path")
	flag.Parse()
}
