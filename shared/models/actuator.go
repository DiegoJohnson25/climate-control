package models

import (
	"time"

	"github.com/google/uuid"
)

type Actuator struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DeviceID     uuid.UUID
	ActuatorType string
	CreatedAt    time.Time
}
