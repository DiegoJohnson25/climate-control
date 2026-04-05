package room

import (
	"context"
	"errors"

	"github.com/DiegoJohnson25/climate-control/shared/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// List returns all rooms belonging to the given user.
func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]models.Room, error) {
	var rms []models.Room
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rms).Error
	return rms, err
}

// GetByIDAndUserID fetches a single room, scoped to the owning user.
// Returns ErrNotFound if the room does not exist or belongs to a different user.
func (r *Repository) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*models.Room, error) {
	var rm models.Room
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&rm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rm, nil
}

// CreateWithDesiredState inserts a room and its initial desired_states row in a
// single transaction. The desired state defaults to mode=OFF with no targets.
func (r *Repository) CreateWithDesiredState(ctx context.Context, room *models.Room) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(room).Error; err != nil {
			if isUniqueViolation(err) {
				return ErrNameTaken
			}
			return err
		}

		ds := models.DesiredState{
			RoomID: room.ID,
			Mode:   "OFF",
		}

		if err := tx.Create(&ds).Error; err != nil {
			return err
		}

		// TODO: notify device-service of new room once device-service exists.
		// tx.Exec("SELECT pg_notify('room_config_changed', ?)", room.ID.String())

		return nil
	})
}

// Update saves name and deadband changes to an existing room.
func (r *Repository) Update(ctx context.Context, rm *models.Room) error {
	err := r.db.WithContext(ctx).Save(rm).Error
	if isUniqueViolation(err) {
		return ErrNameTaken
	}

	// TODO: notify device-service of config change.
	// r.db.WithContext(ctx).Exec("SELECT pg_notify('room_config_changed', ?)", room.ID.String())

	return err
}

// Delete removes a room by ID. Cascades handle desired_states, schedules, and
// devices (SET NULL on room_id — devices become unassigned, not deleted).
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Room{}, id).Error
}

// GetDesiredState returns the desired_states row for the given room.
func (r *Repository) GetDesiredState(ctx context.Context, roomID uuid.UUID) (models.DesiredState, error) {
	var ds models.DesiredState
	err := r.db.WithContext(ctx).Where("room_id = ?", roomID).First(&ds).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.DesiredState{}, ErrNotFound
	}
	return ds, err
}

// UpdateDesiredState persists the desired state for a room.
func (r *Repository) UpdateDesiredState(ctx context.Context, ds *models.DesiredState) error {
	// TODO: notify device-service of state change.
	// r.db.WithContext(ctx).Exec("SELECT pg_notify('desired_state_changed', ?)", ds.RoomID.String())

	return r.db.WithContext(ctx).Save(ds).Error
}

// -------------------------------------------------------------------------------
// Capability queries
// -------------------------------------------------------------------------------

// HasTemperatureCapability returns true if the room has at least one device with
// both a temperature sensor and a heater actuator.
func (r *Repository) HasTemperatureCapability(ctx context.Context, roomID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM devices d
		JOIN sensors s    ON s.device_id = d.id AND s.measurement_type = 'temperature'
		JOIN actuators a  ON a.device_id = d.id AND a.actuator_type    = 'heater'
		WHERE d.room_id = ?
	`, roomID).Scan(&count).Error
	return count > 0, err
}

// HasHumidityCapability returns true if the room has at least one device with
// both a humidity sensor and a humidifier actuator.
func (r *Repository) HasHumidityCapability(ctx context.Context, roomID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM devices d
		JOIN sensors s    ON s.device_id = d.id AND s.measurement_type = 'humidity'
		JOIN actuators a  ON a.device_id = d.id AND a.actuator_type    = 'humidifier'
		WHERE d.room_id = ?
	`, roomID).Scan(&count).Error
	return count > 0, err
}
