package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteRepository struct {
	db *sql.DB
}

func Open(ctx context.Context, dataSource string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite", dataSource)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite：%w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接 SQLite：%w", err)
	}
	repository := &SQLiteRepository{db: db}
	if err := repository.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return repository, nil
}

func (r *SQLiteRepository) Close() error { return r.db.Close() }

func (r *SQLiteRepository) DB() *sql.DB { return r.db }
