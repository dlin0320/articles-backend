package recommendation

import (
	"net/http"
	"strconv"

	"github.com/dustin/articles-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for recommendation operations
type Handler struct {
	service Service
}

// NewHandler creates a new recommendation handler
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// GetRecommendations handles getting recommendations for authenticated user
func (h *Handler) GetRecommendations(c *gin.Context) {
	// Extract user ID from JWT token
	userID, err := utils.GetUserIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 10
	}

	// Get recommendations using default engine
	recommendations, err := h.service.GetRecommendations(userID, limit)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recommendations"})
		return
	}

	response := BuildRecommendationResponse(recommendations, userID, "default")
	c.JSON(http.StatusOK, response)
}

// RegisterRoutes registers all recommendation routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// All recommendation routes require authentication
	recommendations := router.Group("/recommendations")
	recommendations.Use(authMiddleware)
	{
		// Get recommendations
		recommendations.GET("", h.GetRecommendations)
	}
}
