// Package health exposes the liveness probe endpoint.
package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Check responds with 200 OK if the service is running.
func Check(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
