package models

import (
	"time"

	"github.com/google/uuid"
)

type Schedule struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RoomID    uuid.UUID
	UserID    uuid.UUID
	Name      string
	IsActive  bool `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
