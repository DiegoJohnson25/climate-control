package models

import (
	"time"

	"github.com/google/uuid"
)

type Device struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     uuid.UUID
	RoomID     *uuid.UUID
	Name       string
	HwID       string `gorm:"column:hw_id"`
	DeviceType string `gorm:"default:physical"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
