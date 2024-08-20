#!/usr/bin/env bash

DATABASE=sidecar_test

echo 'Dropping database'
dropdb $DATABASE || true

echo 'Creating database'
createdb $DATABASE


if [[ -f ./sql/schema.sql ]];
then
    echo 'Loading schema'
    psql --quiet $DATABASE < ./sql/schema.sql
fi

echo 'Bootstrapping database'
go run cmd/migrate/main.go
if [[ "$?" -ne 0 ]];
then
    echo "Failed to bootstrap database"
    exit 1
else
    echo "Database bootstrapped"
fi

echo "Dumping updated DB schema"
pg_dump --schema-only --no-owner --dbname $DATABASE > ./sql/schema.sql
pg_dump --data-only --inserts --table='migrations' --dbname $DATABASE >> ./sql/schema.sql

echo "Dump complete"
