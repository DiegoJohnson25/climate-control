// Package connect establishes connections to the infrastructure components
// api-service depends on: PostgreSQL (appdb), TimescaleDB (metricsdb), and
// Redis. Internal Docker hostnames and ports are hardcoded — env var ports
// are host-machine mappings only and never used inside Docker.
package connect

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Postgres opens a GORM connection to the appdb PostgreSQL instance.
func Postgres(user, password, dbName string) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=postgres user=%s password=%s dbname=%s port=5432 sslmode=disable",
		user, password, dbName,
	)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
