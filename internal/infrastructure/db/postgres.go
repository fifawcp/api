package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

func NewPostgresDB(
	addr string,
	maxOpenConns int,
	maxIdleConns int,
	maxIdleTime time.Duration,
) (*sql.DB, error) {
	db, err := sql.Open("postgres", addr)
	if err != nil {
		return nil, err
	}

	// Create a context with a timeout to limit the time we wait for the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping the database to verify the connection
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	// Set database connection limits
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxIdleTime(maxIdleTime)

	return db, nil
}
