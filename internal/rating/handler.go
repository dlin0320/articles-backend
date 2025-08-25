package rating

import (
	"net/http"

	"github.com/dustin/articles-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for rating operations
type Handler struct {
	service Service
}

// NewHandler creates a new rating handler
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// RateArticle handles article rating creation/update
func (h *Handler) RateArticle(c *gin.Context) {
	var req RateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract user ID from JWT token
	userID, err := utils.GetUserIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Parse article ID from URL - supports both "articleId" and "id" params
	articleIDParam := c.Param("articleId")
	if articleIDParam == "" {
		articleIDParam = c.Param("id")
	}
	articleID, err := uuid.Parse(articleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	rating, err := h.service.RateArticle(userID, articleID, req.Score)
	if err != nil {
		switch err.Error() {
		case "article not found":
			c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		default:
			if err.Error()[:5] == "score" { // Score validation error
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rate article"})
			}
		}
		return
	}

	c.JSON(http.StatusOK, rating.ToResponse())
}

// GetRating handles getting a specific rating
func (h *Handler) GetRating(c *gin.Context) {
	// Extract user ID from JWT token
	userID, err := utils.GetUserIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Parse article ID from URL - supports both "articleId" and "id" params
	articleIDParam := c.Param("articleId")
	if articleIDParam == "" {
		articleIDParam = c.Param("id")
	}
	articleID, err := uuid.Parse(articleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	rating, err := h.service.GetRating(userID, articleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rating not found"})
		return
	}

	c.JSON(http.StatusOK, rating.ToResponse())
}

// DeleteRating handles rating deletion
func (h *Handler) DeleteRating(c *gin.Context) {
	// Extract user ID from JWT token
	userID, err := utils.GetUserIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Parse article ID from URL - supports both "articleId" and "id" params
	articleIDParam := c.Param("articleId")
	if articleIDParam == "" {
		articleIDParam = c.Param("id")
	}
	articleID, err := uuid.Parse(articleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	err = h.service.DeleteRating(userID, articleID)
	if err != nil {
		if err.Error() == "rating not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Rating not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete rating"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rating deleted successfully"})
}

// RegisterRoutes registers all rating routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// All rating routes require authentication
	ratings := router.Group("/ratings")
	ratings.Use(authMiddleware)
	{
		// Article-specific rating operations
		ratings.POST("/articles/:articleId", h.RateArticle)
		ratings.GET("/articles/:articleId", h.GetRating)
		ratings.DELETE("/articles/:articleId", h.DeleteRating)
	}
}
