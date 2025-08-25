package article

import (
	"net/http"
	"strconv"

	"github.com/dustin/articles-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for article operations
type Handler struct {
	service Service
}

// NewHandler creates a new article handler
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateArticle handles article creation
func (h *Handler) CreateArticle(c *gin.Context) {
	var req CreateArticleRequest
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

	article, err := h.service.CreateArticle(userID, req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create article"})
		return
	}

	c.JSON(http.StatusCreated, article.ToResponse())
}

// GetArticles handles getting user's articles with pagination
func (h *Handler) GetArticles(c *gin.Context) {
	// Extract user ID from JWT token
	userID, err := utils.GetUserIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Parse pagination parameters
	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	articles, total, err := h.service.GetUserArticles(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch articles"})
		return
	}

	response := BuildPaginationResponse(articles, total, page, limit)
	c.JSON(http.StatusOK, response)
}

// DeleteArticle handles article deletion
func (h *Handler) DeleteArticle(c *gin.Context) {
	// Parse article ID from URL
	idParam := c.Param("id")
	articleID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	// Extract user ID from JWT token
	userID, err := utils.GetUserIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	err = h.service.DeleteArticle(articleID, userID)
	if err != nil {
		if err.Error() == "article not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete article"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article deleted successfully"})
}

// RegisterRoutes registers all article routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// All article routes require authentication
	articles := router.Group("/articles")
	articles.Use(authMiddleware)
	{
		articles.POST("", h.CreateArticle)
		articles.GET("", h.GetArticles)
		articles.DELETE("/:id", h.DeleteArticle)
	}
}
