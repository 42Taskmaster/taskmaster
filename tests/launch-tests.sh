#!/usr/bin/env sh

PATH=$PATH:"$PWD/node_modules/.bin":"$PWD/..":"$PWD"
ROOT_PATH=$PWD

setUp() {
    cd $ROOT_PATH/scenarios
}

tearDown() {
    echo $PWD
    pkill taskmasterd
    rm -f taskmasterd.lock
}

testInfinite() {
    cd infinite

    ./test.sh

    assertTrue $?
}

testHotReloadTotalNewConfig() {
    cd hot-reload-total-new-config

    ./test.sh

    assertTrue $?
}

testHotReloadUpdateProgramConfig() {
    cd hot-reload-update-program-config

    ./test.sh

    assertTrue $?
}

. ./vendor/shunit2/shunit2
