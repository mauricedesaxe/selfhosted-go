package auth

import (
	"go-on-rails/common"
	"log"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
)

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
	Store = session.New(session.Config{
		Storage: common.Storage,
	})

	// create tables
	_, err := common.Db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	_, err = common.Db.Exec(`CREATE TABLE IF NOT EXISTS password_resets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	_, err = common.Db.Exec(`CREATE TABLE IF NOT EXISTS signup_codes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		uses INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	_, err = common.Db.Exec(`CREATE TABLE IF NOT EXISTS user_roles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		role TEXT NOT NULL, 
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	// indexes
	_, err = common.Db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_email ON users (email)`)
	if err != nil {
		log.Fatalf("Error creating idx_users_email: %v", err)
	}

	// seed the database with a signup code, only if no user has signed up yet
	_, err = common.Db.Exec(`
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
