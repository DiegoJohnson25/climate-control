package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type SchedulePeriod struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ScheduleID uuid.UUID
	Name       *string
	DaysOfWeek pq.Int64Array `gorm:"type:integer[]"`
	StartTime  time.Time
	EndTime    time.Time
	Mode       string `gorm:"default:OFF"`
	TargetTemp *float64
	TargetHum  *float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
