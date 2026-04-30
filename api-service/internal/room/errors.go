package room

import "errors"

var (
	ErrNotFound        = errors.New("room not found")
	ErrNameTaken       = errors.New("room name already taken")
	ErrNoCapability    = errors.New("room lacks required sensors or actuators for requested mode")
	ErrInvalidState    = errors.New("AUTO mode requires at least one target (temp or humidity)")
	ErrInvalidOverride = errors.New("manual_override must be a valid RFC3339 timestamp, \"indefinite\", or null")
	ErrInvalidTarget   = errors.New("target_temp must be between 5 and 40, target_hum must be between 0 and 100")
)
