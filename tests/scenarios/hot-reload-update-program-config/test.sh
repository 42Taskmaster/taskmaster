#!/usr/bin/env sh

taskmasterd -c ./taskmaster-init.yaml 2> /dev/null

strest hot-reload-update-program-config.strest.yml
