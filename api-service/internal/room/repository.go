package room

import (
	"context"
	"errors"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/models"
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

		return nil
	})
}

// Update saves name and deadband changes to an existing room.
func (r *Repository) Update(ctx context.Context, rm *models.Room) error {
	err := r.db.WithContext(ctx).Save(rm).Error
	if isUniqueViolation(err) {
		return ErrNameTaken
	}
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
	return r.db.WithContext(ctx).Save(ds).Error
}

// ---------------------------------------------------------------------------
// Capability queries
// ---------------------------------------------------------------------------

// RoomCapabilities describes which climate control capabilities are available in a room.
// Temperature is true when the room has both a temperature sensor and a heater.
// Humidity is true when the room has both a humidity sensor and a humidifier.
type RoomCapabilities struct {
	Temperature bool
	Humidity    bool
}

// RoomWithCapabilities pairs a room with its resolved capability flags.
type RoomWithCapabilities struct {
	models.Room
	Capabilities RoomCapabilities
}

// RoomCapabilities returns capability flags for a single room in one query using
// four EXISTS checks — all four conditions evaluated in a single round-trip.
func (r *Repository) RoomCapabilities(ctx context.Context, roomID uuid.UUID) (RoomCapabilities, error) {
	var result struct {
		Temperature bool
		Humidity    bool
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			(
				EXISTS (
					SELECT 1 FROM devices d
					JOIN sensors s ON s.device_id = d.id
					WHERE d.room_id = ? AND s.measurement_type = 'temperature'
				)
				AND
				EXISTS (
					SELECT 1 FROM devices d
					JOIN actuators a ON a.device_id = d.id
					WHERE d.room_id = ? AND a.actuator_type = 'heater'
				)
			) AS temperature,
			(
				EXISTS (
					SELECT 1 FROM devices d
					JOIN sensors s ON s.device_id = d.id
					WHERE d.room_id = ? AND s.measurement_type = 'humidity'
				)
				AND
				EXISTS (
					SELECT 1 FROM devices d
					JOIN actuators a ON a.device_id = d.id
					WHERE d.room_id = ? AND a.actuator_type = 'humidifier'
				)
			) AS humidity
	`, roomID, roomID, roomID, roomID).Scan(&result).Error
	return RoomCapabilities{Temperature: result.Temperature, Humidity: result.Humidity}, err
}

// BulkRoomCapabilities returns capability flags for a set of rooms in a single query.
// Returns an empty map for an empty input slice without hitting the database.
func (r *Repository) BulkRoomCapabilities(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]RoomCapabilities, error) {
	out := make(map[uuid.UUID]RoomCapabilities, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	var rows []struct {
		ID          uuid.UUID
		Temperature bool
		Humidity    bool
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			r.id,
			(
				EXISTS (
					SELECT 1 FROM devices d
					JOIN sensors s ON s.device_id = d.id
					WHERE d.room_id = r.id AND s.measurement_type = 'temperature'
				)
				AND
				EXISTS (
					SELECT 1 FROM devices d
					JOIN actuators a ON a.device_id = d.id
					WHERE d.room_id = r.id AND a.actuator_type = 'heater'
				)
			) AS temperature,
			(
				EXISTS (
					SELECT 1 FROM devices d
					JOIN sensors s ON s.device_id = d.id
					WHERE d.room_id = r.id AND s.measurement_type = 'humidity'
				)
				AND
				EXISTS (
					SELECT 1 FROM devices d
					JOIN actuators a ON a.device_id = d.id
					WHERE d.room_id = r.id AND a.actuator_type = 'humidifier'
				)
			) AS humidity
		FROM rooms r
		WHERE r.id IN ?
	`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		out[row.ID] = RoomCapabilities{Temperature: row.Temperature, Humidity: row.Humidity}
	}
	return out, nil
}

// HasTemperatureCapability returns true if the room has both a temperature sensor
// and a heater. Delegates to RoomCapabilities to avoid SQL duplication.
func (r *Repository) HasTemperatureCapability(ctx context.Context, roomID uuid.UUID) (bool, error) {
	caps, err := r.RoomCapabilities(ctx, roomID)
	return caps.Temperature, err
}

// HasHumidityCapability returns true if the room has both a humidity sensor and a
// humidifier. Delegates to RoomCapabilities to avoid SQL duplication.
func (r *Repository) HasHumidityCapability(ctx context.Context, roomID uuid.UUID) (bool, error) {
	caps, err := r.RoomCapabilities(ctx, roomID)
	return caps.Humidity, err
}
