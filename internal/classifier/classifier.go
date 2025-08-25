package classifier

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/articles-backend/config"
	"github.com/dustin/articles-backend/internal/embedding"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/go-shiori/go-readability"
)

// Classifier defines content classification capabilities
type Classifier interface {
	Classify(url string, html string) (*Result, error)
	Name() string
	IsHealthy() bool
}

// Result contains classification output with metadata
type Result struct {
	IsArticle      bool      `json:"is_article"`
	Confidence     float64   `json:"confidence"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Image          string    `json:"image"`
	Content        string    `json:"content"`
	WordCount      int       `json:"word_count"`
	ClassifierUsed string    `json:"classifier_used"`
	ProcessedAt    time.Time `json:"processed_at"`
}

// ReadabilityClassifier implements article extraction using go-readability + ML classification
type ReadabilityClassifier struct {
	minConfidenceScore float64
	httpTimeout        time.Duration
	userAgent          string
	logger             *logger.Logger
	client             *http.Client
	embeddingClient    *embedding.Client
	isHealthy          bool
}

// NewReadabilityClassifier creates a content classifier with validation and defaults
func NewReadabilityClassifier(cfg *config.ClassifierConfig, embeddingClient *embedding.Client, log *logger.Logger) (*ReadabilityClassifier, error) {
	// Set defaults for nil or empty config values
	var minConfidence float64 = 0.6
	if cfg != nil && cfg.MinConfidenceScore != "" {
		confidence, err := strconv.ParseFloat(cfg.MinConfidenceScore, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid min confidence score '%s': %v", cfg.MinConfidenceScore, err)
		}
		minConfidence = confidence
	}

	var httpTimeout time.Duration = 30 * time.Second
	if cfg != nil && cfg.HTTPTimeout != "" {
		timeout, err := time.ParseDuration(cfg.HTTPTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid HTTP timeout '%s': %v", cfg.HTTPTimeout, err)
		}
		httpTimeout = timeout
	}

	userAgent := "Articles-Backend-Bot/1.0"
	if cfg != nil && cfg.UserAgent != "" {
		userAgent = cfg.UserAgent
	}

	return &ReadabilityClassifier{
		minConfidenceScore: minConfidence,
		httpTimeout:        httpTimeout,
		userAgent:          userAgent,
		logger:             log.WithComponent("readability-classifier"),
		client: &http.Client{
			Timeout: httpTimeout,
		},
		embeddingClient: embeddingClient,
		isHealthy:       true,
	}, nil
}

func (r *ReadabilityClassifier) Name() string {
	return "readability"
}

func (r *ReadabilityClassifier) IsHealthy() bool {
	return r.isHealthy
}

func (r *ReadabilityClassifier) Classify(urlStr string, html string) (*Result, error) {
	r.logger.Info("Starting content classification for URL: " + urlStr)

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		r.logger.Error("Invalid URL: " + urlStr + ", error: " + err.Error())
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// If HTML is empty, try to fetch it
	if html == "" {
		html, err = r.fetchHTML(urlStr)
		if err != nil {
			r.logger.Error("Failed to fetch HTML for " + urlStr + ": " + err.Error())
			return nil, fmt.Errorf("failed to fetch HTML: %w", err)
		}
	}

	// Use readability to parse content
	article, err := readability.FromReader(strings.NewReader(html), parsedURL)
	if err != nil {
		r.logger.Error("Readability parsing failed for " + urlStr + ": " + err.Error())
		return nil, fmt.Errorf("readability parsing failed: %w", err)
	}

	// Calculate basic metrics
	wordCount := len(strings.Fields(article.TextContent))

	// Use ML-based classification for article worthiness
	confidence, isArticle := r.classifyWithML(article, urlStr)

	// Clean and validate content
	title := r.cleanText(article.Title)
	description := r.cleanText(article.Excerpt)
	content := r.cleanText(article.TextContent)
	imageURL := r.validateImageURL(article.Image, parsedURL)

	// Return error if ML classification failed
	if confidence < 0 {
		r.logger.Error("ML classification failed for " + urlStr)
		return nil, fmt.Errorf("ML classification failed")
	}

	result := &Result{
		IsArticle:      isArticle,
		Confidence:     confidence,
		Title:          title,
		Description:    description,
		Image:          imageURL,
		Content:        content,
		WordCount:      wordCount,
		ClassifierUsed: r.Name(),
		ProcessedAt:    time.Now(),
	}

	r.logger.Info("Content classification completed for " + urlStr)

	return result, nil
}

func (r *ReadabilityClassifier) fetchHTML(urlStr string) (string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", r.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := r.client.Do(req)
	if err != nil {
		r.isHealthy = false
		return "", err
	}
	defer resp.Body.Close()

	r.isHealthy = true

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	buf := make([]byte, 0, resp.ContentLength)
	for {
		chunk := make([]byte, 1024)
		n, err := resp.Body.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
		}
		if err != nil {
			break
		}
		// Limit response size to prevent memory issues
		if len(buf) > 5*1024*1024 { // 5MB limit
			break
		}
	}

	return string(buf), nil
}

// classifyWithML uses machine learning model for article classification
func (r *ReadabilityClassifier) classifyWithML(article readability.Article, urlStr string) (confidence float64, isArticle bool) {
	// Prepare text for classification (combine title, excerpt, and content)
	classificationText := strings.TrimSpace(article.Title)
	if article.Excerpt != "" {
		classificationText += " " + strings.TrimSpace(article.Excerpt)
	}
	if article.TextContent != "" {
		classificationText += " " + strings.TrimSpace(article.TextContent)
	}

	// Error if no content to classify
	if classificationText == "" {
		r.logger.Error("No content to classify for URL: " + urlStr)
		return 0, false
	}

	// Call ML classification service
	result, err := r.embeddingClient.ClassifyContent(classificationText)
	if err != nil {
		r.logger.Error("ML classification failed for " + urlStr + ": " + err.Error())
		return 0, false // Return error via negative confidence
	}

	r.logger.Info("ML classification result for " + urlStr + ": confidence=" + fmt.Sprintf("%.2f", result.Confidence) + ", is_article=" + fmt.Sprintf("%t", result.IsArticle))

	// Apply minimum confidence threshold
	isArticleResult := result.IsArticle && result.Confidence >= r.minConfidenceScore

	return result.Confidence, isArticleResult
}

func (r *ReadabilityClassifier) cleanText(text string) string {
	// Basic text cleaning
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n\n\n", "\n\n") // Reduce excessive newlines
	text = strings.ReplaceAll(text, "\t", " ")        // Replace tabs with spaces

	// Remove excessive whitespace
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	return text
}

func (r *ReadabilityClassifier) validateImageURL(imageURL string, baseURL *url.URL) string {
	if imageURL == "" {
		return ""
	}

	// Parse image URL
	imgURL, err := url.Parse(imageURL)
	if err != nil {
		return ""
	}

	// Resolve relative URLs
	if !imgURL.IsAbs() && baseURL != nil {
		imgURL = baseURL.ResolveReference(imgURL)
	}

	// Validate scheme
	if imgURL.Scheme != "http" && imgURL.Scheme != "https" {
		return ""
	}

	return imgURL.String()
}
