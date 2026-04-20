// Package api is the simulator-service's HTTP client for the api-service. It
// wraps register/login and the room and device endpoints used during
// provisioning. The client is idempotent at the caller level — callers inspect
// ErrConflict and fall back to lookup where appropriate.
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

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var ErrConflict = fmt.Errorf("conflict")

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
// HTTP helpers
// ---------------------------------------------------------------------------

func (c *Client) post(path, token string, body any) (*http.Response, error) {
	return c.do(http.MethodPost, path, token, body)
}

func (c *Client) put(path, token string, body any) (*http.Response, error) {
	return c.do(http.MethodPut, path, token, body)
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
