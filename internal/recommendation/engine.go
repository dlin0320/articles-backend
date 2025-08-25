package recommendation

import (
	"github.com/dustin/articles-backend/internal/embedding"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/google/uuid"
)

// ContentBasedEngine recommends articles based on content similarity
type ContentBasedEngine struct {
	articleRepo     ArticleRepository
	ratingRepo      RatingRepository
	embeddingClient embedding.EmbeddingClient
	logger          *logger.Logger
}

// NewContentBasedEngine creates a new content-based recommendation engine
func NewContentBasedEngine(articleRepo ArticleRepository, ratingRepo RatingRepository, embeddingClient embedding.EmbeddingClient, log *logger.Logger) Engine {
	return &ContentBasedEngine{
		articleRepo:     articleRepo,
		ratingRepo:      ratingRepo,
		embeddingClient: embeddingClient,
		logger:          log.WithComponent("recommendation-engine"),
	}
}

func (c *ContentBasedEngine) Recommend(userID uuid.UUID, limit int) ([]*RecommendedArticle, error) {
	c.logger.Info("Generating recommendations for user " + userID.String())

	// Get user's highly rated articles to build profile
	userRatings, err := c.ratingRepo.FindByUserID(userID)
	if err != nil {
		c.logger.Error("Failed to get user ratings: " + err.Error())
		return nil, err
	}

	// Collect highly rated articles for embedding generation
	var userTexts []string
	var userWeights []float64
	for _, rating := range userRatings {
		if rating.Score >= 4 { // Only consider high ratings
			article, err := c.articleRepo.FindByID(rating.ArticleID)
			if err != nil {
				c.logger.Error("Failed to get article " + rating.ArticleID.String() + ": " + err.Error())
				continue
			}

			text := article.Title + " " + article.Description
			if text != "" {
				userTexts = append(userTexts, text)
				userWeights = append(userWeights, float64(rating.Score)/5.0)
			}
		}
	}

	// If no profile can be built, use popular articles as default
	if len(userTexts) == 0 {
		c.logger.Info("No user profile available, using popular articles as default")
		return c.recommendPopular(userID, limit)
	}

	// Generate embeddings for user's preferred articles
	userEmbeddings, err := c.embeddingClient.GetBatchEmbeddings(userTexts)
	if err != nil {
		c.logger.Error("Failed to get user embeddings: " + err.Error())
		return nil, err
	}

	// Calculate weighted user profile embedding
	userProfile := c.calculateWeightedProfile(userEmbeddings, userWeights)

	// Use vector similarity search instead of loading all articles
	// This is much more scalable as it uses database indexing
	similarArticles, err := c.articleRepo.FindSimilar(userProfile, userID, limit*2)
	if err != nil {
		c.logger.Error("Failed to find similar articles: " + err.Error())
		return nil, err
	}

	if len(similarArticles) == 0 {
		c.logger.Info("No similar articles found for user")
		return []*RecommendedArticle{}, nil
	}

	// Convert similar articles to recommendations
	// The similarity score comes from the database query (1 - cosine_distance)
	recommendations := make([]*RecommendedArticle, 0, len(similarArticles))
	for _, article := range similarArticles {
		// pgvector returns cosine distance (0-2), convert to similarity (1-0)
		// For now, we'll use a fixed high confidence since articles are pre-filtered
		similarityScore := 0.8 // High confidence for vector similarity matches

		recommendations = append(recommendations, &RecommendedArticle{
			Article:         article,
			Score:           similarityScore,
			Reason:          "Similar to articles you rated highly",
			RecommenderUsed: c.Name(),
		})
	}

	// Limit results (already sorted by similarity from database)
	if len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	c.logger.Info("Generated recommendations for user " + userID.String())
	return recommendations, nil
}

func (c *ContentBasedEngine) recommendPopular(userID uuid.UUID, limit int) ([]*RecommendedArticle, error) {
	c.logger.Info("Using popular articles as default recommendation for user " + userID.String())

	popularArticles, err := c.articleRepo.FindPopular(limit * 2) // Get more to filter user's own
	if err != nil {
		c.logger.Error("Failed to get popular articles: " + err.Error())
		return nil, err
	}

	recommendations := make([]*RecommendedArticle, 0)
	for _, article := range popularArticles {
		if article.UserID == userID {
			continue // Skip user's own articles
		}

		recommendations = append(recommendations, &RecommendedArticle{
			Article:         article,
			Score:           0.7, // Good confidence for popular content
			Reason:          "Popular article (no rating history available)",
			RecommenderUsed: c.Name(),
		})

		if len(recommendations) >= limit {
			break
		}
	}

	c.logger.Info("Generated popular recommendations for user " + userID.String())
	return recommendations, nil
}

// calculateWeightedProfile creates a weighted average embedding from multiple embeddings
func (c *ContentBasedEngine) calculateWeightedProfile(embeddings [][]float64, weights []float64) []float64 {
	if len(embeddings) == 0 || len(embeddings) != len(weights) {
		return nil
	}

	// Assume all embeddings have the same dimension
	dimension := len(embeddings[0])
	profile := make([]float64, dimension)
	totalWeight := 0.0

	// Calculate weighted sum
	for i, embedding := range embeddings {
		weight := weights[i]
		totalWeight += weight
		for j, value := range embedding {
			profile[j] += value * weight
		}
	}

	// Normalize by total weight
	if totalWeight > 0 {
		for i := range profile {
			profile[i] /= totalWeight
		}
	}

	return profile
}

func (c *ContentBasedEngine) Name() string {
	return "content-based"
}
