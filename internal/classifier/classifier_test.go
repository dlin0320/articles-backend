package classifier

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dustin/articles-backend/config"
	"github.com/dustin/articles-backend/internal/embedding"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create test classifier
func createTestClassifier() (*ReadabilityClassifier, error) {
	cfg := &config.ClassifierConfig{
		HTTPTimeout: "30s",
	}

	// Create a simple real embedding client for tests (it won't actually be used in most tests)
	embeddingClient := embedding.NewClient("http://localhost:8001")

	logCfg := &config.LoggingConfig{
		Level: "error",
	}
	log, _ := logger.NewLogger(logCfg)

	return NewReadabilityClassifier(cfg, embeddingClient, log)
}

func TestNewReadabilityClassifier(t *testing.T) {
	classifier, err := createTestClassifier()

	require.NoError(t, err)
	assert.NotNil(t, classifier)
	assert.Equal(t, "readability", classifier.Name())
	assert.True(t, classifier.IsHealthy())
}

func TestReadabilityClassifier_FetchHTML_Success(t *testing.T) {
	// Create test server with known content
	testHTML := `<html><head><title>Test Article</title></head><body><h1>Test Title</h1><p>Test content here.</p></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testHTML))
	}))
	defer server.Close()

	classifier, err := createTestClassifier()
	require.NoError(t, err)

	html, err := classifier.fetchHTML(server.URL)

	assert.NoError(t, err)
	assert.Equal(t, testHTML, html)
}

func TestReadabilityClassifier_FetchHTML_UnknownContentLength(t *testing.T) {
	// Test the edge case that was causing panics
	testHTML := `<html><body><h1>Test</h1></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't set Content-Length header (simulates unknown content length)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testHTML))
	}))
	defer server.Close()

	classifier, err := createTestClassifier()
	require.NoError(t, err)

	html, err := classifier.fetchHTML(server.URL)

	assert.NoError(t, err)
	assert.Equal(t, testHTML, html)
}

func TestReadabilityClassifier_FetchHTML_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	classifier, err := createTestClassifier()
	require.NoError(t, err)

	html, err := classifier.fetchHTML(server.URL)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
	assert.Empty(t, html)
}

func TestReadabilityClassifier_FetchHTML_LargeContent(t *testing.T) {
	// Test with content larger than 5MB limit
	largeContent := strings.Repeat("x", 6*1024*1024) // 6MB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeContent))
	}))
	defer server.Close()

	classifier, err := createTestClassifier()
	require.NoError(t, err)

	html, err := classifier.fetchHTML(server.URL)

	assert.NoError(t, err)
	// Should be truncated to 5MB limit (may be slightly over due to chunk reading)
	assert.Less(t, len(html), 6*1024*1024) // Allow for slight overflow due to chunk processing
}

func TestReadabilityClassifier_Classify_Success(t *testing.T) {
	testHTML := `<html><head><title>Test Article</title><meta name="description" content="Test description"></head><body><article><h1>Test Title</h1><p>This is a test article with enough content to be classified as a real article. It has multiple paragraphs and meaningful content.</p><p>Another paragraph with more content to ensure we meet the minimum length requirements for article classification.</p></article></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testHTML))
	}))
	defer server.Close()

	classifier, err := createTestClassifier()
	require.NoError(t, err)

	result, err := classifier.Classify(server.URL, "")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Note: Actual classification may depend on embedding service availability
	assert.Equal(t, "Test Article", result.Title)
	assert.Greater(t, result.WordCount, 0)
	assert.Equal(t, "readability", result.ClassifierUsed)
}

func TestReadabilityClassifier_Classify_WithProvidedHTML(t *testing.T) {
	testHTML := `<html><head><title>Test Article</title></head><body><h1>Test Title</h1><p>Test content provided directly.</p></body></html>`

	classifier, err := createTestClassifier()
	require.NoError(t, err)

	result, err := classifier.Classify("https://example.com", testHTML)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Article", result.Title)
	assert.Equal(t, "readability", result.ClassifierUsed)
}

func TestReadabilityClassifier_Classify_EmbeddingServiceError(t *testing.T) {
	testHTML := `<html><head><title>Test</title></head><body><h1>Test</h1><p>Content</p></body></html>`

	classifier, err := createTestClassifier()
	require.NoError(t, err)

	result, err := classifier.Classify("https://example.com", testHTML)

	assert.NoError(t, err) // Should not error, just fall back to readability-only
	assert.NotNil(t, result)
	assert.Equal(t, "Test", result.Title)
}

func TestReadabilityClassifier_Classify_InvalidURL(t *testing.T) {
	classifier, err := createTestClassifier()
	require.NoError(t, err)

	result, err := classifier.Classify("not-a-valid-url", "")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestReadabilityClassifier_Classify_NetworkTimeout(t *testing.T) {
	// Server that never responds to simulate timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Longer than our timeout
	}))
	defer server.Close()

	// Create classifier with very short timeout
	cfg := &config.ClassifierConfig{
		HTTPTimeout: "100ms", // Very short timeout
	}
	embeddingClient := embedding.NewClient("http://localhost:8001")
	logCfg := &config.LoggingConfig{Level: "error"}
	log, _ := logger.NewLogger(logCfg)
	classifier, err := NewReadabilityClassifier(cfg, embeddingClient, log)
	require.NoError(t, err)

	result, err := classifier.Classify(server.URL, "")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "deadline exceeded")
}

func TestReadabilityClassifier_IsHealthy(t *testing.T) {
	// Test healthy classifier
	healthyClassifier, err := createTestClassifier()
	require.NoError(t, err)
	assert.True(t, healthyClassifier.IsHealthy())

	// Test initially healthy classifier (embedding client health not directly tested here)
	unhealthyClassifier, err := createTestClassifier()
	require.NoError(t, err)
	assert.True(t, unhealthyClassifier.IsHealthy()) // Classifier itself is healthy
}

// Test edge cases for content that might cause issues
func TestReadabilityClassifier_Classify_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
	}))
	defer server.Close()

	classifier, err := createTestClassifier()
	require.NoError(t, err)

	result, err := classifier.Classify(server.URL, "")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Empty content should result in low confidence/not an article
	assert.Equal(t, 0, result.WordCount)
}

func TestReadabilityClassifier_Classify_NonHTMLContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "not html content"}`))
	}))
	defer server.Close()

	classifier, err := createTestClassifier()
	require.NoError(t, err)

	result, err := classifier.Classify(server.URL, "")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Should still try to process it, readability can handle various content types
	assert.Equal(t, "readability", result.ClassifierUsed)
}
