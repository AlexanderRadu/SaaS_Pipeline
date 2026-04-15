package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxConns          = 10
	defaultMaxConnLifetime   = time.Hour
	defaultMaxConnIdleTime   = 30 * time.Minute
	defaultConnectionTimeout = 5 * time.Second
)

type Options struct {
	MaxConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	ConnectionTimeout time.Duration
}

func NewPool(ctx context.Context, dsn string, opts ...Options) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres.NewPool: %w", err)
	}

	opt := defaultOptions()
	if len(opts) > 0 {
		applyOptions(cfg, opts[0])
	} else {
		applyOptions(cfg, opt)
	}

	connectCtx, cancel := context.WithTimeout(ctx, defaultConnectionTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres.NewPool: %w", err)
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres.NewPool: %w", err)
	}
	return pool, nil
}

func defaultOptions() Options {
	return Options{
		MaxConns:          defaultMaxConns,
		MaxConnLifetime:   defaultMaxConnLifetime,
		MaxConnIdleTime:   defaultMaxConnIdleTime,
		ConnectionTimeout: defaultConnectionTimeout,
	}
}

func applyOptions(cfg *pgxpool.Config, opts Options) {
	if opts.MaxConns > 0 {
		cfg.MaxConns = opts.MaxConns
	}

	if opts.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = opts.MaxConnLifetime
	}

	if opts.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = opts.MaxConnIdleTime
	}

	if opts.ConnectionTimeout > 0 {
		cfg.PingTimeout = opts.MaxConnLifetime
	}
}
