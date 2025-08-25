package article

import (
	"time"

	"github.com/google/uuid"
)

// Article represents an article with optimized GORM relationships
type Article struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index:idx_user_articles"`
	URL             string    `json:"url" gorm:"not null;size:2048;uniqueIndex:idx_user_url,composite:user_id"`
	Title           string    `json:"title" gorm:"size:500"`
	Description     string    `json:"description" gorm:"type:text"`
	ImageURL        string    `json:"image_url" gorm:"size:2048"`
	Content         string    `json:"content" gorm:"type:text"`
	WordCount       int       `json:"word_count" gorm:"default:0"`
	MetadataStatus  string    `json:"metadata_status" gorm:"size:20;default:'pending';index"`
	RetryCount      int       `json:"retry_count" gorm:"default:0"`
	ConfidenceScore float64   `json:"confidence_score" gorm:"default:0"`
	ClassifierUsed  string    `json:"classifier_used" gorm:"size:50"`
	Embedding       []float64 `json:"-" gorm:"type:vector(384);index"`                   // Store embedding for recommendations
	EmbeddingStatus string    `json:"embedding_status" gorm:"size:20;default:'pending'"` // Track embedding generation status
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Associations
	User    *User    `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Ratings []Rating `json:"ratings,omitempty" gorm:"foreignKey:ArticleID;constraint:OnDelete:CASCADE"`
}

// User represents user for foreign key relationship (forward declaration)
type User struct {
	ID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email string
}

// Rating represents rating for foreign key relationship (forward declaration)
type Rating struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	ArticleID uuid.UUID `gorm:"type:uuid;primaryKey"`
	Score     int
	CreatedAt time.Time
}

// Metadata status constants
const (
	MetadataStatusPending = "pending"
	MetadataStatusSuccess = "success"
	MetadataStatusFailed  = "failed"
)

// Embedding status constants
const (
	EmbeddingStatusPending = "pending"
	EmbeddingStatusSuccess = "success"
	EmbeddingStatusFailed  = "failed"
)

// Repository defines the interface for article data access
type Repository interface {
	Create(article *Article) error
	FindByID(id uuid.UUID) (*Article, error)
	FindByUserID(userID uuid.UUID, offset, limit int) ([]*Article, error)
	FindByUserIDWithRatings(userID uuid.UUID, offset, limit int) ([]*Article, error)
	Update(article *Article) error
	Delete(id uuid.UUID) error

	// Metadata-specific queries
	FindFailedMetadata(maxRetries int) ([]*Article, error)
	FindFailedWithRetryCount(retryCount int, olderThan time.Time, limit int) ([]*Article, error)
}

// Service defines the interface for article business logic
type Service interface {
	CreateArticle(userID uuid.UUID, url string) (*Article, error)
	GetArticle(id uuid.UUID, userID uuid.UUID) (*Article, error)
	GetUserArticles(userID uuid.UUID, page, limit int) ([]*Article, int64, error)
	DeleteArticle(id uuid.UUID, userID uuid.UUID) error
	UpdateMetadata(id uuid.UUID, title, description, content string, wordCount int, confidence float64) error

	// Background processing
	RetryFailedMetadata() error
	ExtractMetadata(articleID uuid.UUID) error
}

// MetadataExtractor interface for content extraction
type MetadataExtractor interface {
	Extract(url string) (*ExtractedMetadata, error)
}

// ExtractedMetadata represents extracted article metadata
type ExtractedMetadata struct {
	Title       string
	Description string
	Content     string
	ImageURL    string
	WordCount   int
	Confidence  float64
}

// CreateArticleRequest represents article creation request
type CreateArticleRequest struct {
	URL string `json:"url" binding:"required,url"`
}

// ArticleResponse represents article in API responses
type ArticleResponse struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	URL             string    `json:"url"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	ImageURL        string    `json:"image_url"`
	WordCount       int       `json:"word_count"`
	MetadataStatus  string    `json:"metadata_status"`
	ConfidenceScore float64   `json:"confidence_score"`
	ClassifierUsed  string    `json:"classifier_used"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Optional associations
	AverageRating *float64 `json:"average_rating,omitempty"`
	RatingCount   *int     `json:"rating_count,omitempty"`
}

// ArticleListResponse represents paginated article list
type ArticleListResponse struct {
	Articles []*ArticleResponse `json:"articles"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	Limit    int                `json:"limit"`
	Pages    int                `json:"pages"`
}

// ToResponse converts Article to ArticleResponse
func (a *Article) ToResponse() *ArticleResponse {
	response := &ArticleResponse{
		ID:              a.ID,
		UserID:          a.UserID,
		URL:             a.URL,
		Title:           a.Title,
		Description:     a.Description,
		ImageURL:        a.ImageURL,
		WordCount:       a.WordCount,
		MetadataStatus:  a.MetadataStatus,
		ConfidenceScore: a.ConfidenceScore,
		ClassifierUsed:  a.ClassifierUsed,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}

	// Calculate average rating if ratings are loaded
	if len(a.Ratings) > 0 {
		total := 0
		for _, rating := range a.Ratings {
			total += rating.Score
		}
		avg := float64(total) / float64(len(a.Ratings))
		response.AverageRating = &avg
		count := len(a.Ratings)
		response.RatingCount = &count
	}

	return response
}

// IsOwnedBy checks if the article belongs to the specified user
func (a *Article) IsOwnedBy(userID uuid.UUID) bool {
	return a.UserID == userID
}

// NeedsMetadataExtraction checks if the article needs metadata extraction
func (a *Article) NeedsMetadataExtraction() bool {
	return a.MetadataStatus == MetadataStatusPending ||
		(a.MetadataStatus == MetadataStatusFailed && a.RetryCount < 3)
}

// TableName returns the table name for GORM
func (Article) TableName() string {
	return "articles"
}
