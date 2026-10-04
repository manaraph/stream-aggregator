package storage

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/manaraph/stream-aggregator/internal/domain"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Postgres struct {
	pool *pgxpool.Pool
}

func OpenPostgres(ctx context.Context, url string) (*Postgres, error) {
	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL is not configured")
	}

	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL configuration: %w", err)
	}
	config.MaxConns = 8

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	schema, err := migrations.ReadFile("migrations/001_sensor_readings.sql")
	if err == nil {
		_, err = pool.Exec(ctx, string(schema))
	}
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply sensor schema: %w", err)
	}

	return &Postgres{pool: pool}, nil
}

func (p *Postgres) InsertBatch(ctx context.Context, readings []domain.Sensor) error {
	if len(readings) == 0 {
		return nil
	}

	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin sensor batch: %w", err)
	}
	defer tx.Rollback(ctx)

	batch := &pgx.Batch{}
	for _, reading := range readings {
		batch.Queue(`
			INSERT INTO sensor_readings
				(event_id, sensor_id, measurement_type, unit, value, event_time)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (event_id) DO NOTHING`,
			reading.EventID, reading.Sensor, reading.MeasurementType, reading.Unit,
			reading.Value, reading.Timestamp.UTC())
	}

	results := tx.SendBatch(ctx, batch)
	for range readings {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return fmt.Errorf("insert sensor batch: %w", err)
		}
	}
	if err := results.Close(); err != nil {
		return fmt.Errorf("finish sensor batch: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit sensor batch: %w", err)
	}
	return nil
}

func (p *Postgres) Close() {
	p.pool.Close()
}
