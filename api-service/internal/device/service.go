// Package device provides HTTP handlers, service logic, and repository access
// for the devices domain. Capability-conflict checks for active schedules
// live here permanently — the device package owns the logic.
package device

import (
	"context"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/room"
	"github.com/DiegoJohnson25/climate-control/shared/models"
	"github.com/google/uuid"
)

var (
	validSensorTypes   = map[string]bool{"temperature": true, "humidity": true, "air_quality": true}
	validActuatorTypes = map[string]bool{"heater": true, "humidifier": true}
)

type Service struct {
	devices *Repository
	rooms   *room.Repository
}

func NewService(devices *Repository, rooms *room.Repository) *Service {
	return &Service{devices: devices, rooms: rooms}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]DeviceWithCapabilities, error) {
	return s.devices.List(ctx, userID)
}

func (s *Service) ListByRoom(ctx context.Context, roomID, userID uuid.UUID) ([]DeviceWithCapabilities, error) {
	if _, err := s.rooms.GetByIDAndUserID(ctx, roomID, userID); err != nil {
		return nil, ErrRoomNotFound
	}
	return s.devices.ListByRoom(ctx, roomID)
}

func (s *Service) GetByID(ctx context.Context, id, userID uuid.UUID) (*DeviceWithCapabilities, error) {
	return s.devices.GetByIDAndUserID(ctx, id, userID)
}

// CreateInput carries validated fields from the handler.
type CreateInput struct {
	Name          string
	HwID          string
	DeviceType    string
	SensorTypes   []string
	ActuatorTypes []string
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, input CreateInput) (*DeviceWithCapabilities, error) {
	seen := make(map[string]bool)
	for _, st := range input.SensorTypes {
		if !validSensorTypes[st] {
			return nil, ErrInvalidSensor
		}
		if seen[st] {
			return nil, ErrDuplicateSensor
		}
		seen[st] = true
	}

	seen = make(map[string]bool)
	for _, at := range input.ActuatorTypes {
		if !validActuatorTypes[at] {
			return nil, ErrInvalidActuator
		}
		if seen[at] {
			return nil, ErrDuplicateActuator
		}
		seen[at] = true
	}

	if err := s.devices.CheckHwIDAvailability(ctx, input.HwID, userID); err != nil {
		return nil, err
	}
	dev := models.Device{
		UserID:     userID,
		Name:       input.Name,
		HwID:       input.HwID,
		DeviceType: input.DeviceType,
	}
	if err := s.devices.Create(ctx, &dev, input.SensorTypes, input.ActuatorTypes); err != nil {
		return nil, err
	}

	return s.devices.GetByIDAndUserID(ctx, dev.ID, userID)
}

// UpdateInput carries validated fields from the handler.
type UpdateInput struct {
	Name   string
	RoomID *uuid.UUID // nil = unassign from room
}

func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, input UpdateInput) (*DeviceWithCapabilities, error) {
	dev, err := s.devices.GetByIDAndUserID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if input.RoomID != nil {
		if _, err := s.rooms.GetByIDAndUserID(ctx, *input.RoomID, userID); err != nil {
			return nil, ErrRoomNotFound
		}
	}

	// device is leaving its current room — check that nothing depends on it
	leavingRoom := dev.RoomID != nil && (input.RoomID == nil || *input.RoomID != *dev.RoomID)

	if leavingRoom {
		conflict, err := s.devices.HasCapabilityConflictAfterRemoval(ctx, *dev.RoomID, id)
		if err != nil {
			return nil, err
		}
		if conflict {
			return nil, ErrCapabilityConflict
		}
	}

	dev.Device.Name = input.Name
	dev.Device.RoomID = input.RoomID

	if err := s.devices.Update(ctx, &dev.Device); err != nil {
		return nil, err
	}
	return s.devices.GetByIDAndUserID(ctx, id, userID)
}

func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	dev, err := s.devices.GetByIDAndUserID(ctx, id, userID)
	if err != nil {
		return err
	}

	if dev.RoomID != nil {
		conflict, err := s.devices.HasCapabilityConflictAfterRemoval(ctx, *dev.RoomID, id)
		if err != nil {
			return err
		}
		if conflict {
			return ErrCapabilityConflict
		}
	}
	return s.devices.Delete(ctx, id)
}
