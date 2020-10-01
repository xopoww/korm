package database

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	dbName = "korm.db"
	initScript = "database_creation.sql"
)

// DB is a combination of embedded database and logger
// made for dependency injection
type DB struct {
	*sqlx.DB
	*logrus.Logger
}

// Global variable used internally throughout the package.
var db *DB

// 	Initialize a database handle.
// Opens and pings a database and executes a starting script. If at any step an error is encountered,
// Start will panic. A Close function must be called when working with database is finished.
func Start(logger *logrus.Logger) {
	// open and ping a database
	handle := sqlx.MustConnect("sqlite3", filepath.Join("..", dbName))
	db = &DB{
		handle,
		logger,
	}
	db.Info("Opened a database.")

	// open, read and execute an initial script
	file, err := os.Open("database_creation.sql")
	if err != nil {
		db.Fatalf("Could not open %s: %s", initScript, err)
	}
	script, err := ioutil.ReadAll(file)
	if err != nil {
		db.Fatalf("Could not read %s: %v", initScript, err)
	}
	_, err = db.Exec(string(script))
	if err != nil {
		db.Fatalf("Could not execute init script: %s", err)
	}
	db.Info("Initialized a database.")
}

func StartWorkers() {
	orderWorker()
}

func Close() {
	err := db.Close()
	if err != nil {
		panic(err)
	}
	db.Info("Closed a database.")
}