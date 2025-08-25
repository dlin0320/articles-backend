package rating

import (
	"time"

	"github.com/google/uuid"
)

// Rating represents a user's rating of an article with optimized GORM relationships
type Rating struct {
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;primaryKey;not null;index:idx_user_ratings"`
	ArticleID uuid.UUID `json:"article_id" gorm:"type:uuid;primaryKey;not null;index:idx_article_ratings"`
	Score     int       `json:"score" gorm:"not null;check:score >= 1 AND score <= 5"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Associations (forward declarations)
	User    *User    `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Article *Article `json:"article,omitempty" gorm:"foreignKey:ArticleID;constraint:OnDelete:CASCADE"`
}

// User represents user for foreign key relationship (forward declaration)
type User struct {
	ID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email string
}

// Article represents article for foreign key relationship (forward declaration)
type Article struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;not null"`
	Title  string
	URL    string
}

// Repository defines the interface for rating data access
type Repository interface {
	Create(rating *Rating) error
	FindByUserAndArticle(userID, articleID uuid.UUID) (*Rating, error)
	Update(rating *Rating) error
	Delete(userID, articleID uuid.UUID) error

	// Analytics method for recommendations
	GetAverageRating(articleID uuid.UUID) (float64, int, error)
}

// Service defines the interface for rating business logic
type Service interface {
	RateArticle(userID, articleID uuid.UUID, score int) (*Rating, error)
	GetRating(userID, articleID uuid.UUID) (*Rating, error)
	DeleteRating(userID, articleID uuid.UUID) error
}

// ArticleService interface for article validation
type ArticleService interface {
	GetArticle(id uuid.UUID, userID uuid.UUID) (*Article, error)
}

// RateArticleRequest represents rating creation/update request
type RateArticleRequest struct {
	Score int `json:"score" binding:"required,min=1,max=5"`
}

// RatingResponse represents rating in API responses
type RatingResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	ArticleID uuid.UUID `json:"article_id"`
	Score     int       `json:"score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts Rating to RatingResponse
func (r *Rating) ToResponse() *RatingResponse {
	return &RatingResponse{
		UserID:    r.UserID,
		ArticleID: r.ArticleID,
		Score:     r.Score,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// IsValidScore checks if the score is within valid range
func (r *Rating) IsValidScore() bool {
	return r.Score >= 1 && r.Score <= 5
}

// TableName returns the table name for GORM
func (Rating) TableName() string {
	return "ratings"
}
