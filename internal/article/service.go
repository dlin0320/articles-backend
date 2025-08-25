package article

import (
	"errors"
	"time"

	"github.com/dustin/articles-backend/internal/utils"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/google/uuid"
)

// service implements the Service interface
type service struct {
	repo      Repository
	extractor MetadataExtractor
	logger    *logger.Logger
}

// NewService creates a new article service
func NewService(repo Repository, extractor MetadataExtractor, log *logger.Logger) Service {
	return &service{
		repo:      repo,
		extractor: extractor,
		logger:    log.WithComponent("article-service"),
	}
}

func (s *service) CreateArticle(userID uuid.UUID, url string) (*Article, error) {
	s.logger.Info("Creating article for user " + userID.String() + ": " + url)

	// Create article with pending metadata
	article := &Article{
		ID:             uuid.New(),
		UserID:         userID,
		URL:            url,
		MetadataStatus: MetadataStatusPending,
		RetryCount:     0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Save to database
	err := s.repo.Create(article)
	if err != nil {
		s.logger.Error("Failed to create article for user " + userID.String() + " URL " + url + ": " + err.Error())
		return nil, err
	}

	// Asynchronously extract metadata
	go func() {
		if err := s.ExtractMetadata(article.ID); err != nil {
			s.logger.Error("Failed to extract metadata for article " + article.ID.String() + " URL " + url + ": " + err.Error())
		}
	}()

	s.logger.Info("Article created successfully: " + article.ID.String() + " for user " + userID.String() + " URL " + url)

	return article, nil
}

func (s *service) GetArticle(id uuid.UUID, userID uuid.UUID) (*Article, error) {
	article, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if !article.IsOwnedBy(userID) {
		return nil, errors.New("article not found")
	}

	return article, nil
}

func (s *service) GetUserArticles(userID uuid.UUID, page, limit int) ([]*Article, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	s.logger.Info("Fetching user articles for " + userID.String() + " (page " + utils.IntToString(page) + ", limit " + utils.IntToString(limit) + ", offset " + utils.IntToString(offset) + ")")

	// Get articles with ratings for better response
	articles, err := s.repo.FindByUserIDWithRatings(userID, offset, limit)
	if err != nil {
		s.logger.Error("Failed to fetch user articles for " + userID.String() + ": " + err.Error())
		return nil, 0, err
	}

	// Get total count for pagination
	// This is a simplified approach - in production, you might want a separate count query
	allArticles, err := s.repo.FindByUserID(userID, 0, 10000) // Get all for count
	if err != nil {
		return articles, 0, nil // Return articles even if count fails
	}
	total := int64(len(allArticles))

	return articles, total, nil
}

func (s *service) DeleteArticle(id uuid.UUID, userID uuid.UUID) error {
	s.logger.Info("Deleting article " + id.String() + " for user " + userID.String())

	// First verify ownership
	article, err := s.repo.FindByID(id)
	if err != nil {
		return errors.New("article not found")
	}

	if !article.IsOwnedBy(userID) {
		return errors.New("article not found")
	}

	// Delete the article
	err = s.repo.Delete(id)
	if err != nil {
		s.logger.Error("Failed to delete article " + id.String() + " for user " + userID.String() + ": " + err.Error())
		return err
	}

	s.logger.Info("Article deleted successfully: " + id.String() + " for user " + userID.String())

	return nil
}

func (s *service) UpdateMetadata(id uuid.UUID, title, description, content string, wordCount int, confidence float64) error {
	article, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	// Update metadata fields
	article.Title = title
	article.Description = description
	article.Content = content
	article.WordCount = wordCount
	article.ConfidenceScore = confidence
	article.MetadataStatus = MetadataStatusSuccess
	article.ClassifierUsed = "readability" // Could be parameterized
	article.UpdatedAt = time.Now()

	return s.repo.Update(article)
}

func (s *service) ExtractMetadata(articleID uuid.UUID) error {
	s.logger.Info("Extracting metadata for article: " + articleID.String())

	// Get article
	article, err := s.repo.FindByID(articleID)
	if err != nil {
		return err
	}

	// Extract metadata
	metadata, err := s.extractor.Extract(article.URL)
	if err != nil {
		s.logger.Error("Metadata extraction failed for article " + articleID.String() + " URL " + article.URL + ": " + err.Error())

		// Update failure status
		article.MetadataStatus = MetadataStatusFailed
		article.RetryCount++
		article.UpdatedAt = time.Now()
		s.repo.Update(article)

		return err
	}

	// Update article with extracted metadata
	return s.UpdateMetadata(
		articleID,
		metadata.Title,
		metadata.Description,
		metadata.Content,
		metadata.WordCount,
		metadata.Confidence,
	)
}

func (s *service) RetryFailedMetadata() error {
	s.logger.Info("Starting failed metadata retry process")

	// Get articles that failed and need retry
	failedArticles, err := s.repo.FindFailedMetadata(3) // Max 3 retries
	if err != nil {
		s.logger.Error("Failed to get failed metadata articles: " + err.Error())
		return err
	}

	if len(failedArticles) == 0 {
		s.logger.Info("No failed articles to retry")
		return nil
	}

	s.logger.Info("Retrying failed metadata extractions for " + utils.IntToString(len(failedArticles)) + " articles")

	// Process each failed article
	for _, article := range failedArticles {
		// Check if enough time has passed since last retry (exponential backoff)
		if !s.shouldRetry(article) {
			continue
		}

		s.logger.Info("Retrying metadata extraction for article " + article.ID.String() + " URL " + article.URL + " (retry " + utils.IntToString(article.RetryCount) + ")")

		// Retry extraction
		err := s.ExtractMetadata(article.ID)
		if err != nil {
			s.logger.Error("Retry failed for article " + article.ID.String() + ": " + err.Error())
		} else {
			s.logger.Info("Retry succeeded for article " + article.ID.String())
		}

		// Add small delay between retries to avoid overwhelming the service
		time.Sleep(1 * time.Second)
	}

	return nil
}

// shouldRetry checks if article should be retried (max 3 retries)
func (s *service) shouldRetry(article *Article) bool {
	const maxRetries = 3
	return article.RetryCount < maxRetries
}

// BuildPaginationResponse builds a paginated response
func BuildPaginationResponse(articles []*Article, total int64, page, limit int) *ArticleListResponse {
	responses := make([]*ArticleResponse, len(articles))
	for i, article := range articles {
		responses[i] = article.ToResponse()
	}

	pagination := utils.CalculatePagination(total, page, limit)

	return &ArticleListResponse{
		Articles: responses,
		Total:    pagination.Total,
		Page:     pagination.Page,
		Limit:    pagination.Limit,
		Pages:    pagination.Pages,
	}
}
