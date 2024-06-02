package auth

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/sqlite3"
	"github.com/jmoiron/sqlx"
)

var AuthDb *sqlx.DB
var Store *session.Store

type UserMetadata struct {
	ID        int       `db:"id"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
}

type SignupCode struct {
	Code      string    `db:"code"`
	Uses      int       `db:"uses"`
	CreatedAt time.Time `db:"created_at"`
}

func init() {
	storage := sqlite3.New(sqlite3.Config{
		Database: "./db/auth.db",
	})
	Store = session.New(session.Config{
		Storage: storage,
	})

	var err error
	AuthDb, err = sqlx.Open("sqlite3", "./db/auth.db")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	// optimize the database
	optimizationStmts := `
    PRAGMA journal_mode = WAL;
    PRAGMA synchronous = NORMAL;
    PRAGMA cache_size = -64000;  -- 64MB
    PRAGMA temp_store = MEMORY;`
	_, err = AuthDb.Exec(optimizationStmts)
	if err != nil {
		log.Fatalf("Error optimizing database: %v", err)
	}

	// create tables
	_, err = AuthDb.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	_, err = AuthDb.Exec(`CREATE TABLE IF NOT EXISTS password_resets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	_, err = AuthDb.Exec(`CREATE TABLE IF NOT EXISTS signup_codes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		uses INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	_, err = AuthDb.Exec(`CREATE TABLE IF NOT EXISTS user_roles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		role TEXT NOT NULL, 
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	// indexes
	_, err = AuthDb.Exec(`CREATE INDEX IF NOT EXISTS idx_users_email ON users (email)`)
	if err != nil {
		log.Fatalf("Error creating idx_users_email: %v", err)
	}

	// seed the database with a signup code, only if no user has signed up yet
	_, err = AuthDb.Exec(`
		INSERT INTO signup_codes (code, uses)
		SELECT 'fresh', 1
		WHERE NOT EXISTS (
			SELECT 1 FROM users
		)
	`)
	if err != nil {
		log.Printf("Error seeding database: %v", err)
	}
}
