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
	"gopkg.in/yaml.v2"
)

type HttpHandleFunc func(http.ResponseWriter, *http.Request)
type HttpEndpointFunc func(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request)

var httpEndpoints = map[string]HttpEndpointFunc{
	"/status":                httpEndpointStatus,
	"/start":                 httpEndpointStart,
	"/stop":                  httpEndpointStop,
	"/restart":               httpEndpointRestart,
	"/configuration":         httpEndpointConfiguration,
	"/configuration/refresh": httpEndpointRefreshConfiguration,
	"/programs":              httpEndpointCreateProgram,
	"/logs":                  httpEndpointLogs,
	"/shutdown":              httpEndpointShutdown,
	"/version":               httpEndpointVersion,
	"/":                      httpNotFound,
}

type HttpJSONResponse struct {
	Error  string      `json:"error,omitempty"`
	Result interface{} `json:"result,omitempty"`
}

type HttpProgramNameInputJSON struct {
	ProgramID string `json:"program_id"`
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

type HttpConfigurationEndpointInputJSON struct {
	ConfigurationData string `json:"data"`
}

type HttpConfiguration struct {
	Data string `json:"data"`
}

type HttpLogs struct {
	Data string `json:"data"`
}

type HttpCreateProgramInputJSON struct {
	ProgramYaml
}

func RespondJSON(resp HttpJSONResponse, w http.ResponseWriter) {
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(resp); err != nil {
		return
	}
}

func httpEndpointStatus(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		programs, err := taskmasterd.GetSortedPrograms()
		if err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		httpPrograms := HttpPrograms{}
		for _, program := range programs {
			processes, err := program.GetSortedProcesses()
			if err != nil {
				RespondJSON(HttpJSONResponse{
					Error: err.Error(),
				}, w)
				return
			}

			config, err := program.GetConfig()
			if err != nil {
				RespondJSON(HttpJSONResponse{
					Error: err.Error(),
				}, w)
				return
			}

			httpProgram := HttpProgram{
				Id:            program.configuration.Name,
				Configuration: config,
				State:         GetProgramState(processes),
			}

			for _, process := range processes {
				processState := process.GetStateMachineCurrentState()

				pid := 0
				if processState == ProcessStateRunning {
					if cmd := process.GetCmd(); cmd != nil && cmd.Process != nil {
						pid = cmd.Process.Pid
					}
				}

				serializedProcess := process.Serialize()

				httpProcess := HttpProcess{
					ID:    serializedProcess.ID,
					Pid:   pid,
					State: processState,

					StartedAt: serializedProcess.StartedAt,
					EndedAt:   serializedProcess.EndedAt,
				}

				httpProgram.Processes = append(httpProgram.Processes, httpProcess)
			}

			httpPrograms.Programs = append(httpPrograms.Programs, httpProgram)
		}

		RespondJSON(HttpJSONResponse{
			Result: httpPrograms,
		}, w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointStart(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var input HttpProgramNameInputJSON

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&input); err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		program, err := taskmasterd.GetProgramById(input.ProgramID)
		if err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		program.Start()

		RespondJSON(HttpJSONResponse{}, w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointStop(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var input HttpProgramNameInputJSON

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&input); err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		program, err := taskmasterd.GetProgramById(input.ProgramID)
		if err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		program.Stop()

		RespondJSON(HttpJSONResponse{}, w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointRestart(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var input HttpProgramNameInputJSON

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&input); err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		program, err := taskmasterd.GetProgramById(input.ProgramID)
		if err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		program.Restart()

		RespondJSON(HttpJSONResponse{}, w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointConfiguration(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		programsConfigurationsChan := make(chan ProgramsYaml)

		taskmasterd.ProgramTaskChan <- TaskmasterdTaskGetProgramsConfigurations{
			TaskBase: TaskBase{
				Action: TaskmasterdTaskActionGetProgramsConfigurations,
			},
			ProgramsConfigurationsChan: programsConfigurationsChan,
		}

		programsConfigurations := <-programsConfigurationsChan

		programsConfigurationsBuffer, err := yaml.Marshal(programsConfigurations)
		if err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		RespondJSON(HttpJSONResponse{
			Result: HttpConfiguration{
				Data: string(programsConfigurationsBuffer),
			},
		}, w)
	case "PUT":
		var input HttpConfigurationEndpointInputJSON

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&input); err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		reader := strings.NewReader(input.ConfigurationData)
		errorChan := make(chan error)

		taskmasterd.ProgramTaskChan <- TaskmasterdTaskRefreshConfigurationFromReader{
			TaskBase: TaskBase{
				Action: TaskmasterdTaskActionRefreshConfigurationFromReader,
			},
			Reader:    reader,
			ErrorChan: errorChan,
		}

		err := <-errorChan

		if err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		RespondJSON(HttpJSONResponse{}, w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointRefreshConfiguration(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		taskmasterd.ProgramTaskChan <- TaskmasterdTaskActionRefreshConfigurationFromConfigurationFile

		RespondJSON(HttpJSONResponse{}, w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointLogs(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		configFileData, err := ioutil.ReadFile(taskmasterd.Args.LogPathArg)
		if err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		RespondJSON(HttpJSONResponse{
			Result: HttpLogs{
				Data: string(configFileData),
			},
		}, w)
	case "DELETE":
		configFile, err := os.OpenFile(taskmasterd.Args.LogPathArg, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		configFile.Truncate(0)
		configFile.Close()

		RespondJSON(HttpJSONResponse{}, w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointCreateProgram(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var newProgramConfiguration HttpCreateProgramInputJSON

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&newProgramConfiguration); err != nil {
			RespondJSON(HttpJSONResponse{
				Error: err.Error(),
			}, w)
			return
		}

		errorChan := make(chan error)

		taskmasterd.ProgramTaskChan <- TaskmasterdTaskAddProgramConfiguration{
			TaskBase: TaskBase{
				Action: TaskmasterdTaskActionAddProgramConfiguration,
			},
			ProgramConfiguration: newProgramConfiguration.ProgramYaml,
			ErrorChan:            errorChan,
		}

		err := <-errorChan
		if err == nil {
			RespondJSON(HttpJSONResponse{}, w)
			return
		}

		RespondJSON(HttpJSONResponse{
			Error: err.Error(),
		}, w)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func httpEndpointVersion(taskmasterd *Taskmasterd, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		RespondJSON(HttpJSONResponse{
			Result: VERSION,
		}, w)
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
			w.Header().Set("Content-Type", "application/json")
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
