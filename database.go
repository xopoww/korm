package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

// main database function
func db_main() {
	db, err := sql.Open("sqlite3", "./korm.db")
	if err != nil {
		dbLogger.Fatalf("Error opening a database: %s", err)
		return
	}
	defer db.Close()
}
