package main

import (
	"flag"
)

var daemonArg bool
var configPathArg string
var portArg int

func argsParse() {
	flag.BoolVar(&daemonArg, "d", false, "Launched as daemon")
	flag.StringVar(&configPathArg, "c", configDefaultPath, "Config file location path")
	flag.IntVar(&portArg, "p", 8080, "HTTP API Port")
	flag.Parse()
}
