package connect

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Timescale opens a pgx connection pool to the metricsdb TimescaleDB instance.
func Timescale(user, password, dbName string) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=timescaledb user=%s password=%s dbname=%s port=5432 sslmode=disable",
		user, password, dbName,
	)
	return pgxpool.New(context.Background(), dsn)
}
