package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/VisorRaptors/taskmaster/machine"
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

type HttpJsonResponse struct {
	Error  string        `json:"error"`
	Result []interface{} `json:"result"`
}

type HttpProgramState struct {
	ProgramName  string             `json:"program_name"`
	ProgramState machine.StateType  `json:"program_state"`
	Processes    []HttpProcessState `json:"processes"`
}

type HttpProcessState struct {
	Id    string            `json:"id"`
	State machine.StateType `json:"state"`
}

func httpEndpointStatus(programManager *ProgramManager, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		httpJsonResponse := HttpJsonResponse{}

		for _, program := range programManager.GetSortedPrograms() {
			httpProgramStatus := HttpProgramState{
				ProgramName:  program.Config.Name,
				ProgramState: program.GetState(),
			}
			for _, process := range program.GetSortedProcesses() {
				httpProcessState := HttpProcessState{
					Id:    process.ID,
					State: process.Machine.Current(),
				}
				httpProgramStatus.Processes = append(httpProgramStatus.Processes, httpProcessState)
			}
			httpJsonResponse.Result = append(httpJsonResponse.Result, httpProgramStatus)
		}
		json, err := json.Marshal(httpJsonResponse)
		if err != nil {
			log.Panic(err)
		}
		w.Write(json)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointStart(programManager *ProgramManager, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		httpJsonResponse := HttpJsonResponse{}
		err := r.ParseForm()
		if err != nil {
			httpJsonResponse.Error = err.Error()
			json, err := json.Marshal(httpJsonResponse)
			if err != nil {
				log.Panic(err)
			}
			w.Write(json)
			return
		}
		programName := r.Form.Get("program_name")
		err = programManager.StartProgramByName(programName)
		if err != nil {
			httpJsonResponse.Error = err.Error()
		}
		json, err := json.Marshal(httpJsonResponse)
		if err != nil {
			log.Panic(err)
		}
		w.Write(json)
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
			fmt.Fprintf(w, "error: %v", err)
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
