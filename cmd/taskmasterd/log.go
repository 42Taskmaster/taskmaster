package main

import (
	"log"
	"os"
)

const logDefaultPath = "./taskmasterd.log"

func logLogo() {
	log.Print(" _____         _                        _            ")
	log.Print("|_   _|       | |                      | |           ")
	log.Print("  | | __ _ ___| | ___ __ ___   __ _ ___| |_ ___ _ __ ")
	log.Print("  | |/ _` / __| |/ / '_ ` _ \\ / _` / __| __/ _ \\ '__|")
	log.Print("  | | (_| \\__ \\   <| | | | | | (_| \\__ \\ ||  __/ |   ")
	log.Print("  \\_/\\__,_|___/_|\\_\\_| |_| |_|\\__,_|___/\\__\\___|_| ")
}

func logGetFile(args Args) *os.File {
	logFile, err := os.OpenFile(args.LogPathArg, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(args.LogPathArg, err)
	}
	return logFile
}
