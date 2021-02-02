package main

import (
	"flag"
)

type Args struct {
	DaemonArg     bool
	ConfigPathArg string
	PortArg       int
	LogPathArg    string
	BypassRootArg bool
}

func (args *Args) Parse() {
	flag.BoolVar(&args.DaemonArg, "d", false, "Launched as daemon")
	flag.StringVar(&args.ConfigPathArg, "c", configDefaultPath, "Config file location path")
	flag.IntVar(&args.PortArg, "p", 8080, "HTTP API Port")
	flag.StringVar(&args.LogPathArg, "l", logDefaultPath, "Log file location path")
	flag.BoolVar(&args.BypassRootArg, "r", false, "Be able to launch as root")
	flag.Parse()
}
