#!/usr/bin/env sh

PATH=$PATH:"$PWD/node_modules/.bin":"$PWD/..":"$PWD"
ROOT_PATH=$PWD

setUp() {
    cd $ROOT_PATH/scenarios
    pkill taskmasterd

    true
}

tearDown() {
    echo $PWD
    pkill taskmasterd

    git checkout .

    true
}

testInfinite() {
    cd infinite
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?

    git diff --exit-code . > /dev/null

    assertTrue "Files should have not been modified but were" $?
}

testHotReloadTotalNewConfig() {
    cd hot-reload-total-new-config
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?

    git diff --exit-code . > /dev/null

    assertFalse "Files should have been modified but were not" $?
}

testHotReloadUpdateProgramConfig() {
    cd hot-reload-update-program-config
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?

    git diff --exit-code . > /dev/null

    assertFalse "Files should have been modified but were not" $?
}

testNotFoundCommand() {
    cd not-found-command
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?

    git diff --exit-code . > /dev/null

    assertTrue "Files should have not been modified but were" $?

}

testCreate() {
    cd create-program
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?

    git diff --exit-code . > /dev/null

    assertFalse "Files should have been modified but were not" $?
}

testEdit() {
    cd edit-program
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?

    git diff --exit-code . > /dev/null

    assertFalse "Files should have been modified but were not" $?
}

testDelete() {
    cd delete-program
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?

    git diff --exit-code . > /dev/null

    assertFalse "Files should have been modified but were not" $?
}

testVersion() {
    cd version
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?

    git diff --exit-code . > /dev/null

    assertTrue "Files should have not been modified but were" $?
}

testAutomaticallyRestartOnBackoffState() {
    cd automatically-restart-on-backoff-state
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?

    git diff --exit-code . > /dev/null

    assertTrue "Files should have not been modified but were" $?
}

testStartStopRestartAll() {
    cd start-stop-restart-all
    rm -f taskmasterd.log

    ./test.sh

    assertTrue $?

    git diff --exit-code . > /dev/null

    assertTrue "Files should have not been modified but were" $?
}

. ./vendor/shunit2/shunit2
