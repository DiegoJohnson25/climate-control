package room

import (
	"errors"
	"net/http"
	"time"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/ctxkeys"
	"github.com/DiegoJohnson25/climate-control/shared/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// -------------------------------------------------------------------------------
// Request types
// -------------------------------------------------------------------------------

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
	Mode       string   `json:"mode"            binding:"required,oneof=OFF AUTO"`
	TargetTemp *float64 `json:"target_temp"`
	TargetHum  *float64 `json:"target_hum"`
	// "indefinite", an RFC3339 string, or null (JSON null or key absent = clear override)
	ManualOverride *string `json:"manual_override"`
}

// -------------------------------------------------------------------------------
// Room CRUD
// -------------------------------------------------------------------------------

func (h *Handler) List(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	rms, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	resp := make([]gin.H, len(rms))
	for i, rm := range rms {
		resp[i] = roomResponse(&rm)
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

	c.JSON(http.StatusCreated, roomResponse(rm))
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

	c.JSON(http.StatusOK, roomResponse(rm))
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

	c.JSON(http.StatusOK, roomResponse(rm))
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

// -------------------------------------------------------------------------------
// Desired state
// -------------------------------------------------------------------------------

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

	// Resolve manual_override → *time.Time before passing to service.
	// nil pointer   = key absent or JSON null → clear override (nil stored in DB)
	// "indefinite"  → indefiniteOverride sentinel
	// RFC3339 string → parsed timestamp
	overrideUntil, err := resolveManualOverride(req.ManualOverride)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": ErrInvalidOverride.Error()})
		return
	}

	input := UpdateDesiredStateInput{
		Mode:                req.Mode,
		TargetTemp:          req.TargetTemp,
		TargetHum:           req.TargetHum,
		ManualOverrideUntil: overrideUntil,
	}

	ds, err := h.svc.UpdateDesiredState(c.Request.Context(), roomID, userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
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

// -------------------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------------------

func roomResponse(rm *models.Room) gin.H {
	return gin.H{
		"id":            rm.ID,
		"name":          rm.Name,
		"deadband_temp": rm.DeadbandTemp,
		"deadband_hum":  rm.DeadbandHum,
		"created_at":    rm.CreatedAt,
		"updated_at":    rm.UpdatedAt,
	}
}

func desiredStateResponse(ds models.DesiredState) gin.H {
	resp := gin.H{
		"id":                    ds.ID,
		"room_id":               ds.RoomID,
		"mode":                  ds.Mode,
		"target_temp":           ds.TargetTemp,
		"target_hum":            ds.TargetHum,
		"manual_override_until": ds.ManualOverrideUntil,
		"updated_at":            ds.UpdatedAt,
	}

	// Surface the indefinite sentinel as the string "indefinite" in responses
	// so clients don't need to know the raw timestamp value.
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
