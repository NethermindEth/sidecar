#!/usr/bin/env bash

export SIDE_CAR_POSTGRES_HOST=localhost
export SIDE_CAR_POSTGRES_PORT=5432
export SIDE_CAR_POSTGRES_USER=""
export SIDE_CAR_POSTGRES_PASSWORD=""
export SIDE_CAR_POSTGRES_DBNAME=blocklake_test

echo 'Prepping database'
./tests/bin/bootstrapDb.sh

echo 'Running tests'
TESTING=true go test -v -p 1 ./...
