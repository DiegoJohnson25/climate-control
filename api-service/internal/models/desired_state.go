package models

import (
	"time"

	"github.com/google/uuid"
)

type DesiredState struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RoomID              uuid.UUID
	Mode                string `gorm:"default:OFF"`
	ManualActive        bool
	TargetTemp          *float64
	TargetHum           *float64
	ManualOverrideUntil *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

