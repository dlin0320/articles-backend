package repository

import (
	"fmt"

	ratingPkg "github.com/dustin/articles-backend/internal/rating"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// gormRatingRepository implements the rating.Repository interface with GORM optimizations
type gormRatingRepository struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewGORMRatingRepository creates a new GORM-based rating repository
func NewGORMRatingRepository(db *gorm.DB, log *logger.Logger) ratingPkg.Repository {
	return &gormRatingRepository{
		db:     db,
		logger: log.WithComponent("gorm-rating-repository"),
	}
}

func (r *gormRatingRepository) Create(rating *ratingPkg.Rating) error {
	r.logger.Info("Repository operation")

	if err := r.db.Create(rating).Error; err != nil {
		r.logger.Error("Repository error")
		return fmt.Errorf("failed to create rating: %w", err)
	}

	r.logger.Info("Repository operation")

	return nil
}

func (r *gormRatingRepository) FindByUserAndArticle(userID, articleID uuid.UUID) (*ratingPkg.Rating, error) {
	var rating ratingPkg.Rating

	// Use compound primary key lookup for optimal performance
	err := r.db.Where("user_id = ? AND article_id = ?", userID, articleID).First(&rating).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Info("Repository operation")
			return nil, fmt.Errorf("rating not found")
		}

		r.logger.Error("Repository error")
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &rating, nil
}

func (r *gormRatingRepository) Update(rating *ratingPkg.Rating) error {
	r.logger.Info("Repository operation")

	// Use Save() for updates with GORM optimizations
	if err := r.db.Save(rating).Error; err != nil {
		r.logger.Error("Repository error")
		return fmt.Errorf("failed to update rating: %w", err)
	}

	r.logger.Info("Repository operation")

	return nil
}

func (r *gormRatingRepository) Delete(userID, articleID uuid.UUID) error {
	r.logger.Info("Repository operation")

	// Use compound key delete
	result := r.db.Delete(&ratingPkg.Rating{}, "user_id = ? AND article_id = ?", userID, articleID)
	if err := result.Error; err != nil {
		r.logger.Error("Repository error")
		return fmt.Errorf("failed to delete rating: %w", err)
	}

	if result.RowsAffected == 0 {
		r.logger.Warn("Repository warning")
		return fmt.Errorf("rating not found")
	}

	r.logger.Info("Repository operation")

	return nil
}

func (r *gormRatingRepository) GetAverageRating(articleID uuid.UUID) (float64, int, error) {
	type Result struct {
		Average float64
		Count   int
	}

	var result Result

	// Use efficient aggregation query
	err := r.db.Model(&ratingPkg.Rating{}).
		Select("AVG(score) as average, COUNT(*) as count").
		Where("article_id = ?", articleID).
		Scan(&result).Error

	if err != nil {
		r.logger.Error("Repository error")
		return 0, 0, fmt.Errorf("database error: %w", err)
	}

	r.logger.Info("Repository operation")

	return result.Average, result.Count, nil
}
