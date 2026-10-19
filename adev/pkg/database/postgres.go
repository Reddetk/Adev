// Package database opens and configures the PostgreSQL connection pool.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config describes the connection pool.
type Config struct {
	DSN         string
	MaxConns    int32
	MaxIdleTime time.Duration
	MaxLifetime time.Duration
}

// Open creates a pool and checks the connection.
func Open(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if cfg.MaxConns > 0 {
		poolConfig.MaxConns = cfg.MaxConns
	}
	if cfg.MaxIdleTime > 0 {
		poolConfig.MaxConnIdleTime = cfg.MaxIdleTime
	}
	if cfg.MaxLifetime > 0 {
		poolConfig.MaxConnLifetime = cfg.MaxLifetime
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
