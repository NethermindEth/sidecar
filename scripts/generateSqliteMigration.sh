#!/usr/bin/env bash

name=$1

if [[ -z $name ]]; then
    echo "Usage: $0 <migration_name>"
    exit 1
fi

timestamp=$(date +"%Y%m%d%H%M")

migration_name="${timestamp}_${name}"

migrations_dir="./internal/sqlite/migrations/${migration_name}"
migration_file="${migrations_dir}/up.go"

mkdir -p $migrations_dir || true

# heredoc that creates a migration go file with an Up function
cat > $migration_file <<EOF
package _${timestamp}_${name}

import (
	"database/sql"
	"gorm.io/gorm"
)

type SqliteMigration struct {
}

func (m *SqliteMigration) Up(grm *gorm.DB) error {
	return nil
}

func (m *SqliteMigration) GetName() string {
	return "${timestamp}_${name}"
}
EOF
