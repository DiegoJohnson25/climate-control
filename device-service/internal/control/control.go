// Package control implements the bang-bang climate control evaluation logic
// for device-service. It is intentionally free of I/O — callers supply the
// RoomCache snapshot and receive a TickResult containing commands to publish
// and a log entry to write.
package control

import (
	"encoding/json"
	"time"

	"github.com/DiegoJohnson25/climate-control/device-service/internal/cache"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/metricsdb"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// ActuatorCommand is a single command destined for one physical actuator.
type ActuatorCommand struct {
	HwID         string
	ActuatorType string
	State        bool
}

// MarshalPayload returns the JSON bytes to publish to devices/{hw_id}/cmd.
func (a ActuatorCommand) MarshalPayload() ([]byte, error) {
	return json.Marshal(struct {
		ActuatorType string `json:"actuator_type"`
		State        bool   `json:"state"`
	}{
		ActuatorType: a.ActuatorType,
		State:        a.State,
	})
}

// TickResult is returned by Evaluate and consumed by the scheduler.
// Commands is the full list of actuator commands to publish this tick —
// every actuator in the room receives a command every tick.
// LogEntry is the control log row to write after commands are sent.
// LastActivePeriod is non-nil when the control source is "schedule" — the
// scheduler must write this to rc.LastActivePeriod under rc.Mu.Lock so that
// grace period logic has a valid reference on the next tick.
type TickResult struct {
	Commands         []ActuatorCommand
	LogEntry         metricsdb.ControlLogEntry
	LastActivePeriod *cache.SchedulePeriodCache
}

// ---------------------------------------------------------------------------
// Effective state resolution
// ---------------------------------------------------------------------------

// effectiveState holds the resolved control intent for a single tick evaluation.
type effectiveState struct {
	mode    string
	targets map[string]*float64 // sensor_type → target value; nil when mode is OFF
	source  string
	period  *cache.SchedulePeriodCache // non-nil when source is "schedule" or "grace_period"
}

// resolveEffectiveState determines the control intent for the current tick by
// walking the priority chain: manual override → active schedule period →
// grace period → off. rc.Mu must be held for reading by the caller.
func resolveEffectiveState(rc *cache.RoomCache, now time.Time) effectiveState {
	// manual override: non-null and not expired
	if rc.DesiredState.ManualOverrideUntil != nil {
		if rc.DesiredState.ManualOverrideUntil.After(now) {
			return effectiveState{
				mode:    rc.DesiredState.Mode,
				targets: rc.DesiredState.Targets,
				source:  "manual_override",
			}
		}
	}

	loc := rc.Location
	if loc == nil {
		loc = time.UTC
	}
	localNow := now.In(loc)
	dayOfWeek := int(localNow.Weekday())
	if dayOfWeek == 0 {
		dayOfWeek = 7 // ISO 8601: Sunday = 7
	}
	nowMinutes := localNow.Hour()*60 + localNow.Minute()

	// active schedule period matching current day and time
	for i := range rc.ActivePeriods {
		p := &rc.ActivePeriods[i]
		if !p.DaysOfWeek[dayOfWeek] {
			continue
		}
		if nowMinutes < p.StartMinutes || nowMinutes >= p.EndMinutes {
			continue
		}
		return effectiveState{
			mode:    p.Mode,
			targets: p.Targets,
			source:  "schedule",
			period:  p,
		}
	}

	// grace period: within 1 minute after the last active period ended
	if rc.LastActivePeriod != nil {
		if rc.LastActivePeriod.DaysOfWeek[dayOfWeek] {
			minutesSinceEnd := nowMinutes - rc.LastActivePeriod.EndMinutes
			if minutesSinceEnd >= 0 && minutesSinceEnd < 1 {
				return effectiveState{
					mode:    rc.LastActivePeriod.Mode,
					targets: rc.LastActivePeriod.Targets,
					source:  "grace_period",
					period:  rc.LastActivePeriod,
				}
			}
		}
	}

	return effectiveState{
		mode:   "OFF",
		source: "none",
	}
}

// ---------------------------------------------------------------------------
// Evaluation
// ---------------------------------------------------------------------------

// Evaluate runs one control tick for the room. It acquires rc.Mu.RLock for
// the duration of evaluation and releases it before returning. Every actuator
// in the room receives a command every tick — commands are never suppressed,
// ensuring devices with a watchdog timer are continuously refreshed and do not
// fall back to their local failsafe state unintentionally.
//
// The caller is responsible for publishing the returned commands, writing the
// log entry, and updating rc.LastActivePeriod and rc.ActuatorStates.
//
// TODO: the unconditional per-tick command cadence doubles as a heartbeat —
// a device that stops receiving commands for more than one tick interval is
// unreachable by definition. Wire publish failures into a connectivity status
// tracker (e.g. Redis hash keyed by hw_id) to surface device online/offline
// state without a separate ping mechanism. Device watchdog TTL should be set
// to slightly more than one tick interval to tolerate a single missed publish.
func Evaluate(rc *cache.RoomCache, now time.Time, staleThreshold time.Duration) TickResult {
	rc.Mu.RLock()
	defer rc.Mu.RUnlock()

	es := resolveEffectiveState(rc, now)

	entry := metricsdb.ControlLogEntry{
		RoomID:        rc.RoomID,
		Time:          now,
		Mode:          es.mode,
		ControlSource: es.source,
	}
	if es.period != nil {
		entry.SchedulePeriodID = &es.period.ID
	}

	// initialise cmd fields only for actuator types present in this room —
	// absent types remain nil and write NULL to room_control_logs
	if _, ok := rc.ActuatorHwIDs["heater"]; ok {
		v := int16(0)
		entry.HeaterCmd = &v
	}
	if _, ok := rc.ActuatorHwIDs["humidifier"]; ok {
		v := int16(0)
		entry.HumidifierCmd = &v
	}

	// compute fresh averages unconditionally — always populated for the
	// historical graph regardless of effective mode or targets
	tempAvg, tempCount, hasTempReadings := freshAverage(rc.LatestReadings["temperature"], now, staleThreshold)
	humAvg, humCount, hasHumReadings := freshAverage(rc.LatestReadings["humidity"], now, staleThreshold)

	if hasTempReadings {
		entry.AvgTemp = &tempAvg
		n := int16(tempCount)
		entry.ReadingCountTemp = &n
	}
	if hasHumReadings {
		entry.AvgHum = &humAvg
		n := int16(humCount)
		entry.ReadingCountHum = &n
	}

	// populate target and deadband fields from effective state for log context;
	// deadbands are null when no target is set — they are only meaningful as a pair
	if es.targets != nil {
		entry.TargetTemp = es.targets["temperature"]
		entry.TargetHum = es.targets["humidity"]
		if entry.TargetTemp != nil {
			entry.DeadbandTemp = &rc.DeadbandTemp
		}
		if entry.TargetHum != nil {
			entry.DeadbandHum = &rc.DeadbandHum
		}
	}

	var commands []ActuatorCommand
	var lastActivePeriod *cache.SchedulePeriodCache
	if es.source == "schedule" || es.source == "grace_period" {
		lastActivePeriod = es.period
	}

	if es.mode == "OFF" {
		commandAllOff(rc, &entry, &commands)
		return TickResult{Commands: commands, LogEntry: entry, LastActivePeriod: lastActivePeriod}
	}

	// AUTO: for each actuator type in the room, evaluate against its target
	// and fresh readings. Send OFF if no target is set for that type or if no
	// fresh readings are available — do not leave actuators in an unknown state.
	evaluateOrOff(rc, "heater", "temperature", tempAvg, hasTempReadings, es.targets, rc.DeadbandTemp, &entry, &commands)
	evaluateOrOff(rc, "humidifier", "humidity", humAvg, hasHumReadings, es.targets, rc.DeadbandHum, &entry, &commands)

	return TickResult{Commands: commands, LogEntry: entry, LastActivePeriod: lastActivePeriod}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// commandAllOff appends an OFF command for every actuator in the room and
// sets cmd fields on the log entry to 0. Called when mode is OFF.
// Commands are sent unconditionally every tick to refresh device watchdog timers.
func commandAllOff(rc *cache.RoomCache, entry *metricsdb.ControlLogEntry, commands *[]ActuatorCommand) {
	for actuatorType, hwIDs := range rc.ActuatorHwIDs {
		setCmd(entry, actuatorType, 0)
		for _, hwID := range hwIDs {
			*commands = append(*commands, ActuatorCommand{
				HwID:         hwID,
				ActuatorType: actuatorType,
				State:        false,
			})
		}
	}
}

// evaluateOrOff applies bang-bang logic for one actuator type when a target
// and fresh readings are available. Falls back to OFF if either is absent —
// no target means the actuator should not be running; no fresh readings means
// the sensor is unavailable and the safe state is off.
//
// Commands are sent unconditionally every tick to refresh device watchdog timers.
func evaluateOrOff(
	rc *cache.RoomCache,
	actuatorType string,
	sensorType string,
	avg float64,
	hasReadings bool,
	targets map[string]*float64,
	deadband float64,
	entry *metricsdb.ControlLogEntry,
	commands *[]ActuatorCommand,
) {
	hwIDs, ok := rc.ActuatorHwIDs[actuatorType]
	if !ok {
		return
	}

	target := targets[sensorType]

	if target == nil || !hasReadings {
		// no target or no fresh readings — send OFF unconditionally
		setCmd(entry, actuatorType, 0)
		for _, hwID := range hwIDs {
			*commands = append(*commands, ActuatorCommand{
				HwID:         hwID,
				ActuatorType: actuatorType,
				State:        false,
			})
		}
		return
	}

	currentState := rc.ActuatorStates[actuatorType]
	var desiredState bool

	switch {
	case avg < *target-deadband:
		desiredState = true
	case avg > *target+deadband:
		desiredState = false
	default:
		desiredState = currentState // within deadband — re-send last commanded state
	}

	if desiredState {
		setCmd(entry, actuatorType, 1)
	} else {
		setCmd(entry, actuatorType, 0)
	}

	for _, hwID := range hwIDs {
		*commands = append(*commands, ActuatorCommand{
			HwID:         hwID,
			ActuatorType: actuatorType,
			State:        desiredState,
		})
	}
}

// freshAverage computes the mean of readings not older than threshold.
// Returns the average, the count of fresh readings, and whether any were found.
// The underlying slice is not modified — trimming is ingestion's responsibility.
func freshAverage(readings []cache.TimestampedReading, now time.Time, threshold time.Duration) (avg float64, count int, ok bool) {
	var sum float64
	for _, r := range readings {
		if now.Sub(r.Timestamp) <= threshold {
			sum += r.Value
			count++
		}
	}
	if count == 0 {
		return 0, 0, false
	}
	return sum / float64(count), count, true
}

// setCmd writes a cmd value into the correct field of the log entry by
// actuator type. No-ops if the field is nil (room has no actuator of that type).
func setCmd(entry *metricsdb.ControlLogEntry, actuatorType string, v int16) {
	switch actuatorType {
	case "heater":
		if entry.HeaterCmd != nil {
			*entry.HeaterCmd = v
		}
	case "humidifier":
		if entry.HumidifierCmd != nil {
			*entry.HumidifierCmd = v
		}
	}
}
