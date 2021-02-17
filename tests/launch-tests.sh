#!/usr/bin/env sh

PATH=$PATH:"$PWD/node_modules/.bin":"$PWD/..":"$PWD"
ROOT_PATH=$PWD

setUp() {
    cd $ROOT_PATH/scenarios
    pkill taskmasterd
    git restore .

    true
}

tearDown() {
    echo $PWD
    pkill taskmasterd
    git restore .

    true
}

testInfinite() {
    cd infinite
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?
}

testHotReloadTotalNewConfig() {
    cd hot-reload-total-new-config
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?
}

testHotReloadUpdateProgramConfig() {
    cd hot-reload-update-program-config
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?
}

testNotFoundCommand() {
    cd not-found-command
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?
}

. ./vendor/shunit2/shunit2
