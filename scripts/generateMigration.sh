#!/usr/bin/env bash

name=$1

if [[ -z $name ]]; then
    echo "Usage: $0 <migration_name>"
    exit 1
fi

timestamp=$(date +"%Y%m%d%H%M")

migration_name="${timestamp}_${name}"

migrations_dir="./pkg/postgres/migrations/${migration_name}"
migration_file="${migrations_dir}/up.go"

mkdir -p $migrations_dir || true

# heredoc that creates a migration go file with an Up function
cat > $migration_file <<EOF
package _${timestamp}_${name}

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	return nil
}

func (m *Migration) GetName() string {
	return "${timestamp}_${name}"
}
EOF
