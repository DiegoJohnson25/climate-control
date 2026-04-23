// Package config loads the simulator's runtime configuration — shared room and
// device templates, a simulation-specific YAML file selected via the
// --simulation flag, and environment variables. Template overrides declared in
// the simulation file win on id collision.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/goccy/go-yaml"
)

// ---------------------------------------------------------------------------
// Physical base constants
// ---------------------------------------------------------------------------

// Base physical values for each measurement type. All template and simulation
// scale fields multiply against these. Defined once here — never in YAML.
const (
	baseThermalMassTemperature float64 = 10_000_000 // J/°C
	baseThermalMassHumidity    float64 = 325        // abstract moisture capacity units

	baseConductanceTemperature float64 = 100   // W/°C
	baseConductanceHumidity    float64 = 0.001 // %RH/s per %RH difference

	baseRateTemperature float64 = 1000  // W
	baseRateHumidity    float64 = 0.009 // %RH/s
)

// baseThermalMass returns the base thermal mass constant for a measurement type.
func baseThermalMass(typ string) float64 {
	switch typ {
	case "temperature":
		return baseThermalMassTemperature
	case "humidity":
		return baseThermalMassHumidity
	default:
		return 1
	}
}

// baseConductance returns the base conductance constant for a measurement type.
func baseConductance(typ string) float64 {
	switch typ {
	case "temperature":
		return baseConductanceTemperature
	case "humidity":
		return baseConductanceHumidity
	default:
		return 1
	}
}

// baseRate returns the base actuator rate constant for a measurement type.
func baseRate(typ string) float64 {
	switch typ {
	case "temperature":
		return baseRateTemperature
	case "humidity":
		return baseRateHumidity
	default:
		return 1
	}
}

// ---------------------------------------------------------------------------
// Timing and bounds constants
// ---------------------------------------------------------------------------

const (
	defaultMinPublishIntervalMS = 500
	defaultTimeScale            = 1.0
	defaultRoomType             = "static"
	maxTimeScale                = 400
	watchdogMultiplier          = 3
)

// MeasurementBounds defines the valid clamp range for each measurement type.
// Applied by the simulator after each model advance.
var MeasurementBounds = map[string][2]float64{
	"temperature": {5, 40},
	"humidity":    {0, 100},
}

// MeasurementToActuatorName maps internal measurement types to the actuator
// names the api-service expects when registering devices.
var MeasurementToActuatorName = map[string]string{
	"temperature": "heater",
	"humidity":    "humidifier",
}

// ActuatorNameToMeasurement maps api-service actuator type names to the
// internal measurement types used by the simulator.
var ActuatorNameToMeasurement = map[string]string{
	"heater":     "temperature",
	"humidifier": "humidity",
}

// ---------------------------------------------------------------------------
// Output types
// ---------------------------------------------------------------------------

// Config holds the fully resolved runtime configuration for the simulator.
type Config struct {
	APIURL                   string
	MQTTHost                 string
	MQTTPort                 int
	MQTTClientID             string
	MQTTUsername             string
	MQTTPassword             string
	EmailTemplate            string
	Password                 string
	BaseTickSeconds          int
	EffectivePublishInterval time.Duration
	SimulatedTickSeconds     float64
	Simulation               Simulation
}

// WatchdogTimeoutSeconds returns the real wall-clock duration after which a
// silent device-service is treated as absent, expressed in seconds.
func (c *Config) WatchdogTimeoutSeconds() int {
	return c.BaseTickSeconds * watchdogMultiplier
}

// Simulation holds the resolved topology for a single simulation run.
type Simulation struct {
	Name                 string
	TimeScale            float64
	MinPublishIntervalMS int
	UserGroups           []UserGroup
}

// UserGroup describes a set of identical simulated users and their rooms.
type UserGroup struct {
	Count       int
	Interactive bool
	Rooms       []Room
}

// Room holds the fully resolved configuration for a simulated room, including
// its environment model parameters, devices, and optional control configuration.
type Room struct {
	NamePrefix   string
	Count        int
	Type         string // "static" | "reactive" | "physics"
	Noisy        bool
	Measurements map[string]MeasurementConfig
	Devices      []Device
	DesiredState *ResolvedDesiredState // nil if not configured for this room
	Schedule     *ResolvedSchedule     // nil if not configured for this room
}

// MeasurementConfig holds the fully resolved environment model parameters for
// a single measurement type. All values are in physical units — scales have
// been applied at load time.
type MeasurementConfig struct {
	Base        float64 // ambient equilibrium value
	ThermalMass float64 // resistance to change
	Conductance float64 // rate of return toward ambient
	Noise       float64 // std dev of room-level environmental variation; zero if not noisy
}

// Device holds the fully resolved configuration for a simulated device.
type Device struct {
	NamePrefix string
	Count      int
	Sensors    []SensorConfig
	Actuators  []ActuatorConfig
}

// SensorConfig describes a single sensor on a device.
type SensorConfig struct {
	Type   string
	Noise  float64 // std dev of sensor measurement noise
	Offset float64
}

// ActuatorConfig describes a single actuator on a device.
type ActuatorConfig struct {
	Type string
	Rate float64 // fully resolved power output in physical units
}

// ResolvedDesiredState holds the target values to provision via
// PUT /rooms/:id/desired-state. Mode is always AUTO and manual_override_until
// is always "indefinite" — neither is configurable.
type ResolvedDesiredState struct {
	TargetTemp *float64
	TargetHum  *float64
}

// ResolvedSchedule holds the schedule name and periods to provision via the
// schedule and period create endpoints.
type ResolvedSchedule struct {
	Name    string
	Periods []ResolvedPeriod
}

// ResolvedPeriod holds the fully resolved parameters for a single schedule
// period. DaysOfWeek uses 1=Monday, 7=Sunday, matching the api-service contract.
type ResolvedPeriod struct {
	DaysOfWeek []int
	StartTime  string
	EndTime    string
	Mode       string
	TargetTemp *float64
	TargetHum  *float64
}

// ---------------------------------------------------------------------------
// Raw YAML types
// ---------------------------------------------------------------------------

type rawRoomTemplates struct {
	RoomTemplates []rawRoomTemplate `yaml:"room_templates"`
}

type rawRoomTemplate struct {
	ID           string                          `yaml:"id"`
	Measurements map[string]rawMeasurementConfig `yaml:"measurements"`
}

type rawMeasurementConfig struct {
	Base             float64 `yaml:"base"`
	Noise            float64 `yaml:"noise"`
	ThermalMassScale float64 `yaml:"thermal_mass_scale"`
	ConductanceScale float64 `yaml:"conductance_scale"`
}

type rawDeviceTemplates struct {
	DeviceTemplates []rawDeviceTemplate `yaml:"device_templates"`
}

type rawDeviceTemplate struct {
	ID        string              `yaml:"id"`
	Sensors   []rawSensorConfig   `yaml:"sensors"`
	Actuators []rawActuatorConfig `yaml:"actuators"`
}

type rawSensorConfig struct {
	Type   string  `yaml:"type"`
	Noise  float64 `yaml:"noise"`
	Offset float64 `yaml:"offset"`
}

type rawActuatorConfig struct {
	Type      string  `yaml:"type"`
	RateScale float64 `yaml:"rate_scale"`
}

type rawDesiredStateTemplates struct {
	DesiredStateTemplates []rawDesiredStateTemplate `yaml:"desired_state_templates"`
}

type rawDesiredStateTemplate struct {
	ID         string   `yaml:"id"`
	TargetTemp *float64 `yaml:"target_temp"`
	TargetHum  *float64 `yaml:"target_hum"`
}

type rawScheduleTemplates struct {
	ScheduleTemplates []rawScheduleTemplate `yaml:"schedule_templates"`
}

type rawScheduleTemplate struct {
	ID      string              `yaml:"id"`
	Name    string              `yaml:"name"`
	Periods []rawPeriodTemplate `yaml:"periods"`
}

type rawPeriodTemplate struct {
	DaysOfWeek []int    `yaml:"days_of_week"`
	StartTime  string   `yaml:"start_time"`
	EndTime    string   `yaml:"end_time"`
	Mode       string   `yaml:"mode"`
	TargetTemp *float64 `yaml:"target_temp"`
	TargetHum  *float64 `yaml:"target_hum"`
}

type rawSimulation struct {
	TemplateOverrides rawTemplateOverrides `yaml:"template_overrides"`
	Simulation        rawSimulationBlock   `yaml:"simulation"`
}

type rawTemplateOverrides struct {
	RoomTemplates   []rawRoomTemplate   `yaml:"room_templates"`
	DeviceTemplates []rawDeviceTemplate `yaml:"device_templates"`
}

type rawSimulationBlock struct {
	TimeScale            float64        `yaml:"time_scale"`
	MinPublishIntervalMS int            `yaml:"min_publish_interval_ms"`
	UserGroups           []rawUserGroup `yaml:"user_groups"`
}

type rawUserGroup struct {
	Count       int       `yaml:"count"`
	Interactive bool      `yaml:"interactive"`
	Rooms       []rawRoom `yaml:"rooms"`
}

type rawRoom struct {
	Template         string      `yaml:"template"`
	NamePrefix       string      `yaml:"name_prefix"`
	Count            int         `yaml:"count"`
	Type             string      `yaml:"type"`
	Noisy            bool        `yaml:"noisy"`
	ThermalMassScale *float64    `yaml:"thermal_mass_scale"`
	ConductanceScale *float64    `yaml:"conductance_scale"`
	Devices          []rawDevice `yaml:"devices"`
	// DesiredStateRef names a desired_state template. Inline TargetTemp and
	// TargetHum override the template values per field when present.
	DesiredStateRef string   `yaml:"desired_state"`
	TargetTemp      *float64 `yaml:"target_temp"`
	TargetHum       *float64 `yaml:"target_hum"`
	// ScheduleRef names a schedule template. Schedule content is not
	// overridable inline — use a separate template for different values.
	ScheduleRef string `yaml:"schedule"`
}

type rawDevice struct {
	Template   string   `yaml:"template"`
	NamePrefix string   `yaml:"name_prefix"`
	Count      int      `yaml:"count"`
	RateScale  *float64 `yaml:"rate_scale"`
}

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

// Load reads the templates and simulation file, applies simulation-local
// overrides, resolves template references into concrete structs, and returns a
// validated Config.
func Load(simulationName string) (*Config, error) {
	roomTemplates, err := loadRoomTemplates("/app/config/templates/rooms.yaml")
	if err != nil {
		return nil, fmt.Errorf("load room templates: %w", err)
	}

	deviceTemplates, err := loadDeviceTemplates("/app/config/templates/devices.yaml")
	if err != nil {
		return nil, fmt.Errorf("load device templates: %w", err)
	}

	desiredStateTemplates, err := loadDesiredStateTemplates("/app/config/templates/desired_states.yaml")
	if err != nil {
		return nil, fmt.Errorf("load desired state templates: %w", err)
	}

	scheduleTemplates, err := loadScheduleTemplates("/app/config/templates/schedules.yaml")
	if err != nil {
		return nil, fmt.Errorf("load schedule templates: %w", err)
	}

	rawSim, err := loadSimulation("/app/config/simulations/" + simulationName + ".yaml")
	if err != nil {
		return nil, fmt.Errorf("load simulation %q: %w", simulationName, err)
	}

	roomTemplates = applyRoomOverrides(roomTemplates, rawSim.TemplateOverrides.RoomTemplates)
	deviceTemplates = applyDeviceOverrides(deviceTemplates, rawSim.TemplateOverrides.DeviceTemplates)

	simulation, err := resolveSimulation(simulationName, rawSim.Simulation, roomTemplates, deviceTemplates, desiredStateTemplates, scheduleTemplates)
	if err != nil {
		return nil, fmt.Errorf("resolve simulation: %w", err)
	}

	if err := validateSimulation(simulation); err != nil {
		return nil, fmt.Errorf("invalid simulation %q: %w", simulationName, err)
	}

	baseTickSeconds, err := mustGetEnvInt("CONTROL_TICK_INTERVAL_SECONDS")
	if err != nil {
		return nil, err
	}

	minPublishInterval := time.Duration(simulation.MinPublishIntervalMS) * time.Millisecond
	naturalInterval := time.Duration(float64(time.Second) * float64(baseTickSeconds) / simulation.TimeScale)
	effectivePublishInterval := naturalInterval
	if effectivePublishInterval < minPublishInterval {
		effectivePublishInterval = minPublishInterval
	}
	simulatedTickSeconds := simulation.TimeScale * effectivePublishInterval.Seconds()

	return &Config{
		APIURL:                   "http://api-service:8080",
		MQTTHost:                 "mosquitto",
		MQTTPort:                 1883,
		MQTTClientID:             "sim-" + simulation.Name,
		MQTTUsername:             mustGetEnv("MQTT_DEVICE_USERNAME"),
		MQTTPassword:             mustGetEnv("MQTT_DEVICE_PASSWORD"),
		EmailTemplate:            mustGetEnv("SIMULATOR_EMAIL"),
		Password:                 mustGetEnv("SIMULATOR_PASSWORD"),
		BaseTickSeconds:          baseTickSeconds,
		EffectivePublishInterval: effectivePublishInterval,
		SimulatedTickSeconds:     simulatedTickSeconds,
		Simulation:               simulation,
	}, nil
}

// ---------------------------------------------------------------------------
// YAML loaders
// ---------------------------------------------------------------------------

func loadRoomTemplates(path string) (map[string]rawRoomTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw rawRoomTemplates
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	templates := make(map[string]rawRoomTemplate, len(raw.RoomTemplates))
	for _, t := range raw.RoomTemplates {
		templates[t.ID] = t
	}
	return templates, nil
}

func loadDeviceTemplates(path string) (map[string]rawDeviceTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw rawDeviceTemplates
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	templates := make(map[string]rawDeviceTemplate, len(raw.DeviceTemplates))
	for _, t := range raw.DeviceTemplates {
		templates[t.ID] = t
	}
	return templates, nil
}

func loadDesiredStateTemplates(path string) (map[string]rawDesiredStateTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw rawDesiredStateTemplates
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	templates := make(map[string]rawDesiredStateTemplate, len(raw.DesiredStateTemplates))
	for _, t := range raw.DesiredStateTemplates {
		templates[t.ID] = t
	}
	return templates, nil
}

func loadScheduleTemplates(path string) (map[string]rawScheduleTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw rawScheduleTemplates
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	templates := make(map[string]rawScheduleTemplate, len(raw.ScheduleTemplates))
	for _, t := range raw.ScheduleTemplates {
		templates[t.ID] = t
	}
	return templates, nil
}

func loadSimulation(path string) (*rawSimulation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw rawSimulation
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &raw, nil
}

// ---------------------------------------------------------------------------
// Override merging
// ---------------------------------------------------------------------------

func applyRoomOverrides(base map[string]rawRoomTemplate, overrides []rawRoomTemplate) map[string]rawRoomTemplate {
	if len(overrides) == 0 {
		return base
	}
	merged := make(map[string]rawRoomTemplate, len(base))
	for k, v := range base {
		merged[k] = v
	}
	for _, o := range overrides {
		merged[o.ID] = o
	}
	return merged
}

func applyDeviceOverrides(base map[string]rawDeviceTemplate, overrides []rawDeviceTemplate) map[string]rawDeviceTemplate {
	if len(overrides) == 0 {
		return base
	}
	merged := make(map[string]rawDeviceTemplate, len(base))
	for k, v := range base {
		merged[k] = v
	}
	for _, o := range overrides {
		merged[o.ID] = o
	}
	return merged
}

// ---------------------------------------------------------------------------
// Resolution
// ---------------------------------------------------------------------------

func resolveSimulation(name string, raw rawSimulationBlock, roomTpls map[string]rawRoomTemplate, devTpls map[string]rawDeviceTemplate, dsTpls map[string]rawDesiredStateTemplate, schedTpls map[string]rawScheduleTemplate) (Simulation, error) {
	timeScale := raw.TimeScale
	if timeScale <= 0 {
		timeScale = defaultTimeScale
	}
	if timeScale > maxTimeScale {
		return Simulation{}, fmt.Errorf("time_scale %.0f exceeds maximum of %d", timeScale, maxTimeScale)
	}

	minPublishIntervalMS := raw.MinPublishIntervalMS
	if minPublishIntervalMS <= 0 {
		minPublishIntervalMS = defaultMinPublishIntervalMS
	}

	groups := make([]UserGroup, 0, len(raw.UserGroups))

	for _, rg := range raw.UserGroups {
		rooms := make([]Room, 0, len(rg.Rooms))

		for _, rr := range rg.Rooms {
			tpl, ok := roomTpls[rr.Template]
			if !ok {
				return Simulation{}, fmt.Errorf("room template %q not found", rr.Template)
			}

			roomType := rr.Type
			if roomType == "" {
				roomType = defaultRoomType
			}

			measurements := resolveMeasurements(tpl.Measurements, rr.Noisy, rr.ThermalMassScale, rr.ConductanceScale)

			devices := make([]Device, 0, len(rr.Devices))
			for _, rd := range rr.Devices {
				dtpl, ok := devTpls[rd.Template]
				if !ok {
					return Simulation{}, fmt.Errorf("device template %q not found", rd.Template)
				}
				devices = append(devices, resolveDevice(rd, dtpl))
			}

			ds, err := resolveDesiredState(rr, dsTpls)
			if err != nil {
				return Simulation{}, fmt.Errorf("room %q: %w", rr.NamePrefix, err)
			}

			sched, err := resolveSchedule(rr, schedTpls)
			if err != nil {
				return Simulation{}, fmt.Errorf("room %q: %w", rr.NamePrefix, err)
			}

			if ds != nil && sched != nil {
				return Simulation{}, fmt.Errorf("room %q: desired_state and schedule are mutually exclusive", rr.NamePrefix)
			}

			rooms = append(rooms, Room{
				NamePrefix:   rr.NamePrefix,
				Count:        rr.Count,
				Type:         roomType,
				Noisy:        rr.Noisy,
				Measurements: measurements,
				Devices:      devices,
				DesiredState: ds,
				Schedule:     sched,
			})
		}

		groups = append(groups, UserGroup{
			Count:       rg.Count,
			Interactive: rg.Interactive,
			Rooms:       rooms,
		})
	}

	return Simulation{
		Name:                 name,
		TimeScale:            timeScale,
		MinPublishIntervalMS: minPublishIntervalMS,
		UserGroups:           groups,
	}, nil
}

// resolveDesiredState builds a ResolvedDesiredState from a raw room entry.
// If a template ref is present, its values are used as the base. Inline
// TargetTemp and TargetHum override the template values per field when present.
// Returns nil if neither a template ref nor inline values are specified.
func resolveDesiredState(rr rawRoom, tpls map[string]rawDesiredStateTemplate) (*ResolvedDesiredState, error) {
	var base rawDesiredStateTemplate

	if rr.DesiredStateRef != "" {
		tpl, ok := tpls[rr.DesiredStateRef]
		if !ok {
			return nil, fmt.Errorf("desired_state template %q not found", rr.DesiredStateRef)
		}
		base = tpl
	}

	targetTemp := base.TargetTemp
	if rr.TargetTemp != nil {
		targetTemp = rr.TargetTemp
	}

	targetHum := base.TargetHum
	if rr.TargetHum != nil {
		targetHum = rr.TargetHum
	}

	if targetTemp == nil && targetHum == nil {
		return nil, nil
	}

	return &ResolvedDesiredState{
		TargetTemp: targetTemp,
		TargetHum:  targetHum,
	}, nil
}

// resolveSchedule builds a ResolvedSchedule from a raw room entry's schedule
// template reference. Returns nil if no schedule ref is specified.
func resolveSchedule(rr rawRoom, tpls map[string]rawScheduleTemplate) (*ResolvedSchedule, error) {
	if rr.ScheduleRef == "" {
		return nil, nil
	}

	tpl, ok := tpls[rr.ScheduleRef]
	if !ok {
		return nil, fmt.Errorf("schedule template %q not found", rr.ScheduleRef)
	}

	periods := make([]ResolvedPeriod, len(tpl.Periods))
	for i, p := range tpl.Periods {
		periods[i] = ResolvedPeriod{
			DaysOfWeek: p.DaysOfWeek,
			StartTime:  p.StartTime,
			EndTime:    p.EndTime,
			Mode:       p.Mode,
			TargetTemp: p.TargetTemp,
			TargetHum:  p.TargetHum,
		}
	}

	return &ResolvedSchedule{
		Name:    tpl.Name,
		Periods: periods,
	}, nil
}

// resolveMeasurements builds the MeasurementConfig map from a raw room
// template. Scale fields from the simulation file override template scales
// when provided. Final physical values are derived by multiplying the resolved
// scale against the base physical constant for each measurement type. Noise is
// zeroed if the room is not noisy.
func resolveMeasurements(raw map[string]rawMeasurementConfig, noisy bool, simThermalMassScale, simConductanceScale *float64) map[string]MeasurementConfig {
	resolved := make(map[string]MeasurementConfig, len(raw))

	for typ, m := range raw {
		thermalMassScale := m.ThermalMassScale
		if thermalMassScale == 0 {
			thermalMassScale = 1.0
		}
		if simThermalMassScale != nil {
			thermalMassScale = *simThermalMassScale
		}

		conductanceScale := m.ConductanceScale
		if conductanceScale == 0 {
			conductanceScale = 1.0
		}
		if simConductanceScale != nil {
			conductanceScale = *simConductanceScale
		}

		noise := m.Noise
		if !noisy {
			noise = 0
		}

		resolved[typ] = MeasurementConfig{
			Base:        m.Base,
			ThermalMass: baseThermalMass(typ) * thermalMassScale,
			Conductance: baseConductance(typ) * conductanceScale,
			Noise:       noise,
		}
	}

	return resolved
}

// resolveDevice builds a Device from a raw device entry and its template.
// The simulation file rate_scale overrides the template rate_scale when
// provided. Sensor noise is always applied as a hardware characteristic —
// it is not affected by the room noisy flag.
func resolveDevice(rd rawDevice, tpl rawDeviceTemplate) Device {
	sensors := make([]SensorConfig, len(tpl.Sensors))
	for i, s := range tpl.Sensors {
		sensors[i] = SensorConfig{
			Type:   s.Type,
			Noise:  s.Noise,
			Offset: s.Offset,
		}
	}

	actuators := make([]ActuatorConfig, len(tpl.Actuators))
	for i, a := range tpl.Actuators {
		rateScale := a.RateScale
		if rateScale == 0 {
			rateScale = 1.0
		}
		if rd.RateScale != nil {
			rateScale = *rd.RateScale
		}
		actuators[i] = ActuatorConfig{
			Type: a.Type,
			Rate: baseRate(a.Type) * rateScale,
		}
	}

	return Device{
		NamePrefix: rd.NamePrefix,
		Count:      rd.Count,
		Sensors:    sensors,
		Actuators:  actuators,
	}
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func validateSimulation(sim Simulation) error {
	for i, group := range sim.UserGroups {
		roomPrefixes := make(map[string]bool, len(group.Rooms))
		for _, rm := range group.Rooms {
			if roomPrefixes[rm.NamePrefix] {
				return fmt.Errorf("user group %d has duplicate room name_prefix %q", i, rm.NamePrefix)
			}
			roomPrefixes[rm.NamePrefix] = true

			devicePrefixes := make(map[string]bool, len(rm.Devices))
			for _, dev := range rm.Devices {
				if devicePrefixes[dev.NamePrefix] {
					return fmt.Errorf("room %q in user group %d has duplicate device name_prefix %q", rm.NamePrefix, i, dev.NamePrefix)
				}
				devicePrefixes[dev.NamePrefix] = true
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Env helpers
// ---------------------------------------------------------------------------

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %q is not set", key))
	}
	return v
}

func mustGetEnvInt(key string) (int, error) {
	v := mustGetEnv(key)
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("env var %q must be an integer: %w", key, err)
	}
	return n, nil
}
