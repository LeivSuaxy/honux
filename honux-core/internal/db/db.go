package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Database struct {
	*sql.DB
}

var (
	instance *Database
	once     sync.Once
	initErr  error
)

// Returns a DB Instance (Singleton Pattern)
func GetDb(dsn string) (*Database, error) {
	once.Do(func() {
		db, err := connect(dsn)
		if err != nil {
			initErr = err
			return
		}
		instance = &Database{db}
	})
	return instance, initErr
}

func connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	return db, nil
}
