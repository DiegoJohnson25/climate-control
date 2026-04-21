// Package debug provides structured debug helpers for device-service.
// All functions are no-ops unless debug mode is enabled via SetLevel.
// Two levels: Info for key events (reloads, stream events, ingestion flow),
// Verbose for full cache detail and per-tick control breakdown.
package debug

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/DiegoJohnson25/climate-control/device-service/internal/cache"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/control"
	"github.com/google/uuid"
)

// Level controls how much debug output is produced.
type Level int

const (
	LevelOff     Level = 0
	LevelInfo    Level = 1 // key events: reloads, stream events, ingestion flow
	LevelVerbose Level = 2 // full cache detail, per-tick control breakdown
)

var level Level

// traceIngestion and traceTick gate LogIngestion and LogTick independently.
// Both are checked in addition to the level — tracing a category only produces
// output when the level is also at Info or above. Set via SetTraceIngestion /
// SetTraceTick from config at startup.
var traceIngestion bool
var traceTick bool

// SetTraceIngestion enables or disables ingestion trace logging.
func SetTraceIngestion(v bool) { traceIngestion = v }

// SetTraceTick enables or disables tick trace logging.
func SetTraceTick(v bool) { traceTick = v }

// String returns the string representation of a Level for logging.
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "info"
	case LevelVerbose:
		return "verbose"
	default:
		return "off"
	}
}

// SetLevel enables debug output at the given verbosity. Must be called after
// SetTraceIngestion and SetTraceTick so the startup log reflects all active
// flags. No-op log when level is off.
func SetLevel(l Level) {
	level = l
	if l > LevelOff {
		log.Printf("debug: level=%s trace_ingestion=%v trace_tick=%v", l, traceIngestion, traceTick)
	}
}

// ParseLevel converts a DEVICE_DEBUG env string to a Level.
// "info" → LevelInfo, "verbose" → LevelVerbose, anything else → LevelOff.
func ParseLevel(s string) Level {
	switch s {
	case "info":
		return LevelInfo
	case "verbose":
		return LevelVerbose
	default:
		return LevelOff
	}
}

func atLevel(l Level) bool { return level >= l }

// AtInfo reports whether Info-level debug output is active. Use to guard
// work that is only needed to produce a debug log — avoids allocations on
// hot paths when debug is off.
func AtInfo() bool { return atLevel(LevelInfo) }

// AtIngestion reports whether ingestion trace logging is active. Use to guard
// the types slice allocation in ingestion.Process.
func AtIngestion() bool { return atLevel(LevelInfo) && traceIngestion }

// ---------------------------------------------------------------------------
// Cache inspection
// ---------------------------------------------------------------------------

// LogStore logs the full cache store. At Info, one compact line per room.
// At Verbose, full field detail per room. Dispatch is automatic — LogStore
// calls LogRoom which reads the current level.
func LogStore(store *cache.Store) {
	if !atLevel(LevelInfo) {
		return
	}
	roomIDs := store.RoomIDs()
	log.Printf("cache: store: %d rooms", len(roomIDs))
	for _, id := range roomIDs {
		rc := store.Room(id)
		if rc == nil {
			continue
		}
		LogRoom(rc)
	}
}

// LogDevices logs all cached devices. At Info, one compact line per device.
// At Verbose, full sensor and actuator detail.
func LogDevices(store *cache.Store) {
	if !atLevel(LevelInfo) {
		return
	}
	hwIDs := store.DeviceHwIDs()
	log.Printf("cache: devices: %d", len(hwIDs))
	for _, hwID := range hwIDs {
		dc := store.Device(hwID)
		if dc == nil {
			continue
		}
		LogDevice(dc)
	}
}

// LogRoom logs a single room. At Info, a compact summary line. At Verbose,
// full field detail including desired state, periods, actuator states, and
// latest readings. Used as the single call site for both reload confirmations
// (appdb) and full store dumps (LogStore).
func LogRoom(rc *cache.RoomCache) {
	if !atLevel(LevelInfo) {
		return
	}
	rc.Mu.RLock()
	defer rc.Mu.RUnlock()

	if atLevel(LevelVerbose) {
		log.Printf("cache: room %s:", rc.RoomID)
		log.Printf("  timezone:      %s", rc.UserTimezone)
		log.Printf("  deadband_temp: %.1f", rc.DeadbandTemp)
		log.Printf("  deadband_hum:  %.1f", rc.DeadbandHum)
		logDesiredState(rc.DesiredState)
		logPeriods(rc.ActivePeriods)
		logActuatorHwIDs(rc.ActuatorHwIDs)
		logActuatorStates(rc.ActuatorStates)
		logLatestReadings(rc.LatestReadings)
		if rc.LastActivePeriod != nil {
			log.Printf("  last_active_period: %s", rc.LastActivePeriod.ID)
		} else {
			log.Printf("  last_active_period: none")
		}
		return
	}

	log.Printf("cache: room %s: mode=%s periods=%d actuators=[%s]",
		rc.RoomID, rc.DesiredState.Mode, len(rc.ActivePeriods),
		strings.Join(sortedKeys(rc.ActuatorHwIDs), ", "))
}

// LogDevice logs a single device. At Info, a compact summary line. At Verbose,
// full detail including sensor and actuator UUIDs.
func LogDevice(dc *cache.DeviceCache) {
	if !atLevel(LevelInfo) {
		return
	}
	roomID := dc.GetRoomID()
	roomStr := "unassigned"
	if roomID != nil {
		roomStr = roomID.String()
	}

	if atLevel(LevelVerbose) {
		log.Printf("cache: device %s:", dc.HwID)
		log.Printf("  device_id: %s", dc.DeviceID)
		log.Printf("  room_id:   %s", roomStr)
		log.Printf("  sensors:   %d", len(dc.Sensors))
		for _, s := range dc.Sensors {
			log.Printf("    %s: %s", s.MeasurementType, s.SensorID)
		}
		log.Printf("  actuators: %d", len(dc.Actuators))
		for _, a := range dc.Actuators {
			log.Printf("    %s: %s", a.ActuatorType, a.ActuatorID)
		}
		return
	}

	log.Printf("cache: device %s: room=%s sensors=[%s] actuators=[%s]",
		dc.HwID, roomStr,
		strings.Join(sensorTypeList(dc.Sensors), ", "),
		strings.Join(actuatorTypeList(dc.Actuators), ", "))
}

// ---------------------------------------------------------------------------
// Stream event tracing
// ---------------------------------------------------------------------------

// LogStreamEvent logs a successfully processed cache invalidation event.
// Info level only — called after ACK so only fully handled events appear.
func LogStreamEvent(event, msgID string, values map[string]any) {
	if !atLevel(LevelInfo) {
		return
	}
	roomID, _ := values["room_id"].(string)
	hwID, _ := values["hw_id"].(string)
	if hwID != "" {
		log.Printf("stream: event=%s msg=%s room=%s hw_id=%s", event, msgID, roomID, hwID)
	} else {
		log.Printf("stream: event=%s msg=%s room=%s", event, msgID, roomID)
	}
}

// ---------------------------------------------------------------------------
// Control and ingestion
// ---------------------------------------------------------------------------

// LogTick logs a control tick evaluation. Requires both Info level and
// DEVICE_TRACE_TICK=true. At Info, a single summary line. At Verbose, full
// readings, targets, and command detail.
func LogTick(roomID uuid.UUID, result control.TickResult) {
	if !atLevel(LevelInfo) || !traceTick {
		return
	}
	entry := result.LogEntry
	log.Printf("tick: room %s: source=%s mode=%s commands=%d",
		roomID, entry.ControlSource, entry.Mode, len(result.Commands))

	if !atLevel(LevelVerbose) {
		return
	}

	if len(result.Commands) == 0 {
		log.Printf("tick:   no commands")
	}
	for _, cmd := range result.Commands {
		log.Printf("tick:   cmd: hw_id=%s actuator=%s state=%v", cmd.HwID, cmd.ActuatorType, cmd.State)
	}

	if entry.AvgTemp != nil {
		log.Printf("tick:   avg_temp=%.2f (n=%s) target=%s",
			*entry.AvgTemp,
			formatInt16Ptr(entry.ReadingCountTemp),
			formatFloat64Ptr(entry.TargetTemp),
		)
	} else {
		log.Printf("tick:   avg_temp=none target=%s", formatFloat64Ptr(entry.TargetTemp))
	}

	if entry.AvgHum != nil {
		log.Printf("tick:   avg_hum=%.2f (n=%s) target=%s",
			*entry.AvgHum,
			formatInt16Ptr(entry.ReadingCountHum),
			formatFloat64Ptr(entry.TargetHum),
		)
	} else {
		log.Printf("tick:   avg_hum=none target=%s", formatFloat64Ptr(entry.TargetHum))
	}

	if entry.HeaterCmd != nil {
		log.Printf("tick:   heater_cmd=%d", *entry.HeaterCmd)
	}
	if entry.HumidifierCmd != nil {
		log.Printf("tick:   humidifier_cmd=%d", *entry.HumidifierCmd)
	}
}

// LogIngestion logs a processed telemetry message. Requires both Info level
// and DEVICE_TRACE_INGESTION=true.
func LogIngestion(hwID string, roomID uuid.UUID, types []string) {
	if !atLevel(LevelInfo) || !traceIngestion {
		return
	}
	log.Printf("ingestion: hw_id=%s room=%s types=[%s]", hwID, roomID, strings.Join(types, ", "))
}

// ---------------------------------------------------------------------------
// Unexported helpers — called from LogRoom at Verbose while lock is already held
// ---------------------------------------------------------------------------

func logDesiredState(ds cache.DesiredStateCache) {
	log.Printf("  desired_state:")
	log.Printf("    mode: %s", ds.Mode)
	for sensorType, target := range ds.Targets {
		if target != nil {
			log.Printf("    target_%s: %.2f", sensorType, *target)
		} else {
			log.Printf("    target_%s: nil", sensorType)
		}
	}
	if ds.ManualOverrideUntil != nil {
		if ds.ManualOverrideUntil.Year() == 9999 {
			log.Printf("    manual_override_until: indefinite")
		} else {
			log.Printf("    manual_override_until: %s", ds.ManualOverrideUntil.Format("2006-01-02T15:04:05Z"))
		}
	} else {
		log.Printf("    manual_override_until: none")
	}
}

func logPeriods(periods []cache.SchedulePeriodCache) {
	log.Printf("  active_periods: %d", len(periods))
	for i, p := range periods {
		days := activeDays(p.DaysOfWeek)
		log.Printf("  period[%d] %s:", i, p.ID)
		log.Printf("    days:  [%s]", strings.Join(days, ", "))
		log.Printf("    start: %02d:%02d", p.StartMinutes/60, p.StartMinutes%60)
		log.Printf("    end:   %02d:%02d", p.EndMinutes/60, p.EndMinutes%60)
		log.Printf("    mode:  %s", p.Mode)
		for sensorType, target := range p.Targets {
			if target != nil {
				log.Printf("    target_%s: %.2f", sensorType, *target)
			}
		}
	}
}

func logActuatorHwIDs(m map[string][]string) {
	log.Printf("  actuator_hw_ids:")
	for _, actuatorType := range sortedKeys(m) {
		log.Printf("    %s: [%s]", actuatorType, strings.Join(m[actuatorType], ", "))
	}
}

func logActuatorStates(states map[string]bool) {
	log.Printf("  actuator_states:")
	if len(states) == 0 {
		log.Printf("    none")
		return
	}
	for actuatorType, state := range states {
		log.Printf("    %s: %v", actuatorType, state)
	}
}

func logLatestReadings(readings map[string][]cache.TimestampedReading) {
	log.Printf("  latest_readings:")
	if len(readings) == 0 {
		log.Printf("    none")
		return
	}
	for sensorType, rs := range readings {
		if len(rs) == 0 {
			log.Printf("    %s: no readings", sensorType)
			continue
		}
		vals := make([]string, 0, len(rs))
		for _, r := range rs {
			vals = append(vals, fmt.Sprintf("%.2f@%s", r.Value, r.Timestamp.Format("15:04:05")))
		}
		log.Printf("    %s: [%s]", sensorType, strings.Join(vals, ", "))
	}
}

func activeDays(dow [8]bool) []string {
	names := [8]string{"", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	var result []string
	for i := 1; i <= 7; i++ {
		if dow[i] {
			result = append(result, names[i])
		}
	}
	return result
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sensorTypeList(m map[string]cache.SensorEntry) []string {
	types := make([]string, 0, len(m))
	for t := range m {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func actuatorTypeList(m map[string]cache.ActuatorEntry) []string {
	types := make([]string, 0, len(m))
	for t := range m {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func formatFloat64Ptr(v *float64) string {
	if v == nil {
		return "none"
	}
	return fmt.Sprintf("%.2f", *v)
}

func formatInt16Ptr(v *int16) string {
	if v == nil {
		return "none"
	}
	return fmt.Sprintf("%d", *v)
}
