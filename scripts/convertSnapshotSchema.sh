#!/bin/bash

# Check if all required arguments are provided
if [ "$#" -ne 6 ]; then
  echo "Usage: $0 <input_schema_name> <output_schema_name> <input_file> <output_file> <db_user> <db_password>"
  exit 1
fi

INPUT_SCHEMA_NAME=$1
OUTPUT_SCHEMA_NAME=$2
INPUT_FILE=$3
OUTPUT_FILE=$4
DB_USER=$5
DB_PASSWORD=$6

# Generate a unique hash for the temporary database
HASH=$(date +%s | sha256sum | head -c 20)
TEMP_DB_NAME="temp_sidecar_dump_schema_conversion_db_${HASH}"

# Drop the temporary database if it exists to ensure a clean slate
echo "Dropping temporary database if it exists: $TEMP_DB_NAME"
PGPASSWORD=$DB_PASSWORD psql -U $DB_USER -c "DROP DATABASE IF EXISTS $TEMP_DB_NAME;" || { echo "Failed during DROP DATABASE IF EXISTS <temp_db_name>"; exit 1; }

# Create a temporary database
echo "Creating temporary database: $TEMP_DB_NAME"
PGPASSWORD=$DB_PASSWORD psql -U $DB_USER -c "CREATE DATABASE $TEMP_DB_NAME;" || { echo "Failed to create database"; exit 1; }

# Restore the snapshot dump into the temporary database
echo "Restoring snapshot into temporary database"
./bin/sidecar restore-snapshot \
    --database.host=localhost \
    --database.port=5432 \
    --database.user=$DB_USER \
    --database.password=$DB_PASSWORD \
    --database.db_name=$TEMP_DB_NAME \
    --database.schema_name=$INPUT_SCHEMA_NAME \
    --input_file=$INPUT_FILE || { echo "Failed to restore snapshot"; exit 1; }

# Rename the schema in the temporary database
echo "Renaming schema from $INPUT_SCHEMA_NAME to $OUTPUT_SCHEMA_NAME"
PGPASSWORD=$DB_PASSWORD psql -U $DB_USER -d $TEMP_DB_NAME -c "ALTER SCHEMA $INPUT_SCHEMA_NAME RENAME TO $OUTPUT_SCHEMA_NAME;" || { echo "Failed to rename schema"; exit 1; }

# Create a new snapshot with the updated schema
echo "Creating new snapshot with updated schema"
./bin/sidecar create-snapshot \
    --database.host=localhost \
    --database.port=5432 \
    --database.user=$DB_USER \
    --database.password=$DB_PASSWORD \
    --database.db_name=$TEMP_DB_NAME \
    --database.schema_name=$OUTPUT_SCHEMA_NAME \
    --output_file=$OUTPUT_FILE || { echo "Failed to create snapshot"; exit 1; }

# Drop the temporary database
echo "Dropping temporary database: $TEMP_DB_NAME"
PGPASSWORD=$DB_PASSWORD psql -U $DB_USER -c "DROP DATABASE IF EXISTS $TEMP_DB_NAME;" || { echo "Failed to drop database"; exit 1; }

echo "Schema conversion completed successfully. Output saved to $OUTPUT_FILE."
