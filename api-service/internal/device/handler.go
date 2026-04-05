package device

import (
	"errors"
	"net/http"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/ctxkeys"
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

type createDeviceRequest struct {
	Name          string   `json:"name"           binding:"required"`
	HwID          string   `json:"hw_id"          binding:"required"`
	DeviceType    string   `json:"device_type"    binding:"omitempty,oneof=physical simulator"`
	SensorTypes   []string `json:"sensors"        binding:"omitempty"`
	ActuatorTypes []string `json:"actuators"      binding:"omitempty"`
}

type updateDeviceRequest struct {
	Name   string     `json:"name"    binding:"required"`
	RoomID *uuid.UUID `json:"room_id"`
}

// -------------------------------------------------------------------------------
// Handlers
// -------------------------------------------------------------------------------

func (h *Handler) List(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	devs, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal sever error"})
		return
	}

	resp := make([]gin.H, len(devs))
	for i, dev := range devs {
		resp[i] = deviceResponse(&dev)
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListByRoom(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	devs, err := h.svc.ListByRoom(c.Request.Context(), roomID, userID)
	if err != nil {
		if errors.Is(err, ErrRoomNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": ErrRoomNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal sever error"})
		return
	}

	resp := make([]gin.H, len(devs))
	for i, dev := range devs {
		resp[i] = deviceResponse(&dev)
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Get(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device id"})
		return
	}

	dev, err := h.svc.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal sever error"})
		return
	}

	c.JSON(http.StatusOK, deviceResponse(dev))
}

func (h *Handler) Create(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	var req createDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default device_type to physical if not provided.
	if req.DeviceType == "" {
		req.DeviceType = "physical"
	}

	input := CreateInput{
		Name:          req.Name,
		HwID:          req.HwID,
		DeviceType:    req.DeviceType,
		SensorTypes:   req.SensorTypes,
		ActuatorTypes: req.ActuatorTypes,
	}

	dev, err := h.svc.Create(c.Request.Context(), userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrAlreadyOwned):
			c.JSON(http.StatusConflict, gin.H{"error": ErrAlreadyOwned.Error()})
		case errors.Is(err, ErrHwIDTaken):
			c.JSON(http.StatusConflict, gin.H{"error": ErrHwIDTaken.Error()})
		case errors.Is(err, ErrNameTaken):
			c.JSON(http.StatusConflict, gin.H{"error": ErrNameTaken.Error()})
		case errors.Is(err, ErrInvalidSensor):
			c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidSensor.Error()})
		case errors.Is(err, ErrInvalidActuator):
			c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidActuator.Error()})
		case errors.Is(err, ErrDuplicateSensor):
			c.JSON(http.StatusBadRequest, gin.H{"error": ErrDuplicateSensor.Error()})
		case errors.Is(err, ErrDuplicateActuator):
			c.JSON(http.StatusBadRequest, gin.H{"error": ErrDuplicateActuator.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, deviceResponse(dev))
}

func (h *Handler) Update(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device id"})
		return
	}

	var req updateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input := UpdateInput{
		Name:   req.Name,
		RoomID: req.RoomID,
	}

	dev, err := h.svc.Update(c.Request.Context(), id, userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
		case errors.Is(err, ErrRoomNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": ErrRoomNotFound.Error()})
		case errors.Is(err, ErrNameTaken):
			c.JSON(http.StatusConflict, gin.H{"error": ErrNameTaken.Error()})
		case errors.Is(err, ErrCapabilityConflict):
			c.JSON(http.StatusConflict, gin.H{
				"error": ErrCapabilityConflict.Error(),
				"hint":  "turn off manual override or deactivate/edit the active schedule before removing this device",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, deviceResponse(dev))
}

func (h *Handler) Delete(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device id"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, userID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
		case errors.Is(err, ErrCapabilityConflict):
			c.JSON(http.StatusConflict, gin.H{
				"error": ErrCapabilityConflict.Error(),
				"hint":  "turn off manual override or deactivate/edit the active schedule before removing this device",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// -------------------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------------------

func deviceResponse(dev *DeviceWithCapabilities) gin.H {
	sensorTypes := make([]string, len(dev.Sensors))
	for i, s := range dev.Sensors {
		sensorTypes[i] = s.MeasurementType
	}

	actuatorTypes := make([]string, len(dev.Actuators))
	for i, a := range dev.Actuators {
		actuatorTypes[i] = a.ActuatorType
	}

	return gin.H{
		"id":          dev.ID,
		"room_id":     dev.RoomID,
		"name":        dev.Name,
		"hw_id":       dev.HwID,
		"device_type": dev.DeviceType,
		"sensors":     sensorTypes,
		"actuators":   actuatorTypes,
		"created_at":  dev.CreatedAt,
		"updated_at":  dev.UpdatedAt,
	}
}
