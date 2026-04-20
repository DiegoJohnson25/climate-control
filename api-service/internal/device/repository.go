package device

import (
	"context"
	"errors"

	"github.com/DiegoJohnson25/climate-control/shared/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeviceWithCapabilities struct {
	models.Device
	Sensors   []models.Sensor
	Actuators []models.Actuator
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// List returns all devices belonging to the given user, with sensors and
// actuators bulk fetched.
func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]DeviceWithCapabilities, error) {
	var devices []models.Device
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&devices).Error; err != nil {
		return nil, err
	}

	return r.attachCapabilities(ctx, devices)
}

// List returns all devices assigned to the given room, with sensors and
// actuators bulk fetched.
func (r *Repository) ListByRoom(ctx context.Context, roomID uuid.UUID) ([]DeviceWithCapabilities, error) {
	var devices []models.Device
	if err := r.db.WithContext(ctx).Where("room_id = ?", roomID).Find(&devices).Error; err != nil {
		return nil, err
	}

	return r.attachCapabilities(ctx, devices)
}

// GetByIDAndUserID fetches a single device scoped to the owning user, with its
// sensors and actuators attached.
// Returns ErrNotFound if the device does not exist or belongs to a different user.
func (r *Repository) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*DeviceWithCapabilities, error) {
	var dev models.Device
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&dev).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	devs, err := r.attachCapabilities(ctx, []models.Device{dev})
	if err != nil {
		return nil, err
	}
	return &devs[0], nil
}

// attachCapabilities bulk fetches sensors and actuators for a slice of devices
// using IN queries, then stitches them together. Always 2 extra queries
// regardless of device count.
func (r *Repository) attachCapabilities(ctx context.Context, devices []models.Device) ([]DeviceWithCapabilities, error) {
	if len(devices) == 0 {
		return []DeviceWithCapabilities{}, nil
	}

	ids := make([]uuid.UUID, len(devices))
	for i, d := range devices {
		ids[i] = d.ID
	}

	var sensors []models.Sensor
	if err := r.db.WithContext(ctx).Where("device_id IN ?", ids).Find(&sensors).Error; err != nil {
		return nil, err
	}

	var actuators []models.Actuator
	if err := r.db.WithContext(ctx).Where("device_id IN ?", ids).Find(&actuators).Error; err != nil {
		return nil, err
	}

	sensorMap := make(map[uuid.UUID][]models.Sensor)
	for _, s := range sensors {
		sensorMap[s.DeviceID] = append(sensorMap[s.DeviceID], s)
	}
	actuatorMap := make(map[uuid.UUID][]models.Actuator)
	for _, a := range actuators {
		actuatorMap[a.DeviceID] = append(actuatorMap[a.DeviceID], a)
	}

	result := make([]DeviceWithCapabilities, len(devices))
	for i, d := range devices {
		result[i] = DeviceWithCapabilities{
			Device:    d,
			Sensors:   sensorMap[d.ID],
			Actuators: actuatorMap[d.ID],
		}
		if result[i].Sensors == nil {
			result[i].Sensors = []models.Sensor{}
		}
		if result[i].Actuators == nil {
			result[i].Actuators = []models.Actuator{}
		}
	}

	return result, nil
}

// CheckHwIDAvailability verifies that hw_id is available for registration.
// Returns nil if available, ErrAlreadyOwned if the requesting user already owns
// a device with this hw_id, or ErrHwIDTaken if it belongs to another user.
//
// Note: in a production system with a manufacturer device registry, this would
// instead verify the hw_id exists in a pre-populated registry and is unclaimed,
// rather than checking for duplicate registrations.
//
// TODO: check admin blacklist here once the blacklisted_devices table exists.
func (r *Repository) CheckHwIDAvailability(ctx context.Context, hwID string, userID uuid.UUID) error {
	var dev models.Device
	err := r.db.WithContext(ctx).Where("hw_id = ?", hwID).First(&dev).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if dev.UserID == userID {
		return ErrAlreadyOwned
	}
	return ErrHwIDTaken
}

// Create inserts a device and its sensors and actuators in a single transaction.
// The service must call CheckHwIDAvailability before calling this.
func (r *Repository) Create(ctx context.Context, dev *models.Device, sensorTypes []string, actuatorTypes []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(dev).Error; err != nil {
			if isUniqueViolation(err) {
				// hw_id was pre-checked, so a unique violation here is the
				// (user_id, name) constraint.
				return ErrNameTaken
			}
			return err
		}

		for _, st := range sensorTypes {
			sensor := models.Sensor{
				DeviceID:        dev.ID,
				MeasurementType: st,
			}
			if err := tx.Create(&sensor).Error; err != nil {
				return err
			}
		}

		for _, at := range actuatorTypes {
			actuator := models.Actuator{
				DeviceID:     dev.ID,
				ActuatorType: at,
			}
			if err := tx.Create(&actuator).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// Update saves name and room_id changes to an existing device.
// The service is responsible for capability conflict checks before calling this.
func (r *Repository) Update(ctx context.Context, dev *models.Device) error {
	err := r.db.WithContext(ctx).Save(dev).Error
	if isUniqueViolation(err) {
		return ErrNameTaken
	}

	// TODO Phase 3e: events.NotifyDeviceChanged via Redis XADD.

	return err
}

// Delete removes a device by ID.
// The service is responsible for capability conflict checks before calling this.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Device{}, id).Error
}

// ---------------------------------------------------------------------------
// Capability conflict checks
// ---------------------------------------------------------------------------
//
// Called by the service before Update/Delete. These check whether removing a
// device would break the room's desired_state or active schedule periods.
// Only active schedules are checked — inactive schedules are ignored until
// activation time.

// HasCapabilityConflictAfterRemoval returns true if removing deviceID from roomID
// would leave desired_state or active schedule periods with targets the room
// can no longer satisfy.
func (r *Repository) HasCapabilityConflictAfterRemoval(ctx context.Context, roomID, deviceID uuid.UUID) (bool, error) {
	hasTempCap, err := r.hasTemperatureCapabilityAfterRemoval(ctx, roomID, deviceID)
	if err != nil {
		return false, err
	}

	hasHumCap, err := r.hasHumidityCapabilityAfterRemoval(ctx, roomID, deviceID)
	if err != nil {
		return false, err
	}

	if hasTempCap && hasHumCap {
		return false, nil
	}

	conflict, err := r.desiredStateHasConflict(ctx, roomID, hasTempCap, hasHumCap)
	if err != nil || conflict {
		return conflict, err
	}

	return r.activeSchedulePeriodsHaveConflict(ctx, roomID, hasTempCap, hasHumCap)
}

// hasTemperatureCapabilityAfterRemoval returns true if the room still has at least
// one device (other than deviceID) with a temperature sensor AND at least one
// device (other than deviceID) with a heater actuator.
// The sensor and actuator may be on different devices.
func (r *Repository) hasTemperatureCapabilityAfterRemoval(ctx context.Context, roomID, deviceID uuid.UUID) (bool, error) {
	var has bool
	err := r.db.WithContext(ctx).Raw(`
		SELECT (
			EXISTS (
				SELECT 1 FROM devices d
				JOIN sensors s ON s.device_id = d.id
				WHERE d.room_id = ? AND d.id != ? AND s.measurement_type = 'temperature'
			)
			AND
			EXISTS (
				SELECT 1 FROM devices d
				JOIN actuators a ON a.device_id = d.id
				WHERE d.room_id = ? AND d.id != ? AND a.actuator_type = 'heater'
			)
		)
	`, roomID, deviceID, roomID, deviceID).Scan(&has).Error
	return has, err
}

// hasHumidityCapabilityAfterRemoval returns true if the room still has at least
// one device (other than deviceID) with a humidity sensor AND at least one
// device (other than deviceID) with a humidifier actuator.
// The sensor and actuator may be on different devices.
func (r *Repository) hasHumidityCapabilityAfterRemoval(ctx context.Context, roomID, deviceID uuid.UUID) (bool, error) {
	var has bool
	err := r.db.WithContext(ctx).Raw(`
		SELECT (
			EXISTS (
				SELECT 1 FROM devices d
				JOIN sensors s ON s.device_id = d.id
				WHERE d.room_id = ? AND d.id != ? AND s.measurement_type = 'humidity'
			)
			AND
			EXISTS (
				SELECT 1 FROM devices d
				JOIN actuators a ON a.device_id = d.id
				WHERE d.room_id = ? AND d.id != ? AND a.actuator_type = 'humidifier'
			)
		)
	`, roomID, deviceID, roomID, deviceID).Scan(&has).Error
	return has, err
}

// desiredStateHasConflict returns true if the room's desired_state has a target
// that the room would no longer be able to satisfy.
func (r *Repository) desiredStateHasConflict(ctx context.Context, roomID uuid.UUID, hasTempCap, hasHumCap bool) (bool, error) {
	var ds models.DesiredState
	err := r.db.WithContext(ctx).Where("room_id = ?", roomID).First(&ds).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if !hasTempCap && ds.TargetTemp != nil {
		return true, nil
	}
	if !hasHumCap && ds.TargetHum != nil {
		return true, nil
	}

	return false, nil
}

// activeSchedulePeriodsHaveConflict returns true if any period in the room's
// active schedule has a target the room would no longer be able to satisfy.
// Inactive schedules are intentionally ignored.
func (r *Repository) activeSchedulePeriodsHaveConflict(ctx context.Context, roomID uuid.UUID, hasTempCap, hasHumCap bool) (bool, error) {
	var periods []models.SchedulePeriod
	err := r.db.WithContext(ctx).Raw(`
		SELECT sp.*
		FROM schedule_periods sp
		JOIN schedules s ON s.id = sp.schedule_id
		WHERE s.room_id   = ?
		AND   s.is_active = true
	`, roomID).Scan(&periods).Error
	if err != nil {
		return false, err
	}

	for _, p := range periods {
		if !hasTempCap && p.TargetTemp != nil {
			return true, nil
		}
		if !hasHumCap && p.TargetHum != nil {
			return true, nil
		}
	}

	return false, nil
}
