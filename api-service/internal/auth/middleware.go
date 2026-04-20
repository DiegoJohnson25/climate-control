package auth

import (
	"net/http"
	"strings"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/ctxkeys"
	"github.com/gin-gonic/gin"
)

// Middleware returns a Gin handler that validates the bearer access token on
// the request and sets ctxkeys.UserID on the context. Requests without a valid
// token are aborted with 401.
func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed authorization header"})
			return
		}
		tokenString := strings.TrimPrefix(header, "Bearer ")

		userID, err := s.ValidateAccessToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(ctxkeys.UserID, userID)
		c.Next()
	}
}
