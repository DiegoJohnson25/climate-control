package models

import (
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID
	Name         string
	DeadbandTemp float64 `gorm:"default:1.5"`
	DeadbandHum  float64 `gorm:"default:5.0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
