package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

type HttpEndpointFn func(http.ResponseWriter, *http.Request)
type HttpEndpointClosure func(*ProgramManager) HttpEndpointFn

var httpEndpoints = map[string]HttpEndpointClosure{
	"/status":        httpEndpointStatus,
	"/start":         httpEndpointStart,
	"/stop":          httpEndpointStop,
	"/restart":       httpEndpointRestart,
	"/configuration": httpEndpointConfiguration,
	"/shutdown":      httpEndpointShutdown,
	"/":              httpNotFound,
}

func httpEndpointStatus(programManager *ProgramManager) HttpEndpointFn {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fmt.Fprintf(w, "%+v\n", programManager)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func httpEndpointStart(programManager *ProgramManager) HttpEndpointFn {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			fmt.Fprintf(w, "start")
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func httpEndpointStop(programManager *ProgramManager) HttpEndpointFn {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			fmt.Fprintf(w, "stop")
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func httpEndpointRestart(programManager *ProgramManager) HttpEndpointFn {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			fmt.Fprintf(w, "restart")
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func httpEndpointConfiguration(programManager *ProgramManager) HttpEndpointFn {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fmt.Fprintf(w, "GET configuration")
		} else if r.Method == "PUT" {
			fmt.Fprintf(w, "PUT configuration")
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func httpEndpointShutdown(programManager *ProgramManager) HttpEndpointFn {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			fmt.Fprintf(w, "shutdown")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func httpNotFound(programManager *ProgramManager) HttpEndpointFn {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}
}

func httpSetup(programManager *ProgramManager) {
	for endpoint, callback := range httpEndpoints {
		http.HandleFunc(endpoint, callback(programManager))
	}
}

func httpListenAndServe() {
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(portArg), nil))
}
