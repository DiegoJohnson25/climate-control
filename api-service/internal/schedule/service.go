// Package schedule provides HTTP handlers, service logic, and repository access
// for the schedules and schedule_periods domain. Schedules are inactive on
// create; capability validation runs only at activation time. Period
// create/update/delete only emits stream events when the parent schedule is
// active.
package schedule

import (
	"context"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/room"
	"github.com/DiegoJohnson25/climate-control/shared/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Service struct {
	schedules *Repository
	rooms     *room.Repository
}

func NewService(schedules *Repository, rooms *room.Repository) *Service {
	return &Service{schedules: schedules, rooms: rooms}
}

// ---------------------------------------------------------------------------
// Schedule methods
// ---------------------------------------------------------------------------

// Create creates a new inactive schedule under the given room.
// Verifies room ownership before creating.
func (s *Service) Create(ctx context.Context, roomID, userID uuid.UUID, name string) (*models.Schedule, error) {
	if _, err := s.rooms.GetByIDAndUserID(ctx, roomID, userID); err != nil {
		return nil, err
	}

	sched := &models.Schedule{
		Name:     name,
		RoomID:   roomID,
		UserID:   userID,
		IsActive: false,
	}

	if err := s.schedules.Create(ctx, sched); err != nil {
		return nil, err
	}

	return sched, nil
}

// GetByID returns the schedule with the given ID owned by userID.
func (s *Service) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.Schedule, error) {
	return s.schedules.GetByID(ctx, id, userID)
}

// ListByRoom returns all schedules for the given room owned by userID.
// Verifies room ownership before listing.
func (s *Service) ListByRoom(ctx context.Context, roomID, userID uuid.UUID) ([]models.Schedule, error) {
	if _, err := s.rooms.GetByIDAndUserID(ctx, roomID, userID); err != nil {
		return nil, err
	}

	return s.schedules.ListByRoom(ctx, roomID, userID)
}

// Update updates the name of an existing schedule.
func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, name string) (*models.Schedule, error) {
	sched, err := s.schedules.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	sched.Name = name
	if err := s.schedules.Update(ctx, sched); err != nil {
		return nil, err
	}

	return sched, nil
}

// Delete removes a schedule and all its periods.
func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := s.schedules.GetByID(ctx, id, userID)
	if err != nil {
		return err
	}

	return s.schedules.Delete(ctx, id)
}

// Activate activates the given schedule, deactivating any currently active
// schedule for the room. Validates that the room has the required capabilities
// for all periods in the schedule before activating.
func (s *Service) Activate(ctx context.Context, id, userID uuid.UUID) (*models.Schedule, error) {
	sched, err := s.schedules.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if sched.IsActive {
		return nil, ErrAlreadyActive
	}

	capable, err := s.schedules.PeriodsHaveCapability(ctx, id, sched.RoomID)
	if err != nil {
		return nil, err
	}
	if !capable {
		return nil, ErrCapabilityConflict
	}

	if err := s.schedules.Activate(ctx, id, sched.RoomID); err != nil {
		return nil, err
	}

	sched.IsActive = true
	return sched, nil

}

// Deactivate deactivates the given schedule.
func (s *Service) Deactivate(ctx context.Context, id, userID uuid.UUID) (*models.Schedule, error) {
	sched, err := s.schedules.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if !sched.IsActive {
		return nil, ErrAlreadyInactive
	}

	if err := s.schedules.Deactivate(ctx, sched.ID, sched.RoomID); err != nil {
		return nil, err
	}

	sched.IsActive = false
	return sched, nil
}

// ---------------------------------------------------------------------------
// Schedule period methods
// ---------------------------------------------------------------------------

// PeriodInput holds the validated inputs for period create and update.
// Lives in the service layer — the handler binds its own request struct and
// maps onto this before calling the service.
type PeriodInput struct {
	Name       *string
	DaysOfWeek []int64
	StartTime  string // "HH:MM"
	EndTime    string // "HH:MM"
	Mode       string
	TargetTemp *float64
	TargetHum  *float64
}

// CreatePeriod creates a new period under the given schedule.
// Verifies schedule ownership and checks for time overlap before creating.
func (s *Service) CreatePeriod(ctx context.Context, scheduleID, userID uuid.UUID, input PeriodInput) (*models.SchedulePeriod, error) {
	sched, err := s.schedules.GetByID(ctx, scheduleID, userID)
	if err != nil {
		return nil, err
	}

	period, err := s.buildPeriod(ctx, sched.ID, input, nil)
	if err != nil {
		return nil, err
	}

	if err := s.schedules.CreatePeriod(ctx, &period); err != nil {
		return nil, err
	}

	return &period, nil
}

// GetPeriodByID returns the period with the given ID.
// Verifies ownership by checking the parent schedule belongs to userID.
func (s *Service) GetPeriodByID(ctx context.Context, periodID, userID uuid.UUID) (*models.SchedulePeriod, error) {
	period, err := s.schedules.GetPeriodByID(ctx, periodID)
	if err != nil {
		return nil, err
	}

	if _, err := s.schedules.GetByID(ctx, period.ScheduleID, userID); err != nil {
		return nil, ErrPeriodNotFound
	}

	return period, nil
}

// ListPeriodsBySchedule returns all periods for the given schedule.
// Verifies schedule ownership before listing.
func (s *Service) ListPeriodsBySchedule(ctx context.Context, scheduleID, userID uuid.UUID) ([]models.SchedulePeriod, error) {
	if _, err := s.schedules.GetByID(ctx, scheduleID, userID); err != nil {
		return nil, err
	}

	return s.schedules.ListPeriodsBySchedule(ctx, scheduleID)
}

// UpdatePeriod updates all mutable fields on an existing period.
// Verifies ownership and checks for overlap (excluding the period being updated).
func (s *Service) UpdatePeriod(ctx context.Context, periodID, userID uuid.UUID, input PeriodInput) (*models.SchedulePeriod, error) {
	period, err := s.GetPeriodByID(ctx, periodID, userID)
	if err != nil {
		return nil, err
	}

	built, err := s.buildPeriod(ctx, period.ScheduleID, input, &period.ID)
	if err != nil {
		return nil, err
	}

	built.ID = period.ID
	if err := s.schedules.UpdatePeriod(ctx, &built); err != nil {
		return nil, err
	}

	return &built, nil
}

// DeletePeriod removes a period by ID.
// Verifies ownership before deleting.
func (s *Service) DeletePeriod(ctx context.Context, periodID, userID uuid.UUID) error {
	if _, err := s.GetPeriodByID(ctx, periodID, userID); err != nil {
		return err
	}

	return s.schedules.DeletePeriod(ctx, periodID)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildPeriod validates the input, checks for overlap, and returns a single
// models.SchedulePeriod. Midnight-crossing periods are not supported — end time
// must be later than start time.
func (s *Service) buildPeriod(ctx context.Context, scheduleID uuid.UUID, input PeriodInput, excludeID *uuid.UUID) (models.SchedulePeriod, error) {
	if input.EndTime <= input.StartTime {
		return models.SchedulePeriod{}, ErrInvalidTimeRange
	}

	overlap, err := s.schedules.HasOverlap(ctx, scheduleID, input.DaysOfWeek, input.StartTime, input.EndTime, excludeID)
	if err != nil {
		return models.SchedulePeriod{}, err
	}
	if overlap {
		return models.SchedulePeriod{}, ErrPeriodOverlap
	}

	return models.SchedulePeriod{
		ScheduleID: scheduleID,
		Name:       input.Name,
		DaysOfWeek: pq.Int64Array(input.DaysOfWeek),
		StartTime:  input.StartTime,
		EndTime:    input.EndTime,
		Mode:       input.Mode,
		TargetTemp: input.TargetTemp,
		TargetHum:  input.TargetHum,
	}, nil
}
