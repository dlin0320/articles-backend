package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EmbeddingClient defines the interface for embedding operations
type EmbeddingClient interface {
	GetEmbedding(text string) ([]float64, error)
	GetBatchEmbeddings(texts []string) ([][]float64, error)
	CalculateSimilarity(embedding1, embedding2 []float64) (float64, error)
	HealthCheck() (*HealthResponse, error)
	ClassifyContent(text string) (*ClassifyResponse, error)
	ClassifyBatchContent(texts []string) (*BatchClassifyResponse, error)
}

// Client handles communication with the embedding microservice
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient creates a new embedding service client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EmbedRequest represents a single text embedding request
type EmbedRequest struct {
	Text string `json:"text"`
}

// EmbedResponse represents the embedding response
type EmbedResponse struct {
	Text      string    `json:"text"`
	Embedding []float64 `json:"embedding"`
	Dimension int       `json:"dimension"`
}

// BatchEmbedRequest represents multiple text embedding request
type BatchEmbedRequest struct {
	Texts []string `json:"texts"`
}

// BatchEmbedResponse represents the batch embedding response
type BatchEmbedResponse struct {
	Texts      []string    `json:"texts"`
	Embeddings [][]float64 `json:"embeddings"`
	Count      int         `json:"count"`
	Dimension  int         `json:"dimension"`
}

// SimilarityRequest represents a similarity calculation request
type SimilarityRequest struct {
	Embedding1 []float64 `json:"embedding1"`
	Embedding2 []float64 `json:"embedding2"`
}

// SimilarityResponse represents the similarity response
type SimilarityResponse struct {
	Similarity float64 `json:"similarity"`
}

// ClassifyRequest represents a content classification request
type ClassifyRequest struct {
	Text string `json:"text"`
}

// ClassifyResponse represents the classification response
type ClassifyResponse struct {
	Text       string                 `json:"text"`
	IsArticle  bool                   `json:"is_article"`
	Confidence float64                `json:"confidence"`
	Details    map[string]interface{} `json:"classification_details,omitempty"`
}

// BatchClassifyRequest represents multiple text classification request
type BatchClassifyRequest struct {
	Texts []string `json:"texts"`
}

// BatchClassifyResponse represents the batch classification response
type BatchClassifyResponse struct {
	Results   []ClassifyResult `json:"results"`
	Count     int              `json:"count"`
	Processed int              `json:"processed"`
}

// ClassifyResult represents a single classification result
type ClassifyResult struct {
	Text       string  `json:"text"`
	IsArticle  bool    `json:"is_article"`
	Confidence float64 `json:"confidence"`
	Index      int     `json:"index"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status               string `json:"status"`
	EmbeddingModel       string `json:"embedding_model"`
	ClassifierModel      string `json:"classifier_model"`
	EmbeddingModelLoaded bool   `json:"embedding_model_loaded"`
	ClassifierLoaded     bool   `json:"classifier_loaded"`
}

// GetEmbedding generates an embedding for a single text
func (c *Client) GetEmbedding(text string) ([]float64, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text provided")
	}

	reqBody := EmbedRequest{Text: text}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/embed", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service error (status %d): %s", resp.StatusCode, string(body))
	}

	var embedResp EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embedResp.Embedding, nil
}

// GetBatchEmbeddings generates embeddings for multiple texts
func (c *Client) GetBatchEmbeddings(texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("empty texts list provided")
	}

	reqBody := BatchEmbedRequest{Texts: texts}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/embed/batch", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service error (status %d): %s", resp.StatusCode, string(body))
	}

	var embedResp BatchEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embedResp.Embeddings, nil
}

// CalculateSimilarity calculates cosine similarity between two embeddings
func (c *Client) CalculateSimilarity(embedding1, embedding2 []float64) (float64, error) {
	if len(embedding1) != len(embedding2) {
		return 0, fmt.Errorf("embedding dimensions don't match: %d vs %d", len(embedding1), len(embedding2))
	}

	reqBody := SimilarityRequest{
		Embedding1: embedding1,
		Embedding2: embedding2,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/similarity", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("embedding service error (status %d): %s", resp.StatusCode, string(body))
	}

	var simResp SimilarityResponse
	if err := json.NewDecoder(resp.Body).Decode(&simResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return simResp.Similarity, nil
}

// HealthCheck checks if the embedding service is healthy
func (c *Client) HealthCheck() (*HealthResponse, error) {
	resp, err := c.client.Get(c.baseURL + "/health")
	if err != nil {
		return nil, fmt.Errorf("failed to make health check request: %w", err)
	}
	defer resp.Body.Close()

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}

	return &healthResp, nil
}

// ClassifyContent classifies if content is article-worthy using ML model
func (c *Client) ClassifyContent(text string) (*ClassifyResponse, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text provided")
	}

	reqBody := ClassifyRequest{Text: text}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/classify", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("classification service error (status %d): %s", resp.StatusCode, string(body))
	}

	var classifyResp ClassifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&classifyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &classifyResp, nil
}

// ClassifyBatchContent classifies multiple texts for article-worthiness
func (c *Client) ClassifyBatchContent(texts []string) (*BatchClassifyResponse, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("empty texts list provided")
	}

	reqBody := BatchClassifyRequest{Texts: texts}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/classify/batch", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("batch classification service error (status %d): %s", resp.StatusCode, string(body))
	}

	var batchResp BatchClassifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &batchResp, nil
}
