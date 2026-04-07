package schedule

import (
	"context"
	"errors"

	"github.com/DiegoJohnson25/climate-control/shared/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// -------------------------------------------------------------------------------
// Schedule methods
// -------------------------------------------------------------------------------

// Create inserts a new schedule row.
func (r *Repository) Create(ctx context.Context, sched *models.Schedule) error {
	err := r.db.WithContext(ctx).Create(sched).Error
	if isUniqueViolation(err) {
		return ErrNameTaken
	}
	return err
}

// GetByID returns the schedule with the given ID owned by userID.
// Returns ErrNotFound if the schedule does not exist or belongs to a different user.
func (r *Repository) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.Schedule, error) {
	var sched models.Schedule
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&sched).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sched, nil
}

// ListByRoom returns all schedules for the given room owned by userID.
func (r *Repository) ListByRoom(ctx context.Context, roomID, userID uuid.UUID) ([]models.Schedule, error) {
	var scheds []models.Schedule
	err := r.db.WithContext(ctx).Where("room_id = ? AND user_id = ?", roomID, userID).Find(&scheds).Error
	if err != nil {
		return nil, err
	}
	if scheds == nil {
		scheds = []models.Schedule{}
	}
	return scheds, nil
}

// Update saves mutable fields on an existing schedule (name only).
// The caller is responsible for fetching the schedule before mutating it.
func (r *Repository) Update(ctx context.Context, sched *models.Schedule) error {
	err := r.db.WithContext(ctx).Save(sched).Error
	if isUniqueViolation(err) {
		return ErrNameTaken
	}
	return err
}

// Delete removes a schedule by ID. Cascade deletes its periods.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Schedule{}, id).Error
}

// Activate atomically deactivates the currently active schedule for the room
// (if any) and activates the target schedule. Fires a pg_notify stub for
// device-service cache invalidation.
func (r *Repository) Activate(ctx context.Context, scheduleID, roomID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Deactivate the currently active schedule for this room, if any.
		err := tx.Model(&models.Schedule{}).
			Where("room_id = ? AND is_active = true AND id != ?", roomID, scheduleID).
			Update("is_active", false).Error
		if err != nil {
			return err
		}

		// Activate the target schedule.
		err = tx.Model(&models.Schedule{}).
			Where("id = ?", scheduleID).
			Update("is_active", true).Error
		if err != nil {
			if isUniqueViolation(err) {
				// Partial unique index violation — another schedule is already active
				// for this room. Should not happen given the deactivation above, but
				// guard against a race.
				return ErrAlreadyActive
			}
			return err
		}

		// TODO: notify device-service of schedule change.
		// tx.Exec("SELECT pg_notify('schedule_changed', ?)", roomID.String())

		return nil
	})
}

// Deactivate sets is_active = false on the given schedule.
// Fires a pg_notify stub for device-service cache invalidation.
func (r *Repository) Deactivate(ctx context.Context, scheduleID, roomID uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&models.Schedule{}).
		Where("id = ?", scheduleID).
		Update("is_active", false).Error
	if err != nil {
		return err
	}

	// TODO: notify device-service of schedule change.
	// r.db.WithContext(ctx).Exec("SELECT pg_notify('schedule_changed', ?)", roomID.String())

	return nil
}

// -------------------------------------------------------------------------------
// Schedule period methods
// -------------------------------------------------------------------------------

// CreatePeriod inserts a new schedule period row.
func (r *Repository) CreatePeriod(ctx context.Context, period *models.SchedulePeriod) error {
	return r.db.WithContext(ctx).Create(period).Error
}

// GetPeriodByID returns the period with the given ID.
// Ownership is verified at the service layer via the parent schedule.
func (r *Repository) GetPeriodByID(ctx context.Context, periodID uuid.UUID) (*models.SchedulePeriod, error) {
	var period models.SchedulePeriod
	err := r.db.WithContext(ctx).First(&period, periodID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPeriodNotFound
	}
	if err != nil {
		return nil, err
	}
	return &period, nil
}

// ListPeriodsBySchedule returns all periods for the given schedule.
func (r *Repository) ListPeriodsBySchedule(ctx context.Context, scheduleID uuid.UUID) ([]models.SchedulePeriod, error) {
	var periods []models.SchedulePeriod
	err := r.db.WithContext(ctx).Where("schedule_id = ?", scheduleID).Find(&periods).Error
	if err != nil {
		return nil, err
	}
	if periods == nil {
		periods = []models.SchedulePeriod{}
	}
	return periods, nil
}

// UpdatePeriod saves all mutable fields on an existing period.
// The caller is responsible for fetching the period before mutating it.
func (r *Repository) UpdatePeriod(ctx context.Context, period *models.SchedulePeriod) error {
	return r.db.WithContext(ctx).Save(period).Error
}

// DeletePeriod removes a period by ID.
func (r *Repository) DeletePeriod(ctx context.Context, periodID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.SchedulePeriod{}, periodID).Error
}

// -------------------------------------------------------------------------------
// Validation queries
// -------------------------------------------------------------------------------

// HasOverlap returns true if any existing period in the schedule shares at least
// one day with the candidate period and has an overlapping time range.
// excludeID is nil on create; on update it excludes the period being edited.
func (r *Repository) HasOverlap(
	ctx context.Context,
	scheduleID uuid.UUID,
	days []int64,
	startTime, endTime string,
	excludeID *uuid.UUID,
) (bool, error) {
	query := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM schedule_periods
		WHERE schedule_id = ?
		AND   days_of_week && ?
		AND   start_time < ?
		AND   end_time   > ?
		AND   (?::uuid IS NULL OR id != ?::uuid)
	`, scheduleID, pq.Int64Array(days), endTime, startTime, excludeID, excludeID)

	var count int64
	if err := query.Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// PeriodsHaveCapability returns true if every AUTO period in the schedule that
// has a target_temp or target_humidity can be satisfied by the room's current
// devices. Called at activation time.
func (r *Repository) PeriodsHaveCapability(ctx context.Context, scheduleID, roomID uuid.UUID) (bool, error) {
	// Check if any period requires temperature control but the room lacks it.
	var tempConflict int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM schedule_periods sp
		WHERE sp.schedule_id  = ?
		AND   sp.target_temp IS NOT NULL
		AND   NOT EXISTS (
			SELECT 1
			FROM devices d
			JOIN sensors s   ON s.device_id = d.id AND s.measurement_type = 'temperature'
			JOIN actuators a ON a.device_id = d.id AND a.actuator_type    = 'heater'
			WHERE d.room_id = ?
		)
	`, scheduleID, roomID).Scan(&tempConflict).Error
	if err != nil {
		return false, err
	}
	if tempConflict > 0 {
		return false, nil
	}

	// Check if any period requires humidity control but the room lacks it.
	var humConflict int64
	err = r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM schedule_periods sp
		WHERE sp.schedule_id     = ?
		AND   sp.target_hum IS NOT NULL
		AND   NOT EXISTS (
			SELECT 1
			FROM devices d
			JOIN sensors s   ON s.device_id = d.id AND s.measurement_type = 'humidity'
			JOIN actuators a ON a.device_id = d.id AND a.actuator_type    = 'humidifier'
			WHERE d.room_id = ?
		)
	`, scheduleID, roomID).Scan(&humConflict).Error
	if err != nil {
		return false, err
	}

	return humConflict == 0, nil
}
