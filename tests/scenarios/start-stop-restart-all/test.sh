#!/usr/bin/env sh

taskmasterd 2> /dev/null

strest start-stop-restart-all.strest.yml
