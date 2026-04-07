package schedule

import (
	"errors"
	"net/http"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/ctxkeys"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/room"
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
// Request structs
// -------------------------------------------------------------------------------

type scheduleRequest struct {
	Name string `json:"name" binding:"required"`
}

type periodRequest struct {
	Name       *string  `json:"name"`
	DaysOfWeek []int    `json:"days_of_week" binding:"required,min=1"`
	StartTime  string   `json:"start_time"   binding:"required,datetime=15:04"`
	EndTime    string   `json:"end_time"     binding:"required,datetime=15:04"`
	Mode       string   `json:"mode"         binding:"required,oneof=OFF AUTO"`
	TargetTemp *float64 `json:"target_temp"`
	TargetHum  *float64 `json:"target_hum"`
}

// -------------------------------------------------------------------------------
// Schedule handlers
// -------------------------------------------------------------------------------

func (h *Handler) Create(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	var req scheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sched, err := h.svc.Create(c.Request.Context(), roomID, userID, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, room.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": room.ErrNotFound.Error()})
		case errors.Is(err, ErrNameTaken):
			c.JSON(http.StatusConflict, gin.H{"error": ErrNameTaken.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, scheduleResponse(sched))
}

func (h *Handler) ListByRoom(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	scheds, err := h.svc.ListByRoom(c.Request.Context(), roomID, userID)
	if err != nil {
		if errors.Is(err, room.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": room.ErrNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	resp := make([]gin.H, len(scheds))
	for i, sched := range scheds {
		resp[i] = scheduleResponse(&sched)
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Get(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	sched, err := h.svc.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, scheduleResponse(sched))
}

func (h *Handler) Update(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	var req scheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sched, err := h.svc.Update(c.Request.Context(), id, userID, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
		case errors.Is(err, ErrNameTaken):
			c.JSON(http.StatusConflict, gin.H{"error": ErrNameTaken.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, scheduleResponse(sched))
}

func (h *Handler) Delete(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) Activate(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	sched, err := h.svc.Activate(c.Request.Context(), id, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
		case errors.Is(err, ErrAlreadyActive):
			c.JSON(http.StatusConflict, gin.H{"error": ErrAlreadyActive.Error()})
		case errors.Is(err, ErrCapabilityConflict):
			c.JSON(http.StatusConflict, gin.H{
				"error": ErrCapabilityConflict.Error(),
				"hint":  "ensure the room has devices with the required sensors and actuators for all periods in this schedule",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, scheduleResponse(sched))
}

func (h *Handler) Deactivate(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	sched, err := h.svc.Deactivate(c.Request.Context(), id, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
		case errors.Is(err, ErrAlreadyInactive):
			c.JSON(http.StatusConflict, gin.H{"error": ErrAlreadyInactive.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, scheduleResponse(sched))
}

// -------------------------------------------------------------------------------
// Schedule period handlers
// -------------------------------------------------------------------------------

func (h *Handler) CreatePeriod(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	var req periodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validatePeriodRequest(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	input := toPeriodInput(&req)

	period, err := h.svc.CreatePeriod(c.Request.Context(), scheduleID, userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
		case errors.Is(err, ErrInvalidTimeRange):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": ErrInvalidTimeRange.Error()})
		case errors.Is(err, ErrPeriodOverlap):
			c.JSON(http.StatusConflict, gin.H{"error": ErrPeriodOverlap.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, periodResponse(period))
}

func (h *Handler) ListPeriods(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	periods, err := h.svc.ListPeriodsBySchedule(c.Request.Context(), scheduleID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	resp := make([]gin.H, len(periods))
	for i, period := range periods {
		resp[i] = periodResponse(&period)
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdatePeriod(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	periodID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid period id"})
		return
	}

	var req periodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validatePeriodRequest(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	input := toPeriodInput(&req)

	period, err := h.svc.UpdatePeriod(c.Request.Context(), periodID, userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrPeriodNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": ErrPeriodNotFound.Error()})
		case errors.Is(err, ErrInvalidTimeRange):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": ErrInvalidTimeRange.Error()})
		case errors.Is(err, ErrPeriodOverlap):
			c.JSON(http.StatusConflict, gin.H{"error": ErrPeriodOverlap.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, periodResponse(period))
}

func (h *Handler) DeletePeriod(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	periodID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid period id"})
		return
	}

	if err := h.svc.DeletePeriod(c.Request.Context(), periodID, userID); err != nil {
		if errors.Is(err, ErrPeriodNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": ErrPeriodNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}

// -------------------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------------------

// validatePeriodRequest performs validation that cannot be expressed via binding tags.
func validatePeriodRequest(req *periodRequest) error {
	for _, d := range req.DaysOfWeek {
		if d < 1 || d > 7 {
			return errors.New("days_of_week values must be between 1 (Monday) and 7 (Sunday)")
		}
	}

	if req.Mode == "AUTO" && req.TargetTemp == nil && req.TargetHum == nil {
		return errors.New("AUTO mode requires at least one of target_temp or target_hum")
	}

	return nil
}

// toPeriodInput maps a periodRequest onto a PeriodInput for the service layer.
func toPeriodInput(req *periodRequest) PeriodInput {
	days := make([]int64, len(req.DaysOfWeek))
	for i, d := range req.DaysOfWeek {
		days[i] = int64(d)
	}

	return PeriodInput{
		Name:       req.Name,
		DaysOfWeek: days,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Mode:       req.Mode,
		TargetTemp: req.TargetTemp,
		TargetHum:  req.TargetHum,
	}
}

func scheduleResponse(sched *models.Schedule) gin.H {
	return gin.H{
		"id":         sched.ID,
		"room_id":    sched.RoomID,
		"name":       sched.Name,
		"is_active":  sched.IsActive,
		"created_at": sched.CreatedAt,
		"updated_at": sched.UpdatedAt,
	}
}

func periodResponse(p *models.SchedulePeriod) gin.H {
	return gin.H{
		"id":           p.ID,
		"schedule_id":  p.ScheduleID,
		"name":         p.Name,
		"days_of_week": []int64(p.DaysOfWeek),
		"start_time":   p.StartTime,
		"end_time":     p.EndTime,
		"mode":         p.Mode,
		"target_temp":  p.TargetTemp,
		"target_hum":   p.TargetHum,
		"created_at":   p.CreatedAt,
		"updated_at":   p.UpdatedAt,
	}
}
