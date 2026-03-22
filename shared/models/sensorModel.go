package models

import (
	"time"

	"github.com/google/uuid"
)

type Sensor struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DeviceID        uuid.UUID
	MeasurementType string
	CreatedAt       time.Time
}
