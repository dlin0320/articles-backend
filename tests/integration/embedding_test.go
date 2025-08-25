//go:build integration
// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type EmbeddingTestSuite struct {
	suite.Suite
	client *http.Client
}

func (suite *EmbeddingTestSuite) SetupSuite() {
	suite.client = &http.Client{Timeout: 30 * time.Second}
}

func (suite *EmbeddingTestSuite) TestEmbeddingServiceHealth() {
	resp, err := suite.client.Get(EmbeddingServiceBaseURL + "/health")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "healthy", health["status"])
	assert.Equal(suite.T(), true, health["embedding_model_loaded"])
	assert.Equal(suite.T(), true, health["classifier_loaded"])
	assert.Equal(suite.T(), true, health["database_healthy"])
	assert.Equal(suite.T(), "all-MiniLM-L6-v2", health["embedding_model"])
	assert.Equal(suite.T(), "distilbert-base-uncased-finetuned-sst-2-english", health["classifier_model"])
}

func (suite *EmbeddingTestSuite) TestGenerateEnglishEmbedding() {
	embedData := map[string]string{
		"text": "This is a test article about artificial intelligence and machine learning technologies.",
	}

	jsonData, _ := json.Marshal(embedData)
	resp, err := suite.client.Post(
		EmbeddingServiceBaseURL+"/embed",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var embedResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&embedResp)
	require.NoError(suite.T(), err)

	// Verify embedding structure
	assert.Equal(suite.T(), float64(384), embedResp["dimension"])
	assert.Equal(suite.T(), "This is a test article about artificial intelligence and machine learning technologies.", embedResp["text"])
	
	embedding := embedResp["embedding"].([]interface{})
	assert.Equal(suite.T(), 384, len(embedding))
	
	// Verify embedding values are valid floats
	for i, val := range embedding {
		floatVal, ok := val.(float64)
		require.True(suite.T(), ok, "Embedding value at index %d should be a float", i)
		assert.False(suite.T(), math.IsNaN(floatVal), "Embedding value should not be NaN")
		assert.False(suite.T(), math.IsInf(floatVal, 0), "Embedding value should not be infinite")
	}
}

func (suite *EmbeddingTestSuite) TestGenerateChineseEmbedding() {
	embedData := map[string]string{
		"text": "這是一篇關於人工智慧和機器學習的測試文章。",
	}

	jsonData, _ := json.Marshal(embedData)
	resp, err := suite.client.Post(
		EmbeddingServiceBaseURL+"/embed",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var embedResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&embedResp)
	require.NoError(suite.T(), err)

	// Verify embedding structure for Chinese text
	assert.Equal(suite.T(), float64(384), embedResp["dimension"])
	assert.Equal(suite.T(), "這是一篇關於人工智慧和機器學習的測試文章。", embedResp["text"])
	
	embedding := embedResp["embedding"].([]interface{})
	assert.Equal(suite.T(), 384, len(embedding))
	
	// Verify embedding values are valid
	for i, val := range embedding {
		floatVal, ok := val.(float64)
		require.True(suite.T(), ok, "Embedding value at index %d should be a float", i)
		assert.False(suite.T(), math.IsNaN(floatVal), "Embedding value should not be NaN")
		assert.False(suite.T(), math.IsInf(floatVal, 0), "Embedding value should not be infinite")
	}
}

func (suite *EmbeddingTestSuite) TestBatchEmbedding() {
	embedData := map[string][]string{
		"texts": {
			"First article about technology and innovation.",
			"Second article about artificial intelligence.",
			"Third article about machine learning algorithms.",
		},
	}

	jsonData, _ := json.Marshal(embedData)
	resp, err := suite.client.Post(
		EmbeddingServiceBaseURL+"/embed/batch",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var embedResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&embedResp)
	require.NoError(suite.T(), err)

	// Verify batch embedding structure
	assert.Equal(suite.T(), float64(384), embedResp["dimension"])
	assert.Equal(suite.T(), float64(3), embedResp["count"])
	
	embeddings := embedResp["embeddings"].([]interface{})
	assert.Equal(suite.T(), 3, len(embeddings))
	
	// Verify each embedding
	for i, emb := range embeddings {
		embedding := emb.([]interface{})
		assert.Equal(suite.T(), 384, len(embedding))
		
		// Verify first few values of each embedding
		for j := 0; j < 5; j++ {
			floatVal, ok := embedding[j].(float64)
			require.True(suite.T(), ok, "Embedding %d value at index %d should be a float", i, j)
			assert.False(suite.T(), math.IsNaN(floatVal), "Embedding value should not be NaN")
			assert.False(suite.T(), math.IsInf(floatVal, 0), "Embedding value should not be infinite")
		}
	}
}

func (suite *EmbeddingTestSuite) TestContentClassification() {
	classifyData := map[string]string{
		"text": "This is a well-written article about machine learning and artificial intelligence technologies with detailed explanations and examples.",
	}

	jsonData, _ := json.Marshal(classifyData)
	resp, err := suite.client.Post(
		EmbeddingServiceBaseURL+"/classify",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var classifyResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&classifyResp)
	require.NoError(suite.T(), err)

	// Verify classification result structure
	assert.NotNil(suite.T(), classifyResp["is_article"])
	assert.NotNil(suite.T(), classifyResp["confidence"])
	assert.NotNil(suite.T(), classifyResp["text"])
	
	// Verify types and ranges
	isArticle, ok := classifyResp["is_article"].(bool)
	require.True(suite.T(), ok, "is_article should be a boolean")
	_ = isArticle // Use the variable
	
	confidence, ok := classifyResp["confidence"].(float64)
	require.True(suite.T(), ok, "Confidence should be a float")
	assert.True(suite.T(), confidence >= 0.0 && confidence <= 1.0, "Confidence should be between 0 and 1")
}

func (suite *EmbeddingTestSuite) TestClassifyBatchContent() {
	batchClassifyData := map[string][]string{
		"texts": {
			"This is a high-quality article with great content.",
			"Poor quality text with no meaningful information.",
			"Average article with some useful details.",
		},
	}

	jsonData, _ := json.Marshal(batchClassifyData)
	resp, err := suite.client.Post(
		EmbeddingServiceBaseURL+"/classify/batch",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var batchResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&batchResp)
	require.NoError(suite.T(), err)

	// Verify batch classification result
	assert.Equal(suite.T(), float64(3), batchResp["count"])
	
	results := batchResp["results"].([]interface{})
	assert.Equal(suite.T(), 3, len(results))
	
	// Verify each classification result
	for i, res := range results {
		result := res.(map[string]interface{})
		assert.NotNil(suite.T(), result["is_article"])
		assert.NotNil(suite.T(), result["confidence"])
		assert.NotNil(suite.T(), result["index"])
		assert.NotNil(suite.T(), result["text"])
		
		confidence, ok := result["confidence"].(float64)
		require.True(suite.T(), ok, "Confidence for result %d should be a float", i)
		assert.True(suite.T(), confidence >= 0.0 && confidence <= 1.0, "Confidence should be between 0 and 1")
		
		index, ok := result["index"].(float64)
		require.True(suite.T(), ok, "Index should be a number")
		assert.Equal(suite.T(), float64(i), index, "Index should match array position")
	}
}

func (suite *EmbeddingTestSuite) TestEmbeddingSimilarity() {
	// Generate embeddings for similar texts
	text1 := "Machine learning and artificial intelligence"
	text2 := "AI and ML technologies"
	text3 := "Cooking recipes and food preparation"
	
	var embedding1, embedding2, embedding3 []float64
	
	// Get first embedding
	embedData1 := map[string]string{"text": text1}
	jsonData1, _ := json.Marshal(embedData1)
	resp1, err := suite.client.Post(EmbeddingServiceBaseURL+"/embed", "application/json", bytes.NewBuffer(jsonData1))
	require.NoError(suite.T(), err)
	defer resp1.Body.Close()
	
	var embedResp1 map[string]interface{}
	json.NewDecoder(resp1.Body).Decode(&embedResp1)
	embeddingArr1 := embedResp1["embedding"].([]interface{})
	for _, val := range embeddingArr1 {
		embedding1 = append(embedding1, val.(float64))
	}
	
	// Get second embedding
	embedData2 := map[string]string{"text": text2}
	jsonData2, _ := json.Marshal(embedData2)
	resp2, err := suite.client.Post(EmbeddingServiceBaseURL+"/embed", "application/json", bytes.NewBuffer(jsonData2))
	require.NoError(suite.T(), err)
	defer resp2.Body.Close()
	
	var embedResp2 map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&embedResp2)
	embeddingArr2 := embedResp2["embedding"].([]interface{})
	for _, val := range embeddingArr2 {
		embedding2 = append(embedding2, val.(float64))
	}
	
	// Get third embedding (different topic)
	embedData3 := map[string]string{"text": text3}
	jsonData3, _ := json.Marshal(embedData3)
	resp3, err := suite.client.Post(EmbeddingServiceBaseURL+"/embed", "application/json", bytes.NewBuffer(jsonData3))
	require.NoError(suite.T(), err)
	defer resp3.Body.Close()
	
	var embedResp3 map[string]interface{}
	json.NewDecoder(resp3.Body).Decode(&embedResp3)
	embeddingArr3 := embedResp3["embedding"].([]interface{})
	for _, val := range embeddingArr3 {
		embedding3 = append(embedding3, val.(float64))
	}
	
	// Calculate cosine similarities
	sim12 := suite.cosineSimilarity(embedding1, embedding2)
	sim13 := suite.cosineSimilarity(embedding1, embedding3)
	sim23 := suite.cosineSimilarity(embedding2, embedding3)
	
	suite.T().Logf("Similarity between '%s' and '%s': %.4f", text1, text2, sim12)
	suite.T().Logf("Similarity between '%s' and '%s': %.4f", text1, text3, sim13)
	suite.T().Logf("Similarity between '%s' and '%s': %.4f", text2, text3, sim23)
	
	// Similar topics (AI/ML) should have higher similarity than different topics
	assert.Greater(suite.T(), sim12, sim13, "AI/ML texts should be more similar to each other than to cooking text")
	assert.Greater(suite.T(), sim12, sim23, "AI/ML texts should be more similar to each other than cooking is to either")
}

func (suite *EmbeddingTestSuite) cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	
	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / (normA * normB)
}

func (suite *EmbeddingTestSuite) TestInvalidRequests() {
	// Test empty text embedding
	embedData := map[string]string{"text": ""}
	jsonData, _ := json.Marshal(embedData)
	resp, err := suite.client.Post(EmbeddingServiceBaseURL+"/embed", "application/json", bytes.NewBuffer(jsonData))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Should handle empty text gracefully
	assert.True(suite.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest)
	
	// Test malformed JSON
	resp2, err := suite.client.Post(EmbeddingServiceBaseURL+"/embed", "application/json", bytes.NewBuffer([]byte("invalid json")))
	require.NoError(suite.T(), err)
	defer resp2.Body.Close()
	
	// Accept either 400 or 500 for malformed JSON (implementation dependent)
	assert.True(suite.T(), resp2.StatusCode == http.StatusBadRequest || resp2.StatusCode == http.StatusInternalServerError,
		"Expected 400 or 500 for malformed JSON, got %d", resp2.StatusCode)
	
	// Test missing text field
	invalidData := map[string]string{"not_text": "some content"}
	jsonData2, _ := json.Marshal(invalidData)
	resp3, err := suite.client.Post(EmbeddingServiceBaseURL+"/embed", "application/json", bytes.NewBuffer(jsonData2))
	require.NoError(suite.T(), err)
	defer resp3.Body.Close()
	
	assert.Equal(suite.T(), http.StatusBadRequest, resp3.StatusCode)
}

func TestEmbeddingSuite(t *testing.T) {
	suite.Run(t, new(EmbeddingTestSuite))
}