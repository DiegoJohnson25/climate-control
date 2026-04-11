package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

// -----------------------------------------------------------------------------
// Output types
// -----------------------------------------------------------------------------

type Config struct {
	APIURL        string
	MQTTHost      string
	MQTTPort      int
	MQTTClientID  string
	MQTTUsername  string
	MQTTPassword  string
	EmailTemplate string
	Password      string
	Simulation    Simulation
}

type Simulation struct {
	Name                string
	TickIntervalSeconds int
	UserGroups          []UserGroup
}

type UserGroup struct {
	Count       int
	Interactive bool
	Rooms       []Room
}

type Room struct {
	NamePrefix string
	Count      int
	Model      RoomModel
	Devices    []Device
}

type RoomModel struct {
	Type         string
	BaseTemp     float64
	BaseHumidity float64
	Noise        map[string]float64
	// TODO: Add other room type pararmeters for more complex models when added.
}

type Device struct {
	NamePrefix string
	Count      int
	Sensors    []string
	Actuators  []string
	Noise      map[string]float64
	Offset     map[string]float64
}

// -----------------------------------------------------------------------------
// Raw YAML types
// -----------------------------------------------------------------------------

type rawRoomTemplates struct {
	RoomTemplates []rawRoomTemplate `yaml:"room_templates"`
}

type rawRoomTemplate struct {
	ID    string       `yaml:"id"`
	Model rawRoomModel `yaml:"model"`
}

type rawRoomModel struct {
	Type         string             `yaml:"type"`
	BaseTemp     float64            `yaml:"base_temp"`
	BaseHumidity float64            `yaml:"base_humidity"`
	Noise        map[string]float64 `yaml:"noise"`
	// TODO: Add other room type pararmeters for more complex models when added.
}

type rawDeviceTemplates struct {
	DeviceTemplates []rawDeviceTemplate `yaml:"device_templates"`
}

type rawDeviceTemplate struct {
	ID        string             `yaml:"id"`
	Sensors   []string           `yaml:"sensors"`
	Actuators []string           `yaml:"actuators"`
	Noise     map[string]float64 `yaml:"noise"`
	Offset    map[string]float64 `yaml:"offset"`
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
	TickIntervalSeconds int            `yaml:"tick_interval_seconds"`
	UserGroups          []rawUserGroup `yaml:"user_groups"`
}

type rawUserGroup struct {
	Count       int       `yaml:"count"`
	Interactive bool      `yaml:"interactive"`
	Rooms       []rawRoom `yaml:"rooms"`
}

type rawRoom struct {
	Template   string      `yaml:"template"`
	NamePrefix string      `yaml:"name_prefix"`
	Count      int         `yaml:"count"`
	Devices    []rawDevice `yaml:"devices"`
}

type rawDevice struct {
	Template   string `yaml:"template"`
	NamePrefix string `yaml:"name_prefix"`
	Count      int    `yaml:"count"`
}

// -----------------------------------------------------------------------------
// Load — the single entry point for config
// -----------------------------------------------------------------------------

func Load(simulationName string) (*Config, error) {
	roomTemplates, err := loadRoomTemplates("/app/config/templates/rooms.yaml")
	if err != nil {
		return nil, fmt.Errorf("load room templates: %w", err)
	}

	deviceTemplates, err := loadDeviceTemplates("/app/config/templates/devices.yaml")
	if err != nil {
		return nil, fmt.Errorf("load device templates: %w", err)
	}

	rawSim, err := loadSimulation("/app/config/simulations/" + simulationName + ".yaml")
	if err != nil {
		return nil, fmt.Errorf("load simulation %q: %w", simulationName, err)
	}

	roomTemplates = applyRoomOverrides(roomTemplates, rawSim.TemplateOverrides.RoomTemplates)
	deviceTemplates = applyDeviceOverrides(deviceTemplates, rawSim.TemplateOverrides.DeviceTemplates)

	simulation, err := resolveSimulation(simulationName, rawSim.Simulation, roomTemplates, deviceTemplates)
	if err != nil {
		return nil, fmt.Errorf("resolve simulation: %w", err)
	}

	if err := validateSimulation(simulation); err != nil {
		return nil, fmt.Errorf("invalid simulation %q: %w", simulationName, err)
	}

	return &Config{
		APIURL:        "http://api-service:8080",
		MQTTHost:      "mosquitto",
		MQTTPort:      1883,
		MQTTClientID:  "sim-" + simulation.Name,
		MQTTUsername:  mustGetEnv("MQTT_DEVICE_USERNAME"),
		MQTTPassword:  mustGetEnv("MQTT_DEVICE_PASSWORD"),
		EmailTemplate: mustGetEnv("SIMULATOR_EMAIL"),
		Password:      mustGetEnv("SIMULATOR_PASSWORD"),
		Simulation:    simulation,
	}, nil
}

// -----------------------------------------------------------------------------
// YAML loaders
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Override merging — simulation-local overrides win on id collision
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Resolution — dissolves template references into concrete structs
// -----------------------------------------------------------------------------

func resolveSimulation(name string, raw rawSimulationBlock, roomTpls map[string]rawRoomTemplate, devTpls map[string]rawDeviceTemplate) (Simulation, error) {
	groups := make([]UserGroup, 0, len(raw.UserGroups))

	for _, rg := range raw.UserGroups {
		rooms := make([]Room, 0, len(rg.Rooms))

		for _, rr := range rg.Rooms {
			tpl, ok := roomTpls[rr.Template]
			if !ok {
				return Simulation{}, fmt.Errorf("room template %q not found", rr.Template)
			}

			devices := make([]Device, 0, len(rr.Devices))
			for _, rd := range rr.Devices {
				dtpl, ok := devTpls[rd.Template]
				if !ok {
					return Simulation{}, fmt.Errorf("device template %q not found", rd.Template)
				}
				devices = append(devices, Device{
					NamePrefix: rd.NamePrefix,
					Count:      rd.Count,
					Sensors:    dtpl.Sensors,
					Actuators:  dtpl.Actuators,
					Noise:      dtpl.Noise,
					Offset:     dtpl.Offset,
				})
			}

			rooms = append(rooms, Room{
				NamePrefix: rr.NamePrefix,
				Count:      rr.Count,
				Model: RoomModel{
					Type:         tpl.Model.Type,
					BaseTemp:     tpl.Model.BaseTemp,
					BaseHumidity: tpl.Model.BaseHumidity,
					Noise:        tpl.Model.Noise,
				},
				Devices: devices,
			})
		}

		groups = append(groups, UserGroup{
			Count:       rg.Count,
			Interactive: rg.Interactive,
			Rooms:       rooms,
		})
	}

	return Simulation{
		Name:                name,
		TickIntervalSeconds: raw.TickIntervalSeconds,
		UserGroups:          groups,
	}, nil
}

// -----------------------------------------------------------------------------
// Validation
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Env helpers
// -----------------------------------------------------------------------------

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %q is not set", key))
	}
	return v
}
