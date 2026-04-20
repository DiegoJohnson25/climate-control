package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectTimescale opens a pgx pool to metricsdb using the internal Docker
// hostname "timescaledb" on port 5432.
func ConnectTimescale(user, password, dbName string) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=timescaledb user=%s password=%s dbname=%s port=5432 sslmode=disable",
		user, password, dbName,
	)
	return pgxpool.New(context.Background(), dsn)
}
