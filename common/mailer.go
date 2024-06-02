package common

import (
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"

	"github.com/jmoiron/sqlx"
)

// This file is responsible for handling the mailer configuration and sending emails.
// It uses the sqlite3 database to store the mailer configuration such that
// the user who deploys the app can configure the mailer even if they are not technical
// (i.e. doesn't know how to set env vars)
//
// This creates a potential case where the mailer is not configured, so
// every part of the codebase that makes a call to send an email will have to check
// if the mailer is configured before sending an email and if not, return an error.
//
// Another important thing to note is that whenever the user changes the mailer configuration,
// we will need to update the database & re-instantiate the mailer.
//
// This is a tradeoff we are willing to make to optimize for self-hosting.

var Mailer *MailerT
var MailDb *sqlx.DB

func init() {
	var err error
	MailDb, err = sqlx.Open("sqlite3", "./db/mail.db")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	// optimize the database
	optimizationStmts := `
    PRAGMA journal_mode = WAL;
    PRAGMA synchronous = NORMAL;
    PRAGMA cache_size = -64000;  -- 64MB
    PRAGMA temp_store = MEMORY;`
	_, err = MailDb.Exec(optimizationStmts)
	if err != nil {
		log.Fatalf("Error optimizing database: %v", err)
	}

	_, err = MailDb.Exec(`
	CREATE TABLE IF NOT EXISTS mailer_config (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		host TEXT,
		port INTEGER,
		username TEXT,
		password TEXT
	)`)
	if err != nil {
		log.Fatalf("Error creating mailer_config table: %v", err)
	}

	// Load the mailer configuration from the database
	// If the mailer is not configured, we will just return
	err = MailDb.Get(&Mailer, "SELECT * FROM mailer_config LIMIT 1")
	if err != nil {
		log.Printf("Error getting mailer configuration: %v", err)
		return
	}

	Mailer = &MailerT{
		Host:     Mailer.Host,
		Port:     Mailer.Port,
		Username: Mailer.Username,
		Password: Mailer.Password,
	}
}

// Updates the mailer configuration in the database
// and instantiates the new mailer.
// Example:
//
//	NewMailer(&MailerT{
//		Host:     "smtp.example.com",
//		Port:     587,
//		Username: "username",
//		Password: "password",
//	})
func NewMailer(config *MailerT) error {
	_, err := MailDb.Exec(`
	INSERT INTO mailer_config (id, host, port, username, password) VALUES (1, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
	host = excluded.host,
	port = excluded.port,
	username = excluded.username,
	password = excluded.password`, config.Host, config.Port, config.Username, config.Password)
	if err != nil {
		return err
	}
	Mailer = &MailerT{
		Host:     Mailer.Host,
		Port:     Mailer.Port,
		Username: Mailer.Username,
		Password: Mailer.Password,
	}
	return nil
}

func IsValidMailer(config *MailerT) bool {
	// basic existence check
	if config.Host == "" || config.Port == 0 || config.Username == "" || config.Password == "" {
		return false
	}

	// is Host a valid IP address
	if net.ParseIP(config.Host) == nil {
		return false
	}

	// check if the port is valid (1-65535)
	if config.Port < 1 || config.Port > 65535 {
		return false
	}

	return true
}

type MailerT struct {
	Host     string // The hostname of the SMTP server
	Port     int    // The port number of the SMTP server
	Username string // The username to use for authentication
	Password string // The password to use for authentication
}

// Sends an email to the specified recipient(s) with the specified subject and body.
func (m *MailerT) SendMail(to []string, subject, body string) error {
	auth := smtp.PlainAuth("", m.Username, m.Password, m.Host)
	msg := []byte("To: " + strings.Join(to, ",") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")
	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	return smtp.SendMail(addr, auth, m.Username, to, msg)
}
