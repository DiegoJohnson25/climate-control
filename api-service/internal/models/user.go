// Package models defines the GORM structs for api-service. These structs are
// the schema contract for appdb — migrations are the source of truth, and the
// tags here exist only for GORM round-trips.
package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email        string
	PasswordHash string
	Timezone     string `gorm:"default:UTC"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
