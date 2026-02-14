package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func ConnectDB() {
	dbURL := os.Getenv("DB_SOURCE")
	if dbURL == "" {
		log.Fatal("DB_SOURCE environment variable is not set")
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Unable to parse DB URL: %v", err)
	}

	// Connection Pool Settings
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	// Try to connect with retries
	count := 0
	for {
		DB, err = pgxpool.NewWithConfig(context.Background(), config)
		if err != nil {
			log.Printf("Postgres not ready... waiting (attempt %d)", count+1)
			count++
		} else {
			// Verify connection
			err = DB.Ping(context.Background())
			if err == nil {
				log.Println("Successfully connected to PostgreSQL")
				return
			}
			log.Printf("Ping failed... waiting (attempt %d): %v", count+1, err)
		}

		if count > 10 {
			log.Fatalf("Could not connect to database after 10 attempts: %v", err)
		}

		time.Sleep(2 * time.Second)
	}
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
