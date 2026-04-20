// Package router wires api-service domain handlers onto a Gin engine and
// applies the auth middleware chain.
package router

import (
	"github.com/DiegoJohnson25/climate-control/api-service/internal/auth"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/device"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/health"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/room"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/schedule"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/user"
	"github.com/gin-gonic/gin"
)

// Setup constructs a Gin engine with all api-service routes registered.
// Protected routes require a valid access token via authMiddleware.
func Setup(
	authHandler *auth.Handler,
	authMiddleware *auth.Service,
	userHandler *user.Handler,
	roomHandler *room.Handler,
	deviceHandler *device.Handler,
	scheduleHandler *schedule.Handler,
) *gin.Engine {
	r := gin.Default()

	r.GET("/health", health.Check)

	api := r.Group("/api/v1")

	api.POST("/auth/register", userHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)

	protected := api.Group("/")
	protected.Use(authMiddleware.Middleware())

	protected.POST("/auth/logout", authHandler.Logout)
	protected.GET("/users/me", userHandler.Me)
	protected.DELETE("/users/me", userHandler.DeleteMe)

	// Rooms
	protected.GET("/rooms", roomHandler.List)
	protected.POST("/rooms", roomHandler.Create)
	protected.GET("/rooms/:id", roomHandler.Get)
	protected.PUT("/rooms/:id", roomHandler.Update)
	protected.DELETE("/rooms/:id", roomHandler.Delete)

	// Desired state
	protected.GET("/rooms/:id/desired-state", roomHandler.GetDesiredState)
	protected.PUT("/rooms/:id/desired-state", roomHandler.UpdateDesiredState)

	// Devices
	protected.GET("/devices", deviceHandler.List)
	protected.POST("/devices", deviceHandler.Create)
	protected.GET("/devices/:id", deviceHandler.Get)
	protected.PUT("/devices/:id", deviceHandler.Update)
	protected.DELETE("/devices/:id", deviceHandler.Delete)

	// Devices by room
	protected.GET("/rooms/:id/devices", deviceHandler.ListByRoom)

	// Schedules
	protected.GET("/rooms/:id/schedules", scheduleHandler.ListByRoom)
	protected.POST("/rooms/:id/schedules", scheduleHandler.Create)
	protected.GET("/schedules/:id", scheduleHandler.Get)
	protected.PUT("/schedules/:id", scheduleHandler.Update)
	protected.DELETE("/schedules/:id", scheduleHandler.Delete)
	protected.PATCH("/schedules/:id/activate", scheduleHandler.Activate)
	protected.PATCH("/schedules/:id/deactivate", scheduleHandler.Deactivate)

	// Schedule periods
	protected.POST("/schedules/:id/periods", scheduleHandler.CreatePeriod)
	protected.GET("/schedules/:id/periods", scheduleHandler.ListPeriods)
	protected.PUT("/schedule-periods/:id", scheduleHandler.UpdatePeriod)
	protected.DELETE("/schedule-periods/:id", scheduleHandler.DeletePeriod)

	return r
}
