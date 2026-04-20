// Package database provides connection helpers for appdb (GORM) and
// metricsdb (pgx/TimescaleDB). The Docker hostnames and internal ports are
// hardcoded; the .env port mappings are for the host machine only.
package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectPostgres opens a GORM connection to appdb using the internal Docker
// hostname "postgres" on port 5432.
func ConnectPostgres(user, password, dbName string) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=postgres user=%s password=%s dbname=%s port=5432 sslmode=disable",
		user, password, dbName,
	)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
