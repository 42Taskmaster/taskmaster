all: taskmasterd taskmastersh

D_SRCS := $(wildcard cmd/taskmasterd/*.go)
SH_SRCS := $(wildcard cmd/taskmastersh/*.go)

taskmasterd: $(D_SRCS)
	go build -o taskmasterd $(D_SRCS)

taskmastersh: $(SH_SRCS)
	go build -o taskmastersh $(SH_SRCS)

clean:
	rm -rf taskmasterd taskmastersh

re: clean all
