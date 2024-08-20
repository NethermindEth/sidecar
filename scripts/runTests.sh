#!/usr/bin/env bash

export SIDECAR_POSTGRES_HOST=localhost
export SIDECAR_POSTGRES_PORT=5432
export SIDECAR_POSTGRES_USER="seanmcgary"
export SIDECAR_POSTGRES_PASSWORD=""
export SIDECAR_POSTGRES_DBNAME=sidecar_test

echo 'Prepping database'
./tests/bin/bootstrapDb.sh

echo 'Running tests'
TESTING=true go test -v -p 1 ./...
