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
		fmt.Fprintf(w, "%d programs\n", len(programManager.programs))
		for _, program := range programManager.programs {
			fmt.Fprintf(w, "%s: %s\n", program.Config.Name, program.GetState())
			for _, process := range program.Processes {
				fmt.Fprintf(w, "  %v: %s\n", process.ID, process.Machine.Current)
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
		program := programManager.GetProgramByName((programName))
		if program == nil {
			fmt.Fprintf(w, "program '%s' not found", programName)
		} else {
			program.Start()
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
		program := programManager.GetProgramByName((programName))
		if program == nil {
			fmt.Fprintf(w, "program '%s' not found", programName)
		} else {
			program.Stop()
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
