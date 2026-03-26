package user

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

func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=4"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	usr, err := h.svc.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": "email already taken"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         usr.ID,
		"email":      usr.Email,
		"timezone":   usr.Timezone,
		"created_at": usr.CreatedAt,
	})
}

func (h *Handler) Me(c *gin.Context) {
	userID := c.MustGet(ctxkeys.UserID).(uuid.UUID)

	usr, err := h.svc.GetByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         usr.ID,
		"email":      usr.Email,
		"timezone":   usr.Timezone,
		"created_at": usr.CreatedAt,
	})
}
