package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// GetUserIDFromToken extracts user ID from JWT token in the request
// This assumes the JWT has already been validated by middleware
func GetUserIDFromToken(c *gin.Context) (uuid.UUID, error) {
	authHeader := c.GetHeader("Authorization")
	tokenString := authHeader[len("Bearer "):]

	// Parse without verification since middleware already validated it
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return uuid.Nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, err
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return uuid.Nil, err
	}

	return uuid.Parse(userIDStr)
}
