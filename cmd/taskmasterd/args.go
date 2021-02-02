package main

import (
	"flag"
)

var daemonArg bool
var configPathArg string
var portArg int
var logPathArg string
var bypassRootArg bool

func argsParse() {
	flag.BoolVar(&daemonArg, "d", false, "Launched as daemon")
	flag.StringVar(&configPathArg, "c", configDefaultPath, "Config file location path")
	flag.IntVar(&portArg, "p", 8080, "HTTP API Port")
	flag.StringVar(&logPathArg, "l", logDefaultPath, "Log file location path")
	flag.BoolVar(&bypassRootArg, "r", false, "Be able to launch as root")
	flag.Parse()
}
