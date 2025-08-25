package recommendation

import (
	"fmt"

	"github.com/dustin/articles-backend/internal/embedding"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/google/uuid"
)

// service implements the Service interface
type service struct {
	defaultEngine Engine
	engines       map[string]Engine
	logger        *logger.Logger
}

// NewService creates a new recommendation service
func NewService(articleRepo ArticleRepository, ratingRepo RatingRepository, embeddingClient embedding.EmbeddingClient, log *logger.Logger) Service {
	// Create content-based recommendation engine
	contentEngine := NewContentBasedEngine(articleRepo, ratingRepo, embeddingClient, log)

	return &service{
		defaultEngine: contentEngine,
		engines: map[string]Engine{
			"content": contentEngine,
		},
		logger: log.WithComponent("recommendation-service"),
	}
}

func (s *service) GetRecommendations(userID uuid.UUID, limit int) ([]*RecommendedArticle, error) {
	s.logger.Info("Getting recommendations for user " + userID.String() + " with limit " + fmt.Sprintf("%d", limit))

	// Validate limit
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Generate recommendations using default engine
	recommendations, err := s.defaultEngine.Recommend(userID, limit)
	if err != nil {
		s.logger.Error("Failed to generate recommendations for user " + userID.String() + " using engine '" + s.defaultEngine.Name() + "' with limit " + fmt.Sprintf("%d", limit) + ": " + err.Error())
		return nil, fmt.Errorf("failed to generate recommendations: %w", err)
	}

	// Validate results
	if recommendations == nil {
		recommendations = make([]*RecommendedArticle, 0)
	}

	// Log success
	s.logger.Info("Recommendations generated successfully for user " + userID.String() + ": " + fmt.Sprintf("%d", len(recommendations)) + " recommendations using engine '" + s.defaultEngine.Name() + "'")

	// Enhance recommendations with additional context
	for i, rec := range recommendations {
		if rec.Score > 0.8 {
			rec.Reason = "Highly " + rec.Reason
		} else if rec.Score < 0.3 {
			rec.Reason = "Potentially " + rec.Reason
		}
		recommendations[i] = rec
	}

	return recommendations, nil
}
