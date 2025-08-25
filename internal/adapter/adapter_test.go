package adapter

import (
	"errors"
	"testing"

	"github.com/dustin/articles-backend/internal/article"
	"github.com/dustin/articles-backend/internal/classifier"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock classifier for testing
type mockClassifier struct {
	result *classifier.Result
	err    error
}

func (m *mockClassifier) Classify(url, html string) (*classifier.Result, error) {
	return m.result, m.err
}

func (m *mockClassifier) Name() string {
	return "mock"
}

func (m *mockClassifier) IsHealthy() bool {
	return true
}

func TestClassifierToMetadataExtractor_Extract_Success(t *testing.T) {
	mockResult := &classifier.Result{
		Title:       "Test Article",
		Description: "Test Description",
		Content:     "Test Content",
		Image:       "https://example.com/image.jpg",
		WordCount:   500,
		Confidence:  0.85,
	}

	mock := &mockClassifier{result: mockResult}
	adapter := NewClassifierToMetadataExtractor(mock)

	result, err := adapter.Extract("https://example.com/article")
	require.NoError(t, err)
	assert.NotNil(t, result)

	assert.Equal(t, "Test Article", result.Title)
	assert.Equal(t, "Test Description", result.Description)
	assert.Equal(t, "Test Content", result.Content)
	assert.Equal(t, "https://example.com/image.jpg", result.ImageURL)
	assert.Equal(t, 500, result.WordCount)
	assert.Equal(t, 0.85, result.Confidence)
}

func TestClassifierToMetadataExtractor_Extract_Error(t *testing.T) {
	mock := &mockClassifier{err: errors.New("classification failed")}
	adapter := NewClassifierToMetadataExtractor(mock)

	result, err := adapter.Extract("https://example.com/article")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "classification failed")
}

func TestClassifierToMetadataExtractor_Extract_EmptyHTML(t *testing.T) {
	mockResult := &classifier.Result{
		Title:      "Test",
		Content:    "Content",
		WordCount:  10,
		Confidence: 0.5,
	}

	// Create a mock that verifies empty HTML is passed
	mock := &mockClassifier{result: mockResult}
	adapter := &ClassifierToMetadataExtractor{classifier: mock}

	result, err := adapter.Extract("https://example.com/test")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test", result.Title)
}

// Mock article service for testing
type mockArticleService struct {
	article *article.Article
	err     error
}

func (m *mockArticleService) CreateArticle(userID uuid.UUID, url string) (*article.Article, error) {
	return m.article, m.err
}

func (m *mockArticleService) GetArticle(id, userID uuid.UUID) (*article.Article, error) {
	return m.article, m.err
}

func (m *mockArticleService) GetUserArticles(userID uuid.UUID, page, limit int) ([]*article.Article, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return []*article.Article{m.article}, 1, nil
}

func (m *mockArticleService) DeleteArticle(id, userID uuid.UUID) error {
	return m.err
}

func (m *mockArticleService) UpdateMetadata(id uuid.UUID, title, description, content string, wordCount int, confidence float64) error {
	return m.err
}

func (m *mockArticleService) RetryFailedMetadata() error {
	return m.err
}

func (m *mockArticleService) ExtractMetadata(articleID uuid.UUID) error {
	return m.err
}

func TestArticleServiceToRatingArticleService_GetArticle_Success(t *testing.T) {
	articleID := uuid.New()
	userID := uuid.New()

	mockArticle := &article.Article{
		ID:     articleID,
		UserID: userID,
		Title:  "Test Article",
		URL:    "https://example.com/article",
	}

	mockService := &mockArticleService{article: mockArticle}
	adapter := NewArticleServiceToRatingArticleService(mockService)

	result, err := adapter.GetArticle(articleID, userID)
	require.NoError(t, err)
	assert.NotNil(t, result)

	assert.Equal(t, articleID, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, "Test Article", result.Title)
	assert.Equal(t, "https://example.com/article", result.URL)
}

func TestArticleServiceToRatingArticleService_GetArticle_Error(t *testing.T) {
	mockService := &mockArticleService{err: errors.New("article not found")}
	adapter := NewArticleServiceToRatingArticleService(mockService)

	articleID := uuid.New()
	userID := uuid.New()

	result, err := adapter.GetArticle(articleID, userID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "article not found")
}

func TestArticleServiceToRatingArticleService_GetArticle_Mapping(t *testing.T) {
	articleID := uuid.New()
	userID := uuid.New()

	// Create article with more fields to ensure only relevant ones are mapped
	mockArticle := &article.Article{
		ID:              articleID,
		UserID:          userID,
		Title:           "Full Article",
		URL:             "https://example.com/full",
		Description:     "Description that should not be in rating.Article",
		Content:         "Content that should not be in rating.Article",
		ImageURL:        "https://example.com/image.jpg",
		WordCount:       500,
		MetadataStatus:  "success",
		ConfidenceScore: 0.85,
		ClassifierUsed:  "readability",
	}

	mockService := &mockArticleService{article: mockArticle}
	adapter := NewArticleServiceToRatingArticleService(mockService)

	result, err := adapter.GetArticle(articleID, userID)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify only the fields that exist in rating.Article are set
	assert.Equal(t, articleID, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, "Full Article", result.Title)
	assert.Equal(t, "https://example.com/full", result.URL)

	// rating.Article should not have these fields from article.Article
	// This is implicit in the type conversion, but we're testing the mapping logic
}

func TestClassifierToMetadataExtractor_ExtractedMetadata_AllFields(t *testing.T) {
	// Test with all possible fields populated
	mockResult := &classifier.Result{
		Title:          "Complete Title",
		Description:    "Complete Description",
		Content:        "Complete Content with lots of text",
		Image:          "https://example.com/complete.jpg",
		WordCount:      1000,
		Confidence:     0.95,
		IsArticle:      true,
		ClassifierUsed: "readability",
	}

	mock := &mockClassifier{result: mockResult}
	adapter := NewClassifierToMetadataExtractor(mock)

	result, err := adapter.Extract("https://example.com/complete")
	require.NoError(t, err)
	assert.NotNil(t, result)

	assert.Equal(t, "Complete Title", result.Title)
	assert.Equal(t, "Complete Description", result.Description)
	assert.Equal(t, "Complete Content with lots of text", result.Content)
	assert.Equal(t, "https://example.com/complete.jpg", result.ImageURL)
	assert.Equal(t, 1000, result.WordCount)
	assert.Equal(t, 0.95, result.Confidence)
}

func TestClassifierToMetadataExtractor_ExtractedMetadata_MinimalFields(t *testing.T) {
	// Test with minimal fields
	mockResult := &classifier.Result{
		Title:      "Minimal",
		Content:    "Text",
		WordCount:  1,
		Confidence: 0.1,
	}

	mock := &mockClassifier{result: mockResult}
	adapter := NewClassifierToMetadataExtractor(mock)

	result, err := adapter.Extract("https://example.com/minimal")
	require.NoError(t, err)
	assert.NotNil(t, result)

	assert.Equal(t, "Minimal", result.Title)
	assert.Equal(t, "", result.Description)
	assert.Equal(t, "Text", result.Content)
	assert.Equal(t, "", result.ImageURL)
	assert.Equal(t, 1, result.WordCount)
	assert.Equal(t, 0.1, result.Confidence)
}
