package device

import "errors"

var (
	ErrNotFound           = errors.New("device not found")
	ErrNameTaken          = errors.New("device name already taken")
	ErrHwIDTaken          = errors.New("hardware ID already registered")
	ErrAlreadyOwned       = errors.New("this device is already registered to your account")
	ErrRoomNotFound       = errors.New("room not found")
	ErrInvalidSensor      = errors.New("invalid sensor type")
	ErrInvalidActuator    = errors.New("invalid actuator type")
	ErrDuplicateSensor    = errors.New("duplicate sensor type")
	ErrDuplicateActuator  = errors.New("duplicate actuator type")
	ErrCapabilityConflict = errors.New("removing this device would leave the room without a required capability")
)
