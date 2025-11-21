package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func Connect(dsn string) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				log.Println("Successfully connected to the database")
				return db, nil
			}
		}
		log.Printf("Failed to connect to DB: %v. Retrying in 5 seconds...", err)
		time.Sleep(5 * time.Second)
	}

	return nil, fmt.Errorf("could not connect to database after retries: %w", err)
}

func Migrate(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS teams (
            team_name TEXT PRIMARY KEY
        );`,
		`CREATE TABLE IF NOT EXISTS users (
            user_id TEXT PRIMARY KEY,
            username TEXT NOT NULL,
            team_name TEXT REFERENCES teams(team_name) ON DELETE CASCADE,
            is_active BOOLEAN DEFAULT TRUE
        );`,
		`CREATE TABLE IF NOT EXISTS pull_requests (
            pull_request_id TEXT PRIMARY KEY,
            pull_request_name TEXT NOT NULL,
            author_id TEXT REFERENCES users(user_id) ON DELETE CASCADE,
            status TEXT NOT NULL DEFAULT 'OPEN',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            merged_at TIMESTAMP
        );`,
		`CREATE TABLE IF NOT EXISTS pr_reviewers (
            pull_request_id TEXT REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
            reviewer_id TEXT REFERENCES users(user_id) ON DELETE CASCADE,
            PRIMARY KEY (pull_request_id, reviewer_id)
        );`,
	}

	for _, q := range queries {
		_, err := db.Exec(q)
		if err != nil {
			return fmt.Errorf("migration failed on query: %s, error: %w", q, err)
		}
	}

	log.Println("Database migration completed successfully")
	return nil
}
