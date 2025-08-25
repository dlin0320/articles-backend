package recommendation

import (
	"testing"
	"time"

	"github.com/dustin/articles-backend/config"
	"github.com/dustin/articles-backend/internal/embedding"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecommendedArticle(t *testing.T) {
	t.Run("Create recommended article", func(t *testing.T) {
		articleID := uuid.New()
		userID := uuid.New()

		article := &Article{
			ID:          articleID,
			UserID:      userID,
			Title:       "Test Article",
			Description: "Test Description",
			URL:         "https://example.com",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		rec := &RecommendedArticle{
			Article:         article,
			Score:           0.85,
			Reason:          "Similar to articles you rated highly",
			RecommenderUsed: "content-based",
		}

		assert.Equal(t, article, rec.Article)
		assert.Equal(t, 0.85, rec.Score)
		assert.Equal(t, "Similar to articles you rated highly", rec.Reason)
		assert.Equal(t, "content-based", rec.RecommenderUsed)
	})
}

func TestBuildRecommendationResponse(t *testing.T) {
	userID := uuid.New()
	articleID1 := uuid.New()
	articleID2 := uuid.New()

	article1 := &Article{
		ID:    articleID1,
		Title: "Article 1",
	}

	article2 := &Article{
		ID:    articleID2,
		Title: "Article 2",
	}

	recommendations := []*RecommendedArticle{
		{
			Article:         article1,
			Score:           0.9,
			Reason:          "Highly similar content",
			RecommenderUsed: "hybrid",
		},
		{
			Article:         article2,
			Score:           0.7,
			Reason:          "Popular article",
			RecommenderUsed: "hybrid",
		},
	}

	response := BuildRecommendationResponse(recommendations, userID, "hybrid")

	assert.Len(t, response.Recommendations, 2)
	assert.Equal(t, userID, response.UserID)
	assert.Equal(t, "hybrid", response.EngineUsed)
	assert.Equal(t, 2, response.Count)
	assert.NotZero(t, response.GeneratedAt)

	// Verify recommendation details
	assert.Equal(t, articleID1, response.Recommendations[0].Article.ID)
	assert.Equal(t, 0.9, response.Recommendations[0].Score)
	assert.Equal(t, "Highly similar content", response.Recommendations[0].Reason)

	assert.Equal(t, articleID2, response.Recommendations[1].Article.ID)
	assert.Equal(t, 0.7, response.Recommendations[1].Score)
	assert.Equal(t, "Popular article", response.Recommendations[1].Reason)
}

func TestContentBasedEngine(t *testing.T) {
	// Create logger for testing
	logConfig := &config.LoggingConfig{
		Level:  "info",
		Format: "text",
	}
	log, err := logger.NewLogger(logConfig)
	require.NoError(t, err)

	t.Run("Recommend with user ratings", func(t *testing.T) {
		// Setup mocks
		mockArticleRepo := &mockArticleRepository{}
		mockRatingRepo := &mockRatingRepositoryWithRatings{}
		mockEmbeddingClient := &mockEmbeddingClient{}

		// Create engine
		engine := NewContentBasedEngine(mockArticleRepo, mockRatingRepo, mockEmbeddingClient, log)

		// Test recommendation
		userID := uuid.New()
		recommendations, err := engine.Recommend(userID, 10)

		assert.NoError(t, err)
		assert.NotEmpty(t, recommendations)
		assert.Equal(t, "content-based", engine.Name())

		// Verify recommendation details
		if len(recommendations) > 0 {
			rec := recommendations[0]
			assert.NotNil(t, rec.Article)
			assert.Greater(t, rec.Score, 0.0)
			assert.NotEmpty(t, rec.Reason)
			assert.Equal(t, "content-based", rec.RecommenderUsed)
		}
	})

	t.Run("Recommend with no user ratings", func(t *testing.T) {
		// Setup mocks - no ratings
		mockArticleRepo := &mockArticleRepository{}
		mockRatingRepo := &mockRatingRepository{} // Empty ratings
		mockEmbeddingClient := &mockEmbeddingClient{}

		// Create engine
		engine := NewContentBasedEngine(mockArticleRepo, mockRatingRepo, mockEmbeddingClient, log)

		// Test recommendation - should fall back to popular articles
		userID := uuid.New()
		recommendations, err := engine.Recommend(userID, 10)

		assert.NoError(t, err)
		// Should return popular articles as fallback
		if len(recommendations) > 0 {
			rec := recommendations[0]
			assert.Contains(t, rec.Reason, "Popular article")
		}
	})

	t.Run("Calculate weighted profile", func(t *testing.T) {
		mockEmbeddingClient := &mockEmbeddingClient{}
		engine := NewContentBasedEngine(&mockArticleRepository{}, &mockRatingRepository{}, mockEmbeddingClient, log)

		// Test that the engine correctly processes embeddings internally
		// We can't test the private method directly, but we can test the overall behavior
		userID := uuid.New()
		recommendations, err := engine.Recommend(userID, 5)

		// Should succeed without error
		assert.NoError(t, err)
		// Should have some recommendations (from popular fallback if no ratings match embedding criteria)
		assert.NotNil(t, recommendations)
	})
}

type mockArticleRepository struct{}

func (m *mockArticleRepository) FindByID(id uuid.UUID) (*Article, error) {
	return &Article{ID: id, Title: "Mock Article"}, nil
}

func (m *mockArticleRepository) FindAll() ([]*Article, error) {
	return []*Article{}, nil
}

func (m *mockArticleRepository) FindPopular(limit int) ([]*Article, error) {
	// Return mock popular articles
	return []*Article{
		{
			ID:          uuid.New(),
			Title:       "Popular Article 1",
			Description: "Popular description",
			URL:         "https://popular1.com",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}, nil
}

func (m *mockArticleRepository) FindSimilar(embedding []float64, userID uuid.UUID, limit int) ([]*Article, error) {
	// Return mock similar articles based on embedding
	return []*Article{
		{
			ID:          uuid.New(),
			Title:       "Similar Article 1",
			Description: "Similar content",
			URL:         "https://similar1.com",
			Embedding:   embedding, // Same embedding for similarity
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			Title:       "Similar Article 2",
			Description: "Related content",
			URL:         "https://similar2.com",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}, nil
}

type mockRatingRepository struct{}

func (m *mockRatingRepository) FindByUserID(userID uuid.UUID) ([]*Rating, error) {
	return []*Rating{}, nil
}

func (m *mockRatingRepository) GetAverageRating(articleID uuid.UUID) (float64, int, error) {
	return 4.0, 10, nil
}

// mockRatingRepositoryWithRatings returns mock ratings for testing
type mockRatingRepositoryWithRatings struct{}

func (m *mockRatingRepositoryWithRatings) FindByUserID(userID uuid.UUID) ([]*Rating, error) {
	// Return mock ratings with high scores to trigger embedding generation
	return []*Rating{
		{
			UserID:    userID,
			ArticleID: uuid.New(),
			Score:     5,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			UserID:    userID,
			ArticleID: uuid.New(),
			Score:     4,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}, nil
}

func (m *mockRatingRepositoryWithRatings) GetAverageRating(articleID uuid.UUID) (float64, int, error) {
	return 4.5, 5, nil
}

// mockEmbeddingClient simulates the embedding service
type mockEmbeddingClient struct{}

func (m *mockEmbeddingClient) GetEmbedding(text string) ([]float64, error) {
	// Return a mock embedding based on text length for deterministic testing
	embeddingSize := 384
	embedding := make([]float64, embeddingSize)
	for i := range embedding {
		embedding[i] = float64(len(text)%100) / 100.0 // Normalize based on text length
	}
	return embedding, nil
}

func (m *mockEmbeddingClient) GetBatchEmbeddings(texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i, text := range texts {
		embedding, _ := m.GetEmbedding(text)
		embeddings[i] = embedding
	}
	return embeddings, nil
}

func (m *mockEmbeddingClient) CalculateSimilarity(embedding1, embedding2 []float64) (float64, error) {
	return 0.85, nil // Mock high similarity
}

func (m *mockEmbeddingClient) HealthCheck() (*embedding.HealthResponse, error) {
	return &embedding.HealthResponse{
		Status:               "healthy",
		EmbeddingModel:       "mock-model",
		ClassifierModel:      "mock-classifier",
		EmbeddingModelLoaded: true,
		ClassifierLoaded:     true,
	}, nil
}

func (m *mockEmbeddingClient) ClassifyContent(text string) (*embedding.ClassifyResponse, error) {
	return &embedding.ClassifyResponse{
		Text:       text,
		IsArticle:  true,
		Confidence: 0.8,
	}, nil
}

func (m *mockEmbeddingClient) ClassifyBatchContent(texts []string) (*embedding.BatchClassifyResponse, error) {
	results := make([]embedding.ClassifyResult, len(texts))
	for i, text := range texts {
		results[i] = embedding.ClassifyResult{
			Text:       text,
			IsArticle:  true,
			Confidence: 0.8,
			Index:      i,
		}
	}
	return &embedding.BatchClassifyResponse{
		Results:   results,
		Count:     len(results),
		Processed: len(results),
	}, nil
}
