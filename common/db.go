package common

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/storage/sqlite3"
	"github.com/jmoiron/sqlx"
)

var Db *sqlx.DB
var Storage *sqlite3.Storage

func init() {
	// make sure the db folder exists
	err := os.MkdirAll("./db", os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating db folder: %v", err)
	}

	InitDB(fmt.Sprintf("./db/%s.db", Env.ENVIRONMENT))
}

func InitDB(dbPath string) {
	var err error
	Db, err = sqlx.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	optimizationStmts := `
    PRAGMA journal_mode = WAL;
    PRAGMA synchronous = NORMAL;
    PRAGMA cache_size = -320000;  -- 320MB
    PRAGMA temp_store = MEMORY;`
	_, err = Db.Exec(optimizationStmts)
	if err != nil {
		log.Fatalf("Error optimizing database: %v", err)
	}

	Storage = sqlite3.New(sqlite3.Config{
		Database: dbPath,
	})
}
