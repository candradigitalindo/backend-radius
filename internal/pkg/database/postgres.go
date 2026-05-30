package database

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/config"
)

func Connect(cfg *config.Config) *pgxpool.Pool {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		log.Fatalf("Unable to parse database config: %v", err)
	}

	poolCfg.MaxConns = int32(cfg.DB.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.DB.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.DB.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	return pool
}
