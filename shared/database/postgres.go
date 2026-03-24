package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectPostgres(user, password, dbName string) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=postgres user=%s password=%s dbname=%s port=5432 sslmode=disable",
		user, password, dbName,
	)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
