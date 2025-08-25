package recommendation

import (
	"time"

	"github.com/google/uuid"
)

// Engine interface for recommendation algorithms
type Engine interface {
	Recommend(userID uuid.UUID, limit int) ([]*RecommendedArticle, error)
	Name() string
}

// RecommendedArticle represents a recommended article with scoring
type RecommendedArticle struct {
	Article         *Article `json:"article"`
	Score           float64  `json:"score"`
	Reason          string   `json:"reason"`
	RecommenderUsed string   `json:"recommender_used"`
}

// Repository interfaces for data access
type ArticleRepository interface {
	FindByID(id uuid.UUID) (*Article, error)
	FindAll() ([]*Article, error)
	FindPopular(limit int) ([]*Article, error)
	FindSimilar(embedding []float64, userID uuid.UUID, limit int) ([]*Article, error)
}

type RatingRepository interface {
	FindByUserID(userID uuid.UUID) ([]*Rating, error)
	GetAverageRating(articleID uuid.UUID) (float64, int, error)
}

// Service defines the interface for recommendation business logic
type Service interface {
	GetRecommendations(userID uuid.UUID, limit int) ([]*RecommendedArticle, error)
}

// Forward declarations for GORM relationships
type Article struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID          uuid.UUID `gorm:"type:uuid;not null"`
	URL             string    `gorm:"not null;size:2048"`
	Title           string    `gorm:"size:500"`
	Description     string    `gorm:"type:text"`
	Content         string    `gorm:"type:text"`
	ImageURL        string    `gorm:"size:2048"`
	WordCount       int       `gorm:"default:0"`
	MetadataStatus  string    `gorm:"size:20;default:'pending'"`
	Embedding       []float64 `gorm:"type:vector(384);index" json:"-"` // Store embedding for recommendations
	EmbeddingStatus string    `gorm:"size:20;default:'pending'"`       // Track embedding generation status
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

type Rating struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	ArticleID uuid.UUID `gorm:"type:uuid;primaryKey"`
	Score     int       `gorm:"not null;check:score >= 1 AND score <= 5"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type User struct {
	ID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email string    `gorm:"uniqueIndex;not null;size:255"`
}

// Response DTOs
type RecommendationResponse struct {
	Recommendations []*RecommendedArticle `json:"recommendations"`
	GeneratedAt     time.Time             `json:"generated_at"`
	EngineUsed      string                `json:"engine_used"`
	UserID          uuid.UUID             `json:"user_id"`
	Count           int                   `json:"count"`
}

// ToResponse converts a slice of RecommendedArticle to RecommendationResponse
func BuildRecommendationResponse(recommendations []*RecommendedArticle, userID uuid.UUID, engineUsed string) *RecommendationResponse {
	return &RecommendationResponse{
		Recommendations: recommendations,
		GeneratedAt:     time.Now(),
		EngineUsed:      engineUsed,
		UserID:          userID,
		Count:           len(recommendations),
	}
}
