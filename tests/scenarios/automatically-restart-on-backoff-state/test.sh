#!/usr/bin/env sh

taskmasterd 2> /dev/null

strest automatically-restart-on-backoff-state.strest.yml
