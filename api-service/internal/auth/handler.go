// Package auth provides HTTP handlers, service logic, and repository access
// for the auth domain.
package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles auth HTTP requests.
type Handler struct {
	svc            *Service
	refreshTTLSecs int
}

// NewHandler constructs a Handler with the given service and refresh token TTL.
// refreshTTLDays must match the TTL used by the service — it is used to set
// the cookie maxAge on login and refresh.
func NewHandler(svc *Service, refreshTTLDays int) *Handler {
	return &Handler{
		svc:            svc,
		refreshTTLSecs: refreshTTLDays * 24 * 60 * 60,
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	h.setRefreshCookie(c, refreshToken)
	c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
}

func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
		return
	}

	accessToken, newRefreshToken, err := h.svc.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	h.setRefreshCookie(c, newRefreshToken)
	c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
}

func (h *Handler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
		return
	}

	if err := h.svc.Logout(c.Request.Context(), refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	h.clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// ---------------------------------------------------------------------------
// Cookie helpers
// ---------------------------------------------------------------------------

// setRefreshCookie writes the refresh token as an httpOnly cookie.
// Secure is false — this service runs over plain HTTP in development.
// TODO: set Secure to true if TLS is configured.
func (h *Handler) setRefreshCookie(c *gin.Context, token string) {
	c.SetCookie("refresh_token", token, h.refreshTTLSecs, "/", "", false, true)
}

// clearRefreshCookie expires the refresh token cookie immediately.
func (h *Handler) clearRefreshCookie(c *gin.Context) {
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
}
