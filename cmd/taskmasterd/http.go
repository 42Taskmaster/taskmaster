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
type HttpEndpointFunc func(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request)

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
	ProgramID    string             `json:"program_id"`
	ProgramState ProgramState       `json:"program_state"`
	Processes    []HttpProcessState `json:"processes"`
}

type HttpProcessState struct {
	Id    string            `json:"id"`
	State machine.StateType `json:"state"`
}

func httpEndpointStatus(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		httpJSONResponse := HttpJSONResponse{}

		programs, err := taskmasterd.GetSortedPrograms()
		if err != nil {
			httpJSONResponse.Error = err.Error()
		} else {
			for _, program := range programs {
				processes, err := program.GetSortedProcesses()
				if err != nil {
					httpJSONResponse.Error = err.Error()
					break
				}
				httpProgramStatus := HttpProgramState{
					ProgramID:    program.configuration.Name,
					ProgramState: GetProgramState(processes),
				}
				for _, process := range processes {
					httpProcessState := HttpProcessState{
						Id:    process.ID,
						State: process.Machine.Current(),
					}
					httpProgramStatus.Processes = append(httpProgramStatus.Processes, httpProcessState)
				}
				httpJSONResponse.Result = append(httpJSONResponse.Result, httpProgramStatus)
			}
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
	ProgramID string `json:"program_id"`
}

func httpEndpointStart(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
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

		program, err := taskmasterd.GetProgramById(input.ProgramID)
		if err != nil {
			httpJSONResponse.Error = err.Error()
		} else {
			program.Start()
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

func httpEndpointStop(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
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

		program, err := taskmasterd.GetProgramById(input.ProgramID)
		if err != nil {
			httpJSONResponse.Error = err.Error()
		} else {
			program.Stop()
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

func httpEndpointRestart(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
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

		program, err := taskmasterd.GetProgramById(input.ProgramID)
		if err != nil {
			httpJSONResponse.Error = err.Error()
		} else {
			program.Restart()
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

func httpEndpointConfiguration(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
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
		taskmasterd.LoadProgramsConfigurations(programsConfigurations)
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

func httpEndpointShutdown(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "DELETE":
		fmt.Fprint(w, "shutdown")
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpNotFound(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func httpHandleEndpoint(taskmasterd *Taskmasterd, callback HttpEndpointFunc) HttpHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RemoteAddr, r.Method, r.RequestURI)
		callback(taskmasterd, w, r)
	}
}

func httpSetup(taskmasterd *Taskmasterd) {
	for uri, callback := range httpEndpoints {
		http.HandleFunc(uri, httpHandleEndpoint(taskmasterd, callback))
	}
}

func httpListenAndServe(port int) {
	log.Printf("Launching HTTP REST API on port :%d", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}
