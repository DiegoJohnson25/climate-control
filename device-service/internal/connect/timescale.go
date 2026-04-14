package connect

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Timescale(user, password, dbName string) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=timescaledb user=%s password=%s dbname=%s port=5432 sslmode=disable",
		user, password, dbName,
	)
	return pgxpool.New(context.Background(), dsn)
}
