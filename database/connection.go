package database

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
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

type Config struct {
	Filename	string
	// Filename/path to the text file with the SQL script that needs to be executed at the start.
	// If InitScript is empty string, it is ignored.
	InitScript	string
	Logger		*logrus.Logger
}

// Global variable used internally throughout the package.
var db *DB

// 	Initialize a database handle.
// Opens and pings a database and executes a starting script. If at any step an error is encountered,
// Start will panic. A Close function must be called when working with database is finished.
func Start(cfg *Config) {
	// open and ping a database
	handle := sqlx.MustConnect("sqlite3", cfg.Filename)
	logger := cfg.Logger
	if logger == nil {
		logger = &logrus.Logger{}
	}

	db = &DB{
		handle,
		logger,
	}
	db.Info("Opened a database.")

	if cfg.InitScript != "" {
		// open, read and execute an initial script
		file, err := os.Open(cfg.InitScript)
		if err != nil {
			db.Panicf("Could not open %s: %s", cfg.InitScript, err)
		}
		script, err := ioutil.ReadAll(file)
		if err != nil {
			db.Panicf("Could not read %s: %v", cfg.InitScript, err)
		}
		_, err = db.Exec(string(script))
		if err != nil {
			db.Panicf("Could not execute init script: %s", err)
		}
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