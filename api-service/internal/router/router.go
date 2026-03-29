package router

import (
	"github.com/DiegoJohnson25/climate-control/api-service/internal/auth"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/health"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/room"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/user"
	"github.com/gin-gonic/gin"
)

func Setup(
	authHandler *auth.Handler,
	authMiddleware *auth.Service,
	userHandler *user.Handler,
	roomHandler *room.Handler,
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

	// Rooms
	protected.GET("/rooms", roomHandler.List)
	protected.POST("/rooms", roomHandler.Create)
	protected.GET("/rooms/:id", roomHandler.Get)
	protected.PUT("/rooms/:id", roomHandler.Update)
	protected.DELETE("/rooms/:id", roomHandler.Delete)

	// Desired state
	protected.GET("/rooms/:id/desired-state", roomHandler.GetDesiredState)
	protected.PUT("/rooms/:id/desired-state", roomHandler.UpdateDesiredState)

	return r
}
