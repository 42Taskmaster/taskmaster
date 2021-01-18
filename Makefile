all: taskmasterd taskmastersh

taskmasterd:
	go build -o taskmasterd cmd/taskmasterd/main.go

taskmastersh:
	go build -o taskmastersh cmd/taskmastersh/main.go

clean:
	rm taskmasterd taskmastersh
