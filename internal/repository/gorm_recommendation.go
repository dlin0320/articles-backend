package repository

import (
	"fmt"
	"strings"

	recommendationPkg "github.com/dustin/articles-backend/internal/recommendation"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// gormRecommendationArticleRepository implements the recommendation.ArticleRepository interface
type gormRecommendationArticleRepository struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewGORMRecommendationArticleRepository creates a new GORM-based recommendation article repository
func NewGORMRecommendationArticleRepository(db *gorm.DB, log *logger.Logger) recommendationPkg.ArticleRepository {
	return &gormRecommendationArticleRepository{
		db:     db,
		logger: log.WithComponent("gorm-recommendation-article-repository"),
	}
}

func (r *gormRecommendationArticleRepository) FindByID(id uuid.UUID) (*recommendationPkg.Article, error) {
	var article recommendationPkg.Article

	// Use primary key lookup for optimal performance
	err := r.db.First(&article, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Info("Repository operation")
			return nil, fmt.Errorf("article not found")
		}

		r.logger.Error("Repository error")
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &article, nil
}

func (r *gormRecommendationArticleRepository) FindAll() ([]*recommendationPkg.Article, error) {
	var articles []*recommendationPkg.Article

	// Only return successfully processed articles for recommendations
	err := r.db.Where("metadata_status = ?", "success").
		Order("created_at DESC").
		Find(&articles).Error

	if err != nil {
		r.logger.Error("Repository error")
		return nil, fmt.Errorf("database error: %w", err)
	}

	r.logger.Info("Repository operation")

	return articles, nil
}

func (r *gormRecommendationArticleRepository) FindPopular(limit int) ([]*recommendationPkg.Article, error) {
	var articles []*recommendationPkg.Article

	// Use subquery to find popular articles based on rating count and average
	err := r.db.Raw(`
		SELECT a.* FROM articles a
		LEFT JOIN (
			SELECT article_id, COUNT(*) as rating_count, AVG(score) as avg_rating
			FROM ratings 
			GROUP BY article_id
			HAVING COUNT(*) >= 2
		) r ON a.id = r.article_id
		WHERE a.metadata_status = ?
		ORDER BY 
			CASE WHEN r.rating_count IS NULL THEN 0 ELSE r.rating_count END DESC,
			CASE WHEN r.avg_rating IS NULL THEN 0 ELSE r.avg_rating END DESC,
			a.created_at DESC
		LIMIT ?
	`, "success", limit).Scan(&articles).Error

	if err != nil {
		r.logger.Error("Repository error")
		return nil, fmt.Errorf("database error: %w", err)
	}

	r.logger.Info("Repository operation")

	return articles, nil
}

func (r *gormRecommendationArticleRepository) FindSimilar(embedding []float64, userID uuid.UUID, limit int) ([]*recommendationPkg.Article, error) {
	var articles []*recommendationPkg.Article

	// Convert embedding to PostgreSQL vector format
	embeddingStr := r.formatEmbeddingForPostgres(embedding)

	// Use GORM's structured query builder with pgvector operations
	// The <-> operator calculates cosine distance (0 = identical, 2 = opposite)
	err := r.db.
		Where("user_id != ?", userID).
		Where("embedding IS NOT NULL").
		Where("metadata_status = ?", "success").
		Where("embedding_status = ?", "success").
		Order(r.db.Raw("embedding <-> ?::vector", embeddingStr)).
		Limit(limit).
		Find(&articles).Error

	if err != nil {
		r.logger.Error("Repository error in FindSimilar: " + err.Error())
		return nil, fmt.Errorf("vector similarity search error: %w", err)
	}

	r.logger.Info("Repository operation: found similar articles")

	return articles, nil
}

// formatEmbeddingForPostgres converts a float64 slice to PostgreSQL vector format
func (r *gormRecommendationArticleRepository) formatEmbeddingForPostgres(embedding []float64) string {
	if len(embedding) == 0 {
		return "[]"
	}

	result := make([]string, len(embedding))
	for i, v := range embedding {
		result[i] = fmt.Sprintf("%f", v)
	}
	return "[" + strings.Join(result, ",") + "]"
}

// gormRecommendationRatingRepository implements the recommendation.RatingRepository interface
type gormRecommendationRatingRepository struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewGORMRecommendationRatingRepository creates a new GORM-based recommendation rating repository
func NewGORMRecommendationRatingRepository(db *gorm.DB, log *logger.Logger) recommendationPkg.RatingRepository {
	return &gormRecommendationRatingRepository{
		db:     db,
		logger: log.WithComponent("gorm-recommendation-rating-repository"),
	}
}

func (r *gormRecommendationRatingRepository) FindByUserID(userID uuid.UUID) ([]*recommendationPkg.Rating, error) {
	var ratings []*recommendationPkg.Rating

	// Use index-optimized query
	err := r.db.Where("user_id = ?", userID).Find(&ratings).Error
	if err != nil {
		r.logger.Error("Repository error")
		return nil, fmt.Errorf("database error: %w", err)
	}

	r.logger.Info("Repository operation")

	return ratings, nil
}

func (r *gormRecommendationRatingRepository) GetAverageRating(articleID uuid.UUID) (float64, int, error) {
	type Result struct {
		Average float64
		Count   int
	}

	var result Result

	// Use efficient aggregation query
	err := r.db.Model(&recommendationPkg.Rating{}).
		Select("AVG(score) as average, COUNT(*) as count").
		Where("article_id = ?", articleID).
		Scan(&result).Error

	if err != nil {
		r.logger.Error("Repository error")
		return 0, 0, fmt.Errorf("database error: %w", err)
	}

	return result.Average, result.Count, nil
}
