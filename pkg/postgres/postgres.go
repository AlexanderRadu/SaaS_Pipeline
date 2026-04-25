package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxConns          = 10
	defaultMaxConnLifetime   = time.Hour
	defaultMaxConnIdleTime   = 30 * time.Minute
	defaultConnectionTimeout = 5 * time.Second

	defaultMaxRetries = 5
	defaultRetryDelay = 2 * time.Second
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

	for _, o := range opts {
		if o.MaxConns > 0 {
			opt.MaxConns = o.MaxConns
		}
		if o.MaxConnLifetime > 0 {
			opt.MaxConnLifetime = o.MaxConnLifetime
		}
		if o.MaxConnIdleTime > 0 {
			opt.MaxConnIdleTime = o.MaxConnIdleTime
		}
		if o.ConnectionTimeout > 0 {
			opt.ConnectionTimeout = o.ConnectionTimeout
		}
	}

	applyOptions(cfg, opt)

	var pool *pgxpool.Pool

	for i := 0; i < defaultMaxRetries; i++ {
		connectCtx, cancel := context.WithTimeout(ctx, opt.ConnectionTimeout)

		pool, err = pgxpool.NewWithConfig(connectCtx, cfg)
		if err == nil {
			err = pool.Ping(connectCtx)
		}
		cancel()

		if err == nil {
			return pool, nil
		}

		log.Printf("Failed to connect to database (attempt %d/%d), retrying in %s: %v\n", i+1, defaultMaxRetries, defaultRetryDelay, err)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("postgres.NewPool: context canceled during retries: %w", ctx.Err())
		case <-time.After(defaultRetryDelay):
		}
	}

	return nil, fmt.Errorf("postgres.NewPool: exhausted %d retries: %w", defaultMaxRetries, err)
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
	cfg.MaxConns = opts.MaxConns
	cfg.MaxConnLifetime = opts.MaxConnLifetime
	cfg.MaxConnIdleTime = opts.MaxConnIdleTime
	cfg.PingTimeout = opts.ConnectionTimeout
}
