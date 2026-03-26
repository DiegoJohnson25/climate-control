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
