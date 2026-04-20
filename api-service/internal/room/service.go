// Package room provides HTTP handlers, service logic, and repository access
// for the rooms domain. Capability queries live here because they are a
// room-level concern — moving them to device would create a circular import.
package room

import (
	"context"
	"time"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/events"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// indefiniteOverride is stored in manual_override_until when the user requests
// an indefinite manual override. The control loop's expiry check (now > until)
// will never trigger for this value within any reasonable system lifetime.
var indefiniteOverride = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

type Service struct {
	rooms *Repository
	rdb   *redis.Client
}

func NewService(rooms *Repository, rdb *redis.Client) *Service {
	return &Service{rooms: rooms, rdb: rdb}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]models.Room, error) {
	return s.rooms.List(ctx, userID)
}

func (s *Service) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.Room, error) {
	return s.rooms.GetByIDAndUserID(ctx, id, userID)
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, name string) (*models.Room, error) {
	rm := models.Room{
		UserID: userID,
		Name:   name,
	}
	if err := s.rooms.CreateWithDesiredState(ctx, &rm); err != nil {
		return nil, err
	}
	events.NotifyRoomCreated(ctx, s.rdb, rm.ID)
	return &rm, nil
}

func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, name string, deadbandTemp, deadbandHum *float64) (*models.Room, error) {
	rm, err := s.rooms.GetByIDAndUserID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	rm.Name = name
	if deadbandTemp != nil {
		rm.DeadbandTemp = *deadbandTemp
	}
	if deadbandHum != nil {
		rm.DeadbandHum = *deadbandHum
	}

	if err := s.rooms.Update(ctx, rm); err != nil {
		return nil, err
	}
	events.NotifyRoomConfigChanged(ctx, s.rdb, rm.ID)
	return rm, nil
}

func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	// GetByIDAndUserID returns ErrNotFound if the room exists but belongs
	// to a different user — ownership gate without leaking existence.
	if _, err := s.rooms.GetByIDAndUserID(ctx, id, userID); err != nil {
		return err
	}
	if err := s.rooms.Delete(ctx, id); err != nil {
		return err
	}
	events.NotifyRoomDeleted(ctx, s.rdb, id)
	return nil
}

// GetDesiredState returns the desired state for a room the user owns.
func (s *Service) GetDesiredState(ctx context.Context, roomID, userID uuid.UUID) (models.DesiredState, error) {
	if _, err := s.rooms.GetByIDAndUserID(ctx, roomID, userID); err != nil {
		return models.DesiredState{}, err
	}
	return s.rooms.GetDesiredState(ctx, roomID)
}

// UpdateDesiredStateInput carries the parsed, validated fields from the handler.
// ManualOverrideUntil is pre-resolved: nil = clear override, non-nil = set override.
// The "indefinite" sentinel is resolved to indefiniteOverride by the handler before
// calling this method.
type UpdateDesiredStateInput struct {
	Mode                string
	TargetTemp          *float64
	TargetHum           *float64
	ManualOverrideUntil *time.Time
}

// UpdateDesiredState validates capability requirements and persists the new state.
func (s *Service) UpdateDesiredState(ctx context.Context, roomID, userID uuid.UUID, input UpdateDesiredStateInput) (models.DesiredState, error) {
	if _, err := s.rooms.GetByIDAndUserID(ctx, roomID, userID); err != nil {
		return models.DesiredState{}, err
	}

	if input.Mode == "AUTO" {
		if input.TargetTemp == nil && input.TargetHum == nil {
			return models.DesiredState{}, ErrInvalidState
		}

		if input.TargetTemp != nil {
			ok, err := s.rooms.HasTemperatureCapability(ctx, roomID)
			if err != nil {
				return models.DesiredState{}, err
			}
			if !ok {
				return models.DesiredState{}, ErrNoCapability
			}
		}

		if input.TargetHum != nil {
			ok, err := s.rooms.HasHumidityCapability(ctx, roomID)
			if err != nil {
				return models.DesiredState{}, err
			}
			if !ok {
				return models.DesiredState{}, ErrNoCapability
			}
		}
	}

	ds, err := s.rooms.GetDesiredState(ctx, roomID)
	if err != nil {
		return models.DesiredState{}, err
	}

	ds.Mode = input.Mode
	ds.TargetTemp = input.TargetTemp
	ds.TargetHum = input.TargetHum
	ds.ManualOverrideUntil = input.ManualOverrideUntil

	if err := s.rooms.UpdateDesiredState(ctx, &ds); err != nil {
		return models.DesiredState{}, err
	}
	events.NotifyDesiredStateChanged(ctx, s.rdb, roomID)
	return ds, nil
}
