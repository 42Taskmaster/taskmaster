GO_FLAGS = -race

D_SRCS = $(wildcard cmd/taskmasterd/*.go)
SH_SRCS = $(wildcard cmd/taskmastersh/*.go)

TESTS_BIN_SRCS = $(wildcard tests/binaries/*/main.go)
TESTS_BIN = $(patsubst tests/binaries/%/main.go,tests/bin_%,$(TESTS_BIN_SRCS))

all: taskmasterd taskmastersh

taskmasterd: $(D_SRCS)
	go build $(GO_FLAGS) -o taskmasterd $(D_SRCS)

taskmastersh: $(SH_SRCS)
	go build $(GO_FLAGS) -o taskmastersh $(SH_SRCS)

tests: taskmasterd $(TESTS_BIN)

tests/bin_%: tests/binaries/%/main.go
	go build $(GO_FLAGS) -o $@ $<

clean:
	rm -rf taskmasterd taskmastersh
	rm -rf $(TESTS_BIN)

re: clean all tests
