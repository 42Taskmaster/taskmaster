package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

type HttpHandleFunc func(http.ResponseWriter, *http.Request)
type HttpEndpointFunc func(programManager *ProgramManager, w http.ResponseWriter, r *http.Request)

var httpEndpoints = map[string]HttpEndpointFunc{
	"/status":        httpEndpointStatus,
	"/start":         httpEndpointStart,
	"/stop":          httpEndpointStop,
	"/restart":       httpEndpointRestart,
	"/configuration": httpEndpointConfiguration,
	"/shutdown":      httpEndpointShutdown,
	"/":              httpNotFound,
}

func httpEndpointStatus(programManager *ProgramManager, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		programsStatuses := programManager.GetProgramsStatus()
		for name, processMap := range programsStatuses {
			fmt.Fprintf(w, "%s: %s\n", name, "OK")
			for id, stateType := range processMap {
				fmt.Fprintf(w, "  %v: %s\n", id, stateType)
			}
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointStart(programManager *ProgramManager, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		err := r.ParseForm()
		if err != nil {
			log.Print(err)
		}
		programName := r.Form.Get("program_name")
		err = programManager.StartProgramByName(programName)
		if err != nil {
			fmt.Fprintf(w, "error: ", err)
		} else {
			fmt.Fprintf(w, "program '%s' started", programName)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointStop(programManager *ProgramManager, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		err := r.ParseForm()
		if err != nil {
			log.Print(err)
		}
		programName := r.Form.Get("program_name")
		err = programManager.StopProgramByName(programName)
		if err != nil {
			fmt.Fprintf(w, "error: ", err)
		} else {
			fmt.Fprintf(w, "program '%s' stopped", programName)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointRestart(programManager *ProgramManager, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		err := r.ParseForm()
		if err != nil {
			log.Print(err)
		}
		programName := r.Form.Get("program_name")
		program := programManager.GetProgramByName((programName))
		if program == nil {
			fmt.Fprintf(w, "program '%s' not found", programName)
		} else {
			program.Restart()
			fmt.Fprintf(w, "program '%s' restarted", programName)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointConfiguration(programManager *ProgramManager, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Fprint(w, "get start")
	case "PUT":
		fmt.Fprint(w, "put start")
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointShutdown(programManager *ProgramManager, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "DELETE":
		fmt.Fprint(w, "shutdown")
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpNotFound(programManager *ProgramManager, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func httpHandleEndpoint(programManager *ProgramManager, callback HttpEndpointFunc) HttpHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		callback(programManager, w, r)
		log.Println(r.RemoteAddr, r.Method, r.RequestURI)
	}
}

func httpSetup(programManager *ProgramManager) {
	for uri, callback := range httpEndpoints {
		http.HandleFunc(uri, httpHandleEndpoint(programManager, callback))
	}
}

func httpListenAndServe() {
	log.Printf("Launching HTTP REST API on port :%d", portArg)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(portArg), nil))
}
