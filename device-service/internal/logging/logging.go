// Package logging provides structured debug helpers for device-service.
// Covers cache inspection, control tick results, and stream event tracing.
package logging

import (
	"fmt"
	"log"
	"strings"

	"github.com/DiegoJohnson25/climate-control/device-service/internal/cache"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/control"
	"github.com/google/uuid"
)

// LogStore logs a summary of the entire cache store — room count and a
// summary line per room. Use LogFullStore for full field detail.
func LogStore(store *cache.Store) {
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

// LogFullStore logs the complete cache state for every room at full granularity.
func LogFullStore(store *cache.Store) {
	roomIDs := store.RoomIDs()
	log.Printf("cache: store: %d rooms", len(roomIDs))
	for _, id := range roomIDs {
		rc := store.Room(id)
		if rc == nil {
			continue
		}
		LogFullRoom(rc)
	}
}

// LogDevices logs all cached devices including their sensors and actuators.
func LogDevices(store *cache.Store) {
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

// LogRoom logs a summary of a single room.
func LogRoom(rc *cache.RoomCache) {
	rc.Mu.RLock()
	defer rc.Mu.RUnlock()

	log.Printf("cache: room %s:", rc.RoomID)
	log.Printf("  timezone:       %s", rc.UserTimezone)
	log.Printf("  deadband_temp:  %.1f", rc.DeadbandTemp)
	log.Printf("  deadband_hum:   %.1f", rc.DeadbandHum)
	log.Printf("  mode:           %s", rc.DesiredState.Mode)
	log.Printf("  active_periods: %d", len(rc.ActivePeriods))
	log.Printf("  actuator_types: %d", len(rc.ActuatorHwIDs))
}

// LogFullRoom logs all fields of a room cache at full granularity.
func LogFullRoom(rc *cache.RoomCache) {
	rc.Mu.RLock()
	defer rc.Mu.RUnlock()

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
}

// LogDevice logs a single device cache entry including sensors and actuators.
func LogDevice(dc *cache.DeviceCache) {
	roomID := dc.GetRoomID()
	roomStr := "unassigned"
	if roomID != nil {
		roomStr = roomID.String()
	}

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
}

// The helpers below are called from LogFullRoom while the room lock is
// already held — they must not attempt to acquire the lock again.

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
	for actuatorType, hwIDs := range m {
		log.Printf("    %s: [%s]", actuatorType, strings.Join(hwIDs, ", "))
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

// activeDays converts a [8]bool DaysOfWeek bitmask to a readable slice of
// day name abbreviations.
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

// LogTickResult logs the commands and control log entry produced by a single
// control tick evaluation. Used for debugging command publishing — call from
// scheduler.tick when investigating missing MQTT commands.
func LogTickResult(roomID uuid.UUID, result control.TickResult) {
	entry := result.LogEntry
	log.Printf("tick: room %s: source=%s mode=%s commands=%d",
		roomID, entry.ControlSource, entry.Mode, len(result.Commands))

	if len(result.Commands) == 0 {
		log.Printf("tick:   no commands generated")
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

// LogStreamEvent logs a successfully processed cache invalidation event.
// Call after ACK so only events that were fully handled appear in the log.
// values is the raw XMessage.Values map — room_id and hw_id are extracted
// if present so device events can be distinguished from room-only events.
func LogStreamEvent(event, msgID string, values map[string]any) {
	roomID, _ := values["room_id"].(string)
	hwID, _ := values["hw_id"].(string)
	if hwID != "" {
		log.Printf("stream: event=%s msg=%s room=%s hw_id=%s", event, msgID, roomID, hwID)
	} else {
		log.Printf("stream: event=%s msg=%s room=%s", event, msgID, roomID)
	}
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
