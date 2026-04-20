// Package provisioning bootstraps simulator users, rooms, and devices against
// the api-service. It is idempotent: Run logs in first and only registers on
// 401, and treats 409 responses from room/device creation as "already exists"
// and falls back to lookup. Interactive user groups emit a credentials file
// under /app/config/credentials for manual login.
package provisioning

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/DiegoJohnson25/climate-control/simulator-service/internal/api"
	"github.com/DiegoJohnson25/climate-control/simulator-service/internal/config"
)

// ---------------------------------------------------------------------------
// Provisioned types
// ---------------------------------------------------------------------------

type ProvisionedDevice struct {
	HwID   string
	Config config.Device
}

type ProvisionedRoom struct {
	ID      string
	Name    string
	Config  config.Room
	Devices []ProvisionedDevice
}

type ProvisionedUser struct {
	Rooms []ProvisionedRoom
}

// ---------------------------------------------------------------------------
// Run
// ---------------------------------------------------------------------------

// Run provisions every user group in cfg, returning a ProvisionedUser per user
// with its rooms and devices. Callers pass the result to simulator.Run, which
// uses it to drive per-device publish loops.
func Run(cfg *config.Config) ([]ProvisionedUser, error) {
	client := api.NewClient(cfg.APIURL)
	domain := emailDomain(cfg.EmailTemplate)

	var users []ProvisionedUser
	globalUserIdx := 0

	for _, group := range cfg.Simulation.UserGroups {
		var groupUsers []provisionedUserWithCreds

		for i := 0; i < group.Count; i++ {
			email := generateEmail(cfg.Simulation.Name, globalUserIdx, domain)
			password := cfg.Password

			token, err := client.Login(email, password)
			if err != nil {
				if err := client.Register(email, password); err != nil {
					return nil, fmt.Errorf("register user %s: %w", email, err)
				}
				token, err = client.Login(email, password)
				if err != nil {
					return nil, fmt.Errorf("login user %s after register: %w", email, err)
				}
			}

			existingRooms, err := client.ListRooms(token)
			if err != nil {
				return nil, fmt.Errorf("list rooms for user %s: %w", email, err)
			}
			existingDevices, err := client.ListDevices(token)
			if err != nil {
				return nil, fmt.Errorf("list devices for user %s: %w", email, err)
			}

			roomsByName := make(map[string]string, len(existingRooms))
			for _, rm := range existingRooms {
				roomsByName[rm.Name] = rm.ID
			}
			devicesByHwID := make(map[string]string, len(existingDevices))
			for _, dev := range existingDevices {
				devicesByHwID[dev.HwID] = dev.ID
			}

			rooms, err := provisionRooms(client, token, cfg.Simulation.Name, globalUserIdx, group.Rooms, roomsByName, devicesByHwID)
			if err != nil {
				return nil, fmt.Errorf("provision rooms for user %s: %w", email, err)
			}

			if group.Interactive {
				groupUsers = append(groupUsers, provisionedUserWithCreds{
					email:    email,
					password: password,
					user:     ProvisionedUser{Rooms: rooms},
				})
			}

			users = append(users, ProvisionedUser{Rooms: rooms})
			globalUserIdx++
		}

		if group.Interactive {
			if err := writeCredentials(cfg.Simulation.Name, groupUsers); err != nil {
				return nil, fmt.Errorf("write credentials: %w", err)
			}
		}
	}

	return users, nil
}

// ---------------------------------------------------------------------------
// Room and device provisioning
// ---------------------------------------------------------------------------

func provisionRooms(client *api.Client, token, simName string, userIdx int, roomDefs []config.Room, roomsByName, devicesByHwID map[string]string) ([]ProvisionedRoom, error) {
	var rooms []ProvisionedRoom
	localRoomIdx := 0

	for _, def := range roomDefs {
		for i := 0; i < def.Count; i++ {
			roomName := fmt.Sprintf("%s-%d", def.NamePrefix, localRoomIdx)

			roomID, err := client.CreateRoom(token, roomName)
			if err != nil {
				if err != api.ErrConflict {
					return nil, fmt.Errorf("create room %s: %w", roomName, err)
				}
				id, ok := roomsByName[roomName]
				if !ok {
					return nil, fmt.Errorf("room %q not found in list after conflict", roomName)
				}
				roomID = id
			}

			devices, err := provisionDevices(client, token, simName, userIdx, localRoomIdx, roomID, roomName, def.Devices, devicesByHwID)
			if err != nil {
				return nil, fmt.Errorf("provision devices for room %s: %w", roomName, err)
			}

			rooms = append(rooms, ProvisionedRoom{
				ID:      roomID,
				Name:    roomName,
				Config:  def,
				Devices: devices,
			})

			localRoomIdx++
		}
	}

	return rooms, nil
}

func provisionDevices(client *api.Client, token, simName string, userIdx, roomIdx int, roomID, roomName string, deviceDefs []config.Device, devicesByHwID map[string]string) ([]ProvisionedDevice, error) {
	var devices []ProvisionedDevice
	globalDeviceIdx := 0

	for _, def := range deviceDefs {
		for i := 0; i < def.Count; i++ {
			deviceName := fmt.Sprintf("%s-%s-%d", roomName, def.NamePrefix, i)
			hwID := generateHwID(simName, userIdx, roomIdx, globalDeviceIdx)

			deviceID, err := client.CreateDevice(token, deviceName, hwID, def.Sensors, def.Actuators)
			if err != nil {
				if err != api.ErrConflict {
					return nil, fmt.Errorf("create device %s: %w", deviceName, err)
				}
				id, ok := devicesByHwID[hwID]
				if !ok {
					return nil, fmt.Errorf("device with hw_id %q not found in list after conflict", hwID)
				}
				deviceID = id
			}

			if err := client.AssignDevice(token, deviceName, deviceID, roomID); err != nil {
				return nil, fmt.Errorf("assign device %s to room %s: %w", deviceName, roomName, err)
			}

			devices = append(devices, ProvisionedDevice{
				HwID:   hwID,
				Config: def,
			})

			globalDeviceIdx++
		}
	}

	return devices, nil
}

// ---------------------------------------------------------------------------
// Identity generation
// ---------------------------------------------------------------------------

func generateEmail(simName string, userIdx int, domain string) string {
	return fmt.Sprintf("sim-%s-user-%03d@%s", simName, userIdx, domain)
}

func generateHwID(simName string, userIdx, roomIdx, deviceIdx int) string {
	return fmt.Sprintf("sim-%s-%d-%d-%d", simName, userIdx, roomIdx, deviceIdx)
}

func emailDomain(emailTemplate string) string {
	parts := strings.SplitN(emailTemplate, "@", 2)
	if len(parts) != 2 {
		return "local.dev"
	}
	return parts[1]
}

// ---------------------------------------------------------------------------
// Credentials file
// ---------------------------------------------------------------------------

type provisionedUserWithCreds struct {
	email    string
	password string
	user     ProvisionedUser
}

const credentialsTemplate = `Simulation: {{.SimName}}
Generated:  {{.Generated}}
{{range $i, $u := .Users}}
--- User {{$i}} ---
Email:    {{$u.Email}}
Password: {{$u.Password}}

Rooms:
{{range $u.User.Rooms}}  {{.Name}}
    Devices:
{{range .Devices}}      {{.HwID}}{{if .Config.Sensors}} (sensors: {{join .Config.Sensors ", "}}){{end}}{{if .Config.Actuators}} (actuators: {{join .Config.Actuators ", "}}){{end}}
{{end}}{{end}}{{end}}`

func writeCredentials(simName string, users []provisionedUserWithCreds) error {
	type userEntry struct {
		Email    string
		Password string
		User     ProvisionedUser
	}
	type templateData struct {
		SimName   string
		Generated string
		Users     []userEntry
	}

	entries := make([]userEntry, len(users))
	for i, u := range users {
		entries[i] = userEntry{Email: u.email, Password: u.password, User: u.user}
	}

	funcMap := template.FuncMap{
		"join": strings.Join,
	}

	tmpl, err := template.New("credentials").Funcs(funcMap).Parse(credentialsTemplate)
	if err != nil {
		return fmt.Errorf("parse credentials template: %w", err)
	}

	path := "/app/config/credentials/" + simName + ".txt"
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create credentials file: %w", err)
	}
	defer f.Close()

	return tmpl.Execute(f, templateData{
		SimName:   simName,
		Generated: time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		Users:     entries,
	})
}
