package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/42Taskmaster/taskmaster/machine"
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
	Error  string      `json:"error,omitempty"`
	Result interface{} `json:"result,omitempty"`
}

type HttpPrograms struct {
	Programs []HttpProgram `json:"programs"`
}

type HttpProgram struct {
	Id            string               `json:"id"`
	State         ProgramState         `json:"state"`
	Configuration ProgramConfiguration `json:"configuration"`
	Processes     []HttpProcess        `json:"processes"`
}

type HttpProcess struct {
	ID    string            `json:"id"`
	Pid   int               `json:"pid"`
	State machine.StateType `json:"state"`

	StartedAt time.Time `json:"startedAt"`
	EndedAt   time.Time `json:"endedAt"`
}
type HttpConfiguration struct {
	Data string `json:"data"`
}

func httpEndpointStatus(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		httpJSONResponse := HttpJSONResponse{}

		programs, err := taskmasterd.GetSortedPrograms()
		if err != nil {
			httpJSONResponse.Error = err.Error()
		} else {
			httpPrograms := HttpPrograms{}
			for _, program := range programs {
				processes, err := program.GetSortedProcesses()
				if err != nil {
					httpJSONResponse.Error = err.Error()
					break
				}
				config, err := program.GetConfig()
				if err != nil {
					httpJSONResponse.Error = err.Error()
					break
				}
				httpProgram := HttpProgram{
					Id:            program.configuration.Name,
					Configuration: config,
					State:         GetProgramState(processes),
				}
				for _, process := range processes {
					pid := 0
					if cmd := process.GetCmd(); cmd != nil && cmd.Process != nil {
						pid = cmd.Process.Pid
					}

					serializedProcess := process.Serialize()

					httpProcess := HttpProcess{
						ID:    serializedProcess.ID,
						Pid:   pid,
						State: process.GetStateMachineCurrentState(),

						StartedAt: serializedProcess.StartedAt,
						EndedAt:   serializedProcess.EndedAt,
					}
					httpProgram.Processes = append(httpProgram.Processes, httpProcess)
				}
				httpPrograms.Programs = append(httpPrograms.Programs, httpProgram)
			}
			httpJSONResponse.Result = httpPrograms
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
	ConfigurationData string `json:"data"`
}

func httpEndpointConfiguration(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		var (
			httpJSONResponse HttpJSONResponse
		)

		configFileData, err := ioutil.ReadFile(taskmasterd.Args.ConfigPathArg)
		if err != nil {
			httpJSONResponse.Error = err.Error()
		} else {
			httpJSONResponse.Result = HttpConfiguration{
				Data: string(configFileData),
			}
		}

		json, err := json.Marshal(httpJSONResponse)
		if err != nil {
			log.Panic(err)
		}
		w.Write(json)
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
		if err != nil {
			httpJSONResponse.Error = err.Error()
		} else {
			configFile, err := os.OpenFile(taskmasterd.Args.ConfigPathArg, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				httpJSONResponse.Error = err.Error()
			} else {
				configFile.Truncate(0)
				configFile.WriteString(input.ConfigurationData)
				configFile.Close()
				taskmasterd.LoadProgramsConfigurations(programsConfigurations)
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

func httpEndpointShutdown(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "DELETE":
		taskmasterd.Quit()
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
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method != "OPTIONS" {
			callback(taskmasterd, w, r)
		}
	}
}

func httpSetup(taskmasterd *Taskmasterd) {
	for uri, callback := range httpEndpoints {
		http.HandleFunc(uri, httpHandleEndpoint(taskmasterd, callback))
	}
}

func httpListenAndServe(ctx context.Context, port int) chan struct{} {
	server := http.Server{
		Addr: ":" + strconv.Itoa(port),
	}
	idleConnectionsClosed := make(chan struct{})

	go func() {
		<-ctx.Done()

		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}

		close(idleConnectionsClosed)
	}()

	log.Printf("Launching HTTP REST API on port :%d", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	return idleConnectionsClosed
}
