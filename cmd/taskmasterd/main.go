package main

func main() {
	argsParse()

	programsYaml := configParse()

	daemonInit()

	lockFileCreate()
	defer lockFileRemove()

	signalsSetup()

	programManager := NewProgramManager()
	programManager.programs = programsParse(programsYaml)

	//programManager.StartPrograms()

	httpSetup(programManager)
	httpListenAndServe()
}
