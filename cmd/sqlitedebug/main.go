package main

import (
	"database/sql"
	"fmt"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var extensionPath = "/Users/seanmcgary/Code/sidecar/sqlite-extensions/yolo.dylib"

func main() {
	sql.Register("sqlite3_with_extensions", &sqlite3.SQLiteDriver{
		Extensions: []string{extensionPath},
	})
	dialector := gorm.Dialector(sqlite.Dialector{
		DSN:        "file:yourdatabase.db?cache=shared&_fk=1",
		DriverName: "sqlite3_with_extensions",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	var r string
	res := db.Raw("SELECT pre_nile_tokens_per_day('500') as result").Scan(&r)
	if res.Error != nil {
		panic(fmt.Sprintf("failed to load extension: %v", err))
	}
	fmt.Printf("Result: %s\n", r)
}
