all: taskmasterd taskmastersh

D_SRCS := $(wildcard cmd/taskmasterd/*.go)
SH_SRCS := $(wildcard cmd/taskmastersh/*.go)

taskmasterd: $(D_SRCS)
	go build -race -o taskmasterd $(D_SRCS)

taskmastersh: $(SH_SRCS)
	go build -race -o taskmastersh $(SH_SRCS)

tests: tests/binaries/backoff/backoff.go tests/binaries/exited/exited.go
	go build -o tests/binaries/backoff/backoff tests/binaries/backoff/backoff.go
	go build -o tests/binaries/exited/exited tests/binaries/exited/exited.go

clean:
	rm -rf taskmasterd taskmastersh

re: clean all
