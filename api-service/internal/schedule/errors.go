package schedule

import "errors"

var (
	ErrNotFound           = errors.New("schedule not found")
	ErrPeriodNotFound     = errors.New("schedule period not found")
	ErrNameTaken          = errors.New("schedule name already exists for this room")
	ErrPeriodOverlap      = errors.New("period overlaps with an existing period")
	ErrInvalidTimeRange   = errors.New("end time must be later than start time")
	ErrCapabilityConflict = errors.New("room lacks required capability for one or more periods")
	ErrAlreadyActive      = errors.New("schedule is already active")
	ErrAlreadyInactive    = errors.New("schedule is already inactive")
)
