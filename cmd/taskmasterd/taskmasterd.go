package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"

	"gopkg.in/yaml.v2"
)

const VERSION = "1.0.0"

var ErrProgramNotFound = errors.New("program not found")

type Taskmasterd struct {
	Args Args

	ProgramsConfiguration ProgramsYaml

	ProgramTaskChan chan Tasker

	Context context.Context
	Cancel  context.CancelFunc

	Closed chan struct{}
}

type NewTaskmasterdArgs struct {
	Args                  Args
	ProgramsConfiguration ProgramsYaml
	Context               context.Context
	Cancel                context.CancelFunc
}

func NewTaskmasterd(args NewTaskmasterdArgs) *Taskmasterd {
	taskmasterd := &Taskmasterd{
		Args:                  args.Args,
		ProgramsConfiguration: args.ProgramsConfiguration,
		Context:               args.Context,
		Cancel:                args.Cancel,
		ProgramTaskChan:       make(chan Tasker),
		Closed:                make(chan struct{}),
	}

	go taskmasterd.WaitDeath()

	go taskmasterd.Monitor()

	return taskmasterd
}

func copyProgramMap(programs map[string]Program) map[string]Program {
	programsCopy := make(map[string]Program)

	for programID, program := range programs {
		programsCopy[programID] = program
	}

	return programsCopy
}

func (taskmasterd *Taskmasterd) WaitDeath() {
	<-taskmasterd.Context.Done()

	programsClosed := make(chan struct{})

	programs, err := taskmasterd.GetPrograms()
	if err != nil {
		log.Fatalf("fetching programs: %v\n", err)
		return
	}

	go func() {
		programsToWait := make([]chan interface{}, 0, len(programs))

		for _, program := range programs {
			doneCh := program.StopAndWait()

			programsToWait = append(programsToWait, doneCh)
		}

		for _, doneCh := range programsToWait {
			<-doneCh
		}

		close(programsClosed)
	}()

	<-programsClosed

	close(taskmasterd.Closed)
}

func (taskmasterd *Taskmasterd) Monitor() {
	programs := make(map[string]Program)

	for task := range taskmasterd.ProgramTaskChan {
		switch task.GetAction() {
		case TaskmasterdTaskActionGet:
			taskmasterdTask := task.(TaskmasterdTaskGet)

			program, ok := programs[taskmasterdTask.ProgramID]
			if !ok {
				close(taskmasterdTask.ResponseChan)
				break
			}

			taskmasterdTask.ResponseChan <- program
		case TaskmasterdTaskActionGetAll:
			taskmasterdTask := task.(TaskmasterdTaskGetAll)

			taskmasterdTask.ResponseChan <- copyProgramMap(programs)
		case TaskmasterdTaskActionAdd:
			taskmasterdTask := task.(TaskmasterdTaskAdd)

			program := taskmasterdTask.Program
			programs[program.configuration.Name] = program
			if program.configuration.Autostart {
				program.Start()
			}
		case TaskmasterdTaskActionRemove:
			taskmasterdTask := task.(TaskmasterdTask)

			program := programs[taskmasterdTask.ProgramID]
			delete(programs, taskmasterdTask.ProgramID)
			program.Stop()

		case TaskmasterdTaskActionGetProgramsConfigurations:
			getProgramsConfigurationsTask := task.(TaskmasterdTaskGetProgramsConfigurations)

			getProgramsConfigurationsTask.ProgramsConfigurationsChan <- taskmasterd.ProgramsConfiguration

		case TaskmasterdTaskActionRefreshConfigurationFromConfigurationFile:
			configReader, err := configGetFileReader(taskmasterd.Args.ConfigPathArg)
			if err != nil {
				// TODO: improve log
				log.Println("could not get config file reader", err)
				break
			}

			programsYamlConfiguration, programsConfigurations, err := configParse(configReader)

			taskmasterd.ProgramsConfiguration = programsYamlConfiguration

			go taskmasterd.LoadProgramsConfigurations(programsConfigurations)

		case TaskmasterdTaskActionAddProgram:
			addProgramConfigurationTask := task.(TaskmasterdTaskAddProgram)
			programConfiguration := addProgramConfigurationTask.ProgramConfiguration

			configuration, err := programConfiguration.Validate(ProgramYamlValidateArgs{
				PickProgramName: true,
			})
			if err != nil {
				addProgramConfigurationTask.ErrorChan <- err
				break
			}

			_, ok := programs[configuration.Name]
			if ok {
				addProgramConfigurationTask.ErrorChan <- fmt.Errorf("program '%s' already exists", configuration.Name)
				break
			}

			go taskmasterd.LoadProgramConfiguration(configuration)

			taskmasterd.ProgramsConfiguration.Programs[configuration.Name] = programConfiguration

			if err := taskmasterd.PersistProgramsConfigurationsToDisk(); err != nil {
				addProgramConfigurationTask.ErrorChan <- err
				break
			}

			close(addProgramConfigurationTask.ErrorChan)

		case TaskmasterdTaskActionEditProgram:
			editProgramTask := task.(TaskmasterdTaskEditProgram)
			programConfiguration := editProgramTask.ProgramConfiguration

			configuration, err := programConfiguration.Validate(ProgramYamlValidateArgs{
				PickProgramName: true,
			})
			if err != nil {
				editProgramTask.ErrorChan <- err
				break
			}

			if editProgramTask.ProgramId != configuration.Name {
				program, ok := programs[editProgramTask.ProgramId]
				if !ok {
					editProgramTask.ErrorChan <- fmt.Errorf("program not found")
					break
				}
				program.Stop()
				delete(programs, editProgramTask.ProgramId)
				delete(taskmasterd.ProgramsConfiguration.Programs, editProgramTask.ProgramId)
			}

			go taskmasterd.LoadProgramConfiguration(configuration)
			taskmasterd.ProgramsConfiguration.Programs[configuration.Name] = programConfiguration

			if err := taskmasterd.PersistProgramsConfigurationsToDisk(); err != nil {
				editProgramTask.ErrorChan <- err
				break
			}

			close(editProgramTask.ErrorChan)

		case TaskmasterdTaskActionDeleteProgram:
			deleteProgramTask := task.(TaskmasterdTaskDeleteProgram)

			delete(programs, deleteProgramTask.ProgramId)
			delete(taskmasterd.ProgramsConfiguration.Programs, deleteProgramTask.ProgramId)

			if err := taskmasterd.PersistProgramsConfigurationsToDisk(); err != nil {
				deleteProgramTask.ErrorChan <- err
				break
			}

			close(deleteProgramTask.ErrorChan)

		case TaskmasterdTaskActionRefreshConfigurationFromReader:
			refreshConfigurationFromReaderTask := task.(TaskmasterdTaskRefreshConfigurationFromReader)

			programsYamlConfiguration, programsConfigurations, err := configParse(refreshConfigurationFromReaderTask.Reader)
			if err != nil {
				refreshConfigurationFromReaderTask.ErrorChan <- err
				break
			}

			taskmasterd.ProgramsConfiguration = programsYamlConfiguration

			go taskmasterd.LoadProgramsConfigurations(programsConfigurations)

			if err := taskmasterd.PersistProgramsConfigurationsToDisk(); err != nil {
				refreshConfigurationFromReaderTask.ErrorChan <- err
				break
			}

			close(refreshConfigurationFromReaderTask.ErrorChan)
		}
	}
}

func (taskmasterd *Taskmasterd) PersistProgramsConfigurationsToDisk() error {
	file, err := os.OpenFile(taskmasterd.Args.ConfigPathArg, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	encoder := yaml.NewEncoder(file)
	if err := encoder.Encode(taskmasterd.ProgramsConfiguration); err != nil {
		return err
	}

	return nil
}

func (taskmasterd *Taskmasterd) LoadProgramConfiguration(config ProgramConfiguration) error {
	log.Printf("Loading program '%s' configuration...", config.Name)
	program, err := taskmasterd.GetProgramById(config.Name)
	if err != nil {
		program := NewProgram(NewProgramArgs{
			Context:       taskmasterd.Context,
			Configuration: config,
		})

		select {
		case taskmasterd.ProgramTaskChan <- TaskmasterdTaskAdd{
			TaskBase: TaskBase{
				Action: TaskmasterdTaskActionAdd,
			},
			Program: program,
		}:
		case <-taskmasterd.Context.Done():
			return ErrChannelClosed
		}

		return nil
	}

	program.ProcessTaskChan <- ProgramTaskRootActionWithPayload{
		ProgramTaskRootAction: ProgramTaskRootAction{
			TaskBase{
				Action: ProgramTaskActionSetConfig,
			},
		},
		Payload: config,
	}

	return nil
}

func (taskmasterd *Taskmasterd) LoadProgramsConfigurations(configs ProgramsConfigurations) error {
	log.Printf("Loading %d program(s) configuration(s)...", len(configs))
	programs, err := taskmasterd.GetPrograms()
	if err != nil {
		return err
	}

	for _, config := range configs {
		err := taskmasterd.LoadProgramConfiguration(config)
		if err != nil {
			return err
		}
	}

	for programID := range programs {
		_, isProgramInNewConfig := configs[programID]
		if isProgramInNewConfig {
			continue
		}

		select {
		case taskmasterd.ProgramTaskChan <- TaskmasterdTask{
			TaskBase: TaskBase{
				Action: TaskmasterdTaskActionRemove,
			},
			ProgramID: programID,
		}:
		case <-taskmasterd.Context.Done():
			return ErrChannelClosed
		}
	}
	return nil
}

func (taskmasterd *Taskmasterd) GetProgramById(id string) (Program, error) {
	responseChan := make(chan Program)

	select {
	case taskmasterd.ProgramTaskChan <- TaskmasterdTaskGet{
		TaskBase: TaskBase{
			Action: TaskmasterdTaskActionGet,
		},

		ProgramID:    id,
		ResponseChan: responseChan,
	}:
	case <-taskmasterd.Context.Done():
		return Program{}, ErrChannelClosed
	}

	select {
	case program := <-responseChan:
		if !program.Valid {
			return Program{}, ErrProgramNotFound
		}
		return program, nil
	case <-taskmasterd.Context.Done():
		return Program{}, ErrChannelClosed
	}
}

func (taskmasterd *Taskmasterd) GetPrograms() (map[string]Program, error) {
	responseChan := make(chan map[string]Program)

	taskmasterd.ProgramTaskChan <- TaskmasterdTaskGetAll{
		TaskBase: TaskBase{
			Action: TaskmasterdTaskActionGetAll,
		},
		ResponseChan: responseChan,
	}

	programs := <-responseChan
	return programs, nil
}

func (taskmasterd *Taskmasterd) GetSortedPrograms() ([]Program, error) {
	programs, err := taskmasterd.GetPrograms()
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(programs))
	for id := range programs {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	sortedPrograms := make([]Program, 0, len(ids))
	for _, id := range ids {
		program, ok := programs[id]
		if ok {
			sortedPrograms = append(sortedPrograms, program)
		}
	}

	return sortedPrograms, nil
}

func (taskmasterd *Taskmasterd) Quit() {
	taskmasterd.Cancel()
}
