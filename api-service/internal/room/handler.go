package room

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/ctxkeys"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ---------------------------------------------------------------------------
// Request types
// ---------------------------------------------------------------------------

type roomRequest struct {
	Name         string   `json:"name"          binding:"required"`
	DeadbandTemp *float64 `json:"deadband_temp" binding:"omitempty,gt=0"`
	DeadbandHum  *float64 `json:"deadband_hum"  binding:"omitempty,gt=0"`
}

// updateDesiredStateRequest uses a raw json.RawMessage for manual_override so
// the handler can distinguish between the key being absent, null, a timestamp
// string, or "indefinite". Gin's ShouldBindJSON leaves RawMessage nil when the
// key is absent entirely; explicit null arrives as the literal bytes `null`.
type updateDesiredStateRequest struct {
	Mode         string   `json:"mode" binding:"required,oneof=OFF AUTO"`
	ManualActive bool     `json:"manual_active"`
	TargetTemp   *float64 `json:"target_temp"`
	TargetHum    *float64 `json:"target_hum"`
	// "indefinite", an RFC3339 string, or null (JSON null or key absent = clear override)
	ManualOverride *string `json:"manual_override_until"`
}

// ---------------------------------------------------------------------------
// Room CRUD
// ---------------------------------------------------------------------------

func (h *Handler) List(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	rms, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	resp := make([]gin.H, len(rms))
	for i, rm := range rms {
		resp[i] = roomResponse(rm)
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Create(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	var req roomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rm, err := h.svc.Create(c.Request.Context(), userID, req.Name)
	if err != nil {
		if errors.Is(err, ErrNameTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": ErrNameTaken.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, roomResponse(RoomWithCapabilities{Room: *rm}))
}

func (h *Handler) Get(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	rm, err := h.svc.GetByID(c.Request.Context(), roomID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, roomResponse(*rm))
}

func (h *Handler) Update(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	var req roomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rm, err := h.svc.Update(c.Request.Context(), roomID, userID, req.Name, req.DeadbandTemp, req.DeadbandHum)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
			return
		}
		if errors.Is(err, ErrNameTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": ErrNameTaken.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, roomResponse(*rm))
}

func (h *Handler) Delete(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), roomID, userID); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Desired state
// ---------------------------------------------------------------------------

func (h *Handler) GetDesiredState(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	ds, err := h.svc.GetDesiredState(c.Request.Context(), roomID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, desiredStateResponse(ds))
}

func (h *Handler) UpdateDesiredState(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	var req updateDesiredStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	overrideUntil, err := resolveManualOverride(req.ManualOverride)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": ErrInvalidOverride.Error()})
		return
	}

	input := UpdateDesiredStateInput{
		Mode:                req.Mode,
		ManualActive:        req.ManualActive,
		TargetTemp:          req.TargetTemp,
		TargetHum:           req.TargetHum,
		ManualOverrideUntil: overrideUntil,
	}

	ds, err := h.svc.UpdateDesiredState(c.Request.Context(), roomID, userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
		case errors.Is(err, ErrInvalidTarget):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": ErrInvalidTarget.Error()})
		case errors.Is(err, ErrInvalidState):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": ErrInvalidState.Error()})
		case errors.Is(err, ErrNoCapability):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": ErrNoCapability.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, desiredStateResponse(ds))
}

// ---------------------------------------------------------------------------
// Climate
// ---------------------------------------------------------------------------

func (h *Handler) GetClimate(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	reading, err := h.svc.GetClimate(c.Request.Context(), roomID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	if reading == nil {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, reading)
}

func (h *Handler) GetClimateHistory(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	window := c.Query("window")
	if window != "" {
		switch window {
		case "1h", "6h", "24h", "7d":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid window: must be one of 1h, 6h, 24h, 7d"})
			return
		}
	}

	effectiveWindow := window
	if effectiveWindow == "" {
		effectiveWindow = "24h"
	}

	var density int
	if raw := c.Query("density"); raw != "" {
		n, parseErr := strconv.Atoi(raw)
		if parseErr != nil || n <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid density: must be a positive integer"})
			return
		}
		density = n
	}

	result, err := h.svc.GetClimateHistory(c.Request.Context(), roomID, userID, effectiveWindow, density)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"window":         effectiveWindow,
		"bucket_seconds": result.BucketSeconds,
		"points":         result.Points,
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func roomResponse(rm RoomWithCapabilities) gin.H {
	return gin.H{
		"id":            rm.ID,
		"name":          rm.Name,
		"deadband_temp": rm.DeadbandTemp,
		"deadband_hum":  rm.DeadbandHum,
		"created_at":    rm.CreatedAt,
		"updated_at":    rm.UpdatedAt,
		"capabilities": gin.H{
			"temperature": rm.Capabilities.Temperature,
			"humidity":    rm.Capabilities.Humidity,
		},
	}
}

func desiredStateResponse(ds models.DesiredState) gin.H {
	resp := gin.H{
		"id":                    ds.ID,
		"room_id":               ds.RoomID,
		"mode":                  ds.Mode,
		"manual_active":         ds.ManualActive,
		"target_temp":           ds.TargetTemp,
		"target_hum":            ds.TargetHum,
		"manual_override_until": ds.ManualOverrideUntil,
		"updated_at":            ds.UpdatedAt,
	}

	// surface the indefinite sentinel as a readable string for clients
	if ds.ManualOverrideUntil != nil && ds.ManualOverrideUntil.Equal(indefiniteOverride) {
		resp["manual_override_until"] = "indefinite"
	}

	return resp
}

// resolveManualOverride converts the raw request string pointer to a *time.Time.
// nil in → nil out (clear override)
// "indefinite" → indefiniteOverride sentinel
// RFC3339 string → parsed timestamp
func resolveManualOverride(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	if *raw == "indefinite" {
		t := indefiniteOverride
		return &t, nil
	}
	t, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		return nil, ErrInvalidOverride
	}
	return &t, nil
}
