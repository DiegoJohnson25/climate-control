// Package api is the simulator-service's HTTP client for the api-service. It
// wraps register/login and the room, device, desired state, and schedule
// endpoints used during provisioning. The client is idempotent at the caller
// level — callers inspect ErrConflict and fall back to lookup where appropriate.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}

type RoomResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DeviceResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	HwID string `json:"hw_id"`
}

// ScheduleResponse carries the fields needed for idempotent schedule lookup.
type ScheduleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrConflict           = fmt.Errorf("conflict")
	ErrCapabilityConflict = fmt.Errorf("capability conflict")
)

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

func (c *Client) Register(email, password string) error {
	body := map[string]string{"email": email, "password": password}
	resp, err := c.post("/api/v1/auth/register", "", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("register failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) Login(email, password string) (string, error) {
	body := map[string]string{"email": email, "password": password}
	resp, err := c.post("/api/v1/auth/login", "", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed: status %d", resp.StatusCode)
	}
	var ar AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return "", fmt.Errorf("decode login response: %w", err)
	}
	return ar.AccessToken, nil
}

func (c *Client) DeleteMe(token string) error {
	resp, err := c.delete("/api/v1/users/me", token)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete user failed: status %d", resp.StatusCode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Rooms
// ---------------------------------------------------------------------------

func (c *Client) CreateRoom(token, name string) (string, error) {
	body := map[string]string{"name": name}
	resp, err := c.post("/api/v1/rooms", token, body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return "", ErrConflict
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create room failed: status %d", resp.StatusCode)
	}
	var rm RoomResponse
	if err := json.NewDecoder(resp.Body).Decode(&rm); err != nil {
		return "", fmt.Errorf("decode room response: %w", err)
	}
	return rm.ID, nil
}

func (c *Client) ListRooms(token string) ([]RoomResponse, error) {
	resp, err := c.get("/api/v1/rooms", token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list rooms failed: status %d", resp.StatusCode)
	}
	var rooms []RoomResponse
	if err := json.NewDecoder(resp.Body).Decode(&rooms); err != nil {
		return nil, fmt.Errorf("decode rooms response: %w", err)
	}
	return rooms, nil
}

// ---------------------------------------------------------------------------
// Devices
// ---------------------------------------------------------------------------

func (c *Client) CreateDevice(token, name, hwID string, sensors, actuators []string) (string, error) {
	body := map[string]any{
		"name":      name,
		"hw_id":     hwID,
		"sensors":   sensors,
		"actuators": actuators,
	}
	resp, err := c.post("/api/v1/devices", token, body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return "", ErrConflict
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create device failed: status %d", resp.StatusCode)
	}
	var dev DeviceResponse
	if err := json.NewDecoder(resp.Body).Decode(&dev); err != nil {
		return "", fmt.Errorf("decode device response: %w", err)
	}
	return dev.ID, nil
}

func (c *Client) ListDevices(token string) ([]DeviceResponse, error) {
	resp, err := c.get("/api/v1/devices", token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list devices failed: status %d", resp.StatusCode)
	}
	var devices []DeviceResponse
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		return nil, fmt.Errorf("decode devices response: %w", err)
	}
	return devices, nil
}

func (c *Client) AssignDevice(token, name, deviceID, roomID string) error {
	body := map[string]string{"name": name, "room_id": roomID}
	resp, err := c.put("/api/v1/devices/"+deviceID, token, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("assign device failed: status %d", resp.StatusCode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Desired state
// ---------------------------------------------------------------------------

// UpdateDesiredState sets AUTO mode with an indefinite manual override on the
// given room. At least one of targetTemp and targetHum must be non-nil.
func (c *Client) UpdateDesiredState(token, roomID string, targetTemp, targetHum *float64) error {
	overrideUntil := "indefinite"
	body := map[string]any{
		"mode":                  "AUTO",
		"manual_active":         true,
		"manual_override_until": overrideUntil,
		"target_temp":           targetTemp,
		"target_hum":            targetHum,
	}
	resp, err := c.put("/api/v1/rooms/"+roomID+"/desired-state", token, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update desired state failed: status %d", resp.StatusCode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Schedules
// ---------------------------------------------------------------------------

// CreateSchedule creates a named schedule for the given room. Returns
// ErrConflict if a schedule with that name already exists for the room.
func (c *Client) CreateSchedule(token, roomID, name string) (string, error) {
	body := map[string]string{"name": name}
	resp, err := c.post("/api/v1/rooms/"+roomID+"/schedules", token, body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return "", ErrConflict
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create schedule failed: status %d", resp.StatusCode)
	}
	var sched ScheduleResponse
	if err := json.NewDecoder(resp.Body).Decode(&sched); err != nil {
		return "", fmt.Errorf("decode schedule response: %w", err)
	}
	return sched.ID, nil
}

// ListSchedules returns all schedules for the given room. Used for idempotent
// schedule upsert — on ErrConflict from CreateSchedule, find the existing
// schedule by name in this list.
func (c *Client) ListSchedules(token, roomID string) ([]ScheduleResponse, error) {
	resp, err := c.get("/api/v1/rooms/"+roomID+"/schedules", token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list schedules failed: status %d", resp.StatusCode)
	}
	var scheds []ScheduleResponse
	if err := json.NewDecoder(resp.Body).Decode(&scheds); err != nil {
		return nil, fmt.Errorf("decode schedules response: %w", err)
	}
	return scheds, nil
}

// CreatePeriod adds a period to the given schedule. Returns ErrConflict if the
// period overlaps an existing period — treated as already provisioned by callers.
func (c *Client) CreatePeriod(token, scheduleID string, daysOfWeek []int, startTime, endTime, mode string, targetTemp, targetHum *float64) error {
	body := map[string]any{
		"days_of_week": daysOfWeek,
		"start_time":   startTime,
		"end_time":     endTime,
		"mode":         mode,
		"target_temp":  targetTemp,
		"target_hum":   targetHum,
	}
	resp, err := c.post("/api/v1/schedules/"+scheduleID+"/periods", token, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return ErrConflict
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("create period failed: status %d", resp.StatusCode)
	}
	return nil
}

// ActivateSchedule activates the given schedule. Returns ErrConflict if the
// schedule is already active — treated as already provisioned by callers.
// Returns ErrCapabilityConflict if the room lacks the required devices for one
// or more periods — this is a fatal provisioning error.
func (c *Client) ActivateSchedule(token, scheduleID string) error {
	resp, err := c.patch("/api/v1/schedules/"+scheduleID+"/activate", token)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		var body struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return fmt.Errorf("activate schedule failed: status 409, could not decode body: %w", err)
		}
		if body.Error == "room lacks required capability for one or more periods" {
			return ErrCapabilityConflict
		}
		// ErrAlreadyActive — already activated, treat as success
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("activate schedule failed: status %d", resp.StatusCode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

func (c *Client) post(path, token string, body any) (*http.Response, error) {
	return c.do(http.MethodPost, path, token, body)
}

func (c *Client) put(path, token string, body any) (*http.Response, error) {
	return c.do(http.MethodPut, path, token, body)
}

func (c *Client) patch(path, token string) (*http.Response, error) {
	return c.do(http.MethodPatch, path, token, nil)
}

func (c *Client) get(path, token string) (*http.Response, error) {
	return c.do(http.MethodGet, path, token, nil)
}

func (c *Client) delete(path, token string) (*http.Response, error) {
	return c.do(http.MethodDelete, path, token, nil)
}

func (c *Client) do(method, path, token string, body any) (*http.Response, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}
	req, err := http.NewRequest(method, c.baseURL+path, &buf)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.httpClient.Do(req)
}
