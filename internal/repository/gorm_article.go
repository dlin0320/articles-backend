package repository

import (
	"fmt"
	"time"

	articlePkg "github.com/dustin/articles-backend/internal/article"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// gormArticleRepository implements the article.Repository interface with GORM optimizations
type gormArticleRepository struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewGORMArticleRepository creates a new GORM-based article repository
func NewGORMArticleRepository(db *gorm.DB, log *logger.Logger) articlePkg.Repository {
	return &gormArticleRepository{
		db:     db,
		logger: log.WithComponent("gorm-article-repository"),
	}
}

func (r *gormArticleRepository) Create(article *articlePkg.Article) error {
	r.logger.Info("Creating article " + article.ID.String() + " for user " + article.UserID.String() + " URL " + article.URL)

	if err := r.db.Create(article).Error; err != nil {
		r.logger.Error("Failed to create article " + article.ID.String() + " for user " + article.UserID.String() + " URL " + article.URL + ": " + err.Error())
		return fmt.Errorf("failed to create article: %w", err)
	}

	r.logger.Info("Article created successfully: " + article.ID.String() + " for user " + article.UserID.String() + " URL " + article.URL)

	return nil
}

func (r *gormArticleRepository) FindByID(id uuid.UUID) (*articlePkg.Article, error) {
	var article articlePkg.Article

	// Use primary key lookup for optimal performance
	err := r.db.First(&article, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Info("Article not found: " + id.String())
			return nil, fmt.Errorf("article not found")
		}

		r.logger.Error("Database error finding article " + id.String() + ": " + err.Error())
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &article, nil
}

func (r *gormArticleRepository) FindByUserID(userID uuid.UUID, offset, limit int) ([]*articlePkg.Article, error) {
	var articles []*articlePkg.Article

	// Use index-optimized query with proper ordering
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&articles).Error

	if err != nil {
		r.logger.Error("Database error finding articles by user " + userID.String() + " (offset " + fmt.Sprintf("%d", offset) + ", limit " + fmt.Sprintf("%d", limit) + "): " + err.Error())
		return nil, fmt.Errorf("database error: %w", err)
	}

	r.logger.Info("Found " + fmt.Sprintf("%d", len(articles)) + " articles by user " + userID.String() + " (offset " + fmt.Sprintf("%d", offset) + ", limit " + fmt.Sprintf("%d", limit) + ")")

	return articles, nil
}

func (r *gormArticleRepository) FindByUserIDWithRatings(userID uuid.UUID, offset, limit int) ([]*articlePkg.Article, error) {
	var articles []*articlePkg.Article

	// Use Preload for efficient rating loading
	err := r.db.Preload("Ratings").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&articles).Error

	if err != nil {
		r.logger.Error("Database error finding articles with ratings by user " + userID.String() + " (offset " + fmt.Sprintf("%d", offset) + ", limit " + fmt.Sprintf("%d", limit) + "): " + err.Error())
		return nil, fmt.Errorf("database error: %w", err)
	}

	r.logger.Info("Found " + fmt.Sprintf("%d", len(articles)) + " articles with ratings by user " + userID.String() + " (offset " + fmt.Sprintf("%d", offset) + ", limit " + fmt.Sprintf("%d", limit) + ")")

	return articles, nil
}

func (r *gormArticleRepository) Update(article *articlePkg.Article) error {
	r.logger.Info("Updating article " + article.ID.String() + " for user " + article.UserID.String())

	// Use Save() for updates with GORM optimizations
	if err := r.db.Save(article).Error; err != nil {
		r.logger.Error("Failed to update article " + article.ID.String() + " for user " + article.UserID.String() + ": " + err.Error())
		return fmt.Errorf("failed to update article: %w", err)
	}

	r.logger.Info("Article updated successfully: " + article.ID.String() + " for user " + article.UserID.String())

	return nil
}

func (r *gormArticleRepository) Delete(id uuid.UUID) error {
	r.logger.Info("Deleting article: " + id.String())

	// Soft delete with GORM
	result := r.db.Delete(&articlePkg.Article{}, id)
	if err := result.Error; err != nil {
		r.logger.Error("Failed to delete article " + id.String() + ": " + err.Error())
		return fmt.Errorf("failed to delete article: %w", err)
	}

	if result.RowsAffected == 0 {
		r.logger.Warn("No article found to delete: " + id.String())
		return fmt.Errorf("article not found")
	}

	r.logger.Info("Article deleted successfully: " + id.String())

	return nil
}

func (r *gormArticleRepository) FindFailedMetadata(maxRetries int) ([]*articlePkg.Article, error) {
	var articles []*articlePkg.Article

	// Use index-optimized query for metadata status and retry count
	err := r.db.Where("metadata_status = ? AND retry_count < ?",
		articlePkg.MetadataStatusFailed, maxRetries).
		Order("updated_at ASC"). // Process oldest failures first
		Find(&articles).Error

	if err != nil {
		r.logger.Error("Database error finding failed metadata articles (max retries " + fmt.Sprintf("%d", maxRetries) + "): " + err.Error())
		return nil, fmt.Errorf("database error: %w", err)
	}

	r.logger.Info("Found " + fmt.Sprintf("%d", len(articles)) + " failed metadata articles (max retries " + fmt.Sprintf("%d", maxRetries) + ")")

	return articles, nil
}

func (r *gormArticleRepository) FindFailedWithRetryCount(retryCount int, olderThan time.Time, limit int) ([]*articlePkg.Article, error) {
	var articles []*articlePkg.Article

	// Use compound index query for efficient filtering
	err := r.db.Where("metadata_status = ? AND retry_count = ? AND updated_at < ?",
		articlePkg.MetadataStatusFailed, retryCount, olderThan).
		Order("updated_at ASC").
		Limit(limit).
		Find(&articles).Error

	if err != nil {
		r.logger.Error("Database error finding failed articles with retry count " + fmt.Sprintf("%d", retryCount) + " older than " + olderThan.Format("2006-01-02") + " limit " + fmt.Sprintf("%d", limit) + ": " + err.Error())
		return nil, fmt.Errorf("database error: %w", err)
	}

	r.logger.Info("Found " + fmt.Sprintf("%d", len(articles)) + " failed articles with retry count " + fmt.Sprintf("%d", retryCount) + " older than " + olderThan.Format("2006-01-02") + " limit " + fmt.Sprintf("%d", limit))

	return articles, nil
}
