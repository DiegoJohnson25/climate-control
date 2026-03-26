package router

import (
	"github.com/DiegoJohnson25/climate-control/api-service/internal/auth"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/health"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/user"
	"github.com/gin-gonic/gin"
)

func Setup(
	authHandler *auth.Handler,
	authMiddleware *auth.Service,
	userHandler *user.Handler,
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

	return r
}
