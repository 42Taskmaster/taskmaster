package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

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

type HttpJSONResponse struct {
	Error  string        `json:"error,omitempty"`
	Result []interface{} `json:"result,omitempty"`
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
		httpJSONResponse := HttpJSONResponse{}

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
			httpJSONResponse.Result = append(httpJSONResponse.Result, httpProgramStatus)
		}
		json, err := json.Marshal(httpJSONResponse)
		if err != nil {
			log.Panic(err)
		}
		w.Write(json)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type HttpProgramNameInputJSON struct {
	ProgramName string `json:"program_name"`
}

func httpEndpointStart(programManager *ProgramManager, w http.ResponseWriter, r *http.Request) {
	// {"program_name":""}
	switch r.Method {
	case "POST":
		var (
			input            HttpProgramNameInputJSON
			httpJSONResponse HttpJSONResponse
		)

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&input); err != nil {
			log.Panic(err)
			return
		}

		programName := input.ProgramName
		err := programManager.StartProgramByName(programName)
		if err != nil {
			httpJSONResponse.Error = err.Error()
		}
		json, err := json.Marshal(httpJSONResponse)
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
		var (
			input            HttpProgramNameInputJSON
			httpJSONResponse HttpJSONResponse
		)

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&input); err != nil {
			log.Panic(err)
			return
		}

		programName := input.ProgramName
		err := programManager.StopProgramByName(programName)
		if err != nil {
			httpJSONResponse.Error = err.Error()
		}
		json, err := json.Marshal(httpJSONResponse)
		if err != nil {
			log.Panic(err)
		}
		w.Write(json)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointRestart(programManager *ProgramManager, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var (
			input            HttpProgramNameInputJSON
			httpJSONResponse HttpJSONResponse
		)

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&input); err != nil {
			log.Panic(err)
			return
		}

		programName := input.ProgramName
		err := programManager.RestartProgramByName((programName))
		if err != nil {
			httpJSONResponse.Error = err.Error()
		}
		json, err := json.Marshal(httpJSONResponse)
		if err != nil {
			log.Panic(err)
		}
		w.Write(json)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type HttpConfigurationEndpointInputJSON struct {
	ConfigurationData string `json:"file"`
}

func httpEndpointConfiguration(programManager *ProgramManager, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Fprint(w, "get start")
	case "PUT":
		var (
			input            HttpConfigurationEndpointInputJSON
			httpJSONResponse HttpJSONResponse
		)

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&input); err != nil {
			log.Panic(err)
			return
		}

		reader := strings.NewReader(input.ConfigurationData)

		programsConfigurations, err := configParse(reader)
		programManager.LoadConfiguration(programsConfigurations)
		if err != nil {
			httpJSONResponse.Error = err.Error()
		}
		json, err := json.Marshal(httpJSONResponse)
		if err != nil {
			log.Panic(err)
		}
		w.Write(json)
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
		log.Println(r.RemoteAddr, r.Method, r.RequestURI)
		callback(programManager, w, r)
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
