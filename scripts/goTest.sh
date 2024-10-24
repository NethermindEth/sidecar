#!/usr/bin/env bash

export PROJECT_ROOT=$(pwd)
export TESTING=true
export SIDECAR_DATABASE_HOST="localhost"
export SIDECAR_DATABASE_PORT="5432"
export SIDECAR_DATABASE_USER=""
export SIDECAR_DATABASE_PASSWORD=""

go test $@
