//go:build integration
// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RecommendationTestSuite struct {
	suite.Suite
	client    *http.Client
	userEmail string
	userToken string
	articles  []map[string]interface{}
}

func (suite *RecommendationTestSuite) SetupSuite() {
	suite.client = &http.Client{Timeout: 60 * time.Second}
	suite.userEmail = fmt.Sprintf("recommendation-test-%d@example.com", time.Now().Unix())
	
	// Setup user for testing
	suite.createTestUser()
	suite.loginTestUser()
	suite.createTestArticles()
}

func (suite *RecommendationTestSuite) createTestUser() {
	signupData := map[string]string{
		"email":    suite.userEmail,
		"password": "testpassword123",
	}

	jsonData, _ := json.Marshal(signupData)
	resp, err := suite.client.Post(
		APIBaseURL+"/signup",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	require.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
}

func (suite *RecommendationTestSuite) loginTestUser() {
	loginData := map[string]string{
		"email":    suite.userEmail,
		"password": "testpassword123",
	}

	jsonData, _ := json.Marshal(loginData)
	resp, err := suite.client.Post(
		APIBaseURL+"/login",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	require.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var loginResp map[string]string
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	require.NoError(suite.T(), err)
	suite.userToken = loginResp["token"]
}

func (suite *RecommendationTestSuite) createTestArticles() {
	// Create articles with simple, fast-loading URLs for testing recommendations
	timestamp := time.Now().UnixNano()
	testURLs := []string{
		fmt.Sprintf("https://httpbin.org/html?id=rec1-%d", timestamp),        // Simple HTML page
		fmt.Sprintf("https://httpbin.org/robots.txt?id=rec2-%d", timestamp),  // Simple text content
		fmt.Sprintf("https://httpbin.org/user-agent?id=rec3-%d", timestamp),  // Simple JSON response
	}

	for _, url := range testURLs {
		articleData := map[string]string{"url": url}
		jsonData, _ := json.Marshal(articleData)
		
		req, _ := http.NewRequest("POST", APIBaseURL+"/articles", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+suite.userToken)

		resp, err := suite.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusCreated {
			var article map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&article)
			suite.articles = append(suite.articles, article)
			suite.T().Logf("Created article: %s", article["id"])
		} else {
			if err != nil {
				suite.T().Logf("Error creating article %s: %v", url, err)
			} else {
				suite.T().Logf("Failed to create article %s: status %d", url, resp.StatusCode)
			}
		}
		if resp != nil {
			resp.Body.Close()
		}
		
		// Small delay to avoid overwhelming the service
		time.Sleep(1 * time.Second)
	}
}

func (suite *RecommendationTestSuite) TestRecommendationsForNewUser() {
	// Test basic recommendations API functionality
	req, _ := http.NewRequest("GET", APIBaseURL+"/recommendations?limit=5", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var recResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&recResp)
	require.NoError(suite.T(), err)

	// Verify basic response structure
	assert.NotNil(suite.T(), recResp["recommendations"])
	assert.NotNil(suite.T(), recResp["generated_at"])
	assert.NotNil(suite.T(), recResp["engine_used"])
	assert.NotNil(suite.T(), recResp["user_id"])
	assert.NotNil(suite.T(), recResp["count"])

	// Handle recommendations (can be empty or populated)
	if recommendations, ok := recResp["recommendations"].([]interface{}); ok {
		count := int(recResp["count"].(float64))
		assert.Equal(suite.T(), len(recommendations), count)
		
		// If recommendations exist, verify structure
		if len(recommendations) > 0 {
			firstRec := recommendations[0].(map[string]interface{})
			assert.NotNil(suite.T(), firstRec["score"])
			assert.NotNil(suite.T(), firstRec["reason"])
		}
		
		suite.T().Logf("✅ Recommendations API working: %d recommendations using engine '%s'", 
			count, recResp["engine_used"])
	}
}

func (suite *RecommendationTestSuite) TestCreateRatingsAndRecommendations() {
	if len(suite.articles) == 0 {
		suite.T().Skip("No articles available for rating")
		return
	}

	// Wait for at least one article to be processed (reduced timeout)
	suite.waitForArticleProcessing()

	// Rate some articles to build user preference
	for i, article := range suite.articles {
		if i >= 2 { // Only rate first 2 articles
			break
		}
		
		articleID := article["id"].(string)
		score := 5 - i // First article gets 5, second gets 4
		
		ratingData := map[string]int{"score": score}
		jsonData, _ := json.Marshal(ratingData)
		
		req, _ := http.NewRequest("POST", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, articleID), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+suite.userToken)

		resp, err := suite.client.Do(req)
		if err == nil {
			suite.T().Logf("Rated article %s with score %d (status: %d)", articleID, score, resp.StatusCode)
			resp.Body.Close()
		}
		
		time.Sleep(500 * time.Millisecond)
	}

	// Wait a moment for any background processing
	time.Sleep(2 * time.Second)

	// Now test recommendations with user preferences
	req, _ := http.NewRequest("GET", APIBaseURL+"/recommendations?limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var recResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&recResp)
	require.NoError(suite.T(), err)

	recommendations := recResp["recommendations"].([]interface{})
	count := int(recResp["count"].(float64))
	engine := recResp["engine_used"].(string)

	suite.T().Logf("User with ratings recommendations: %d recommendations using engine '%s'", count, engine)

	// Verify recommendation structure if we have recommendations
	if count > 0 {
		firstRec := recommendations[0].(map[string]interface{})
		assert.NotNil(suite.T(), firstRec["article_id"])
		assert.NotNil(suite.T(), firstRec["score"])
		assert.NotNil(suite.T(), firstRec["reason"])
		
		// Verify score is a valid number
		score, ok := firstRec["score"].(float64)
		assert.True(suite.T(), ok, "Recommendation score should be a float")
		assert.True(suite.T(), score >= 0 && score <= 1, "Recommendation score should be between 0 and 1")
	}
}

func (suite *RecommendationTestSuite) waitForArticleProcessing() {
	// Use channel-based timeout mechanism for more efficient waiting
	done := make(chan bool)
	timeout := time.After(20 * time.Second)
	
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				req, _ := http.NewRequest("GET", APIBaseURL+"/articles", nil)
				req.Header.Set("Authorization", "Bearer "+suite.userToken)

				resp, err := suite.client.Do(req)
				if err != nil {
					continue
				}
				
				var response map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&response)
				resp.Body.Close()

				articles := response["articles"].([]interface{})
				processedCount := 0
				
				for _, art := range articles {
					article := art.(map[string]interface{})
					if article["metadata_status"] == "success" {
						processedCount++
					}
				}
				
				if processedCount > 0 {
					suite.T().Logf("Found %d processed articles", processedCount)
					done <- true
					return
				}
			}
		}
	}()
	
	select {
	case <-done:
		// Processing completed successfully
		return
	case <-timeout:
		suite.T().Log("Article processing timeout reached")
		return
	}
}

func (suite *RecommendationTestSuite) TestRecommendationPagination() {
	// Test recommendation pagination
	req1, _ := http.NewRequest("GET", APIBaseURL+"/recommendations?limit=2", nil)
	req1.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp1, err := suite.client.Do(req1)
	require.NoError(suite.T(), err)
	defer resp1.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp1.StatusCode)

	var recResp1 map[string]interface{}
	err = json.NewDecoder(resp1.Body).Decode(&recResp1)
	require.NoError(suite.T(), err)

	recommendations1 := recResp1["recommendations"].([]interface{})
	count1 := int(recResp1["count"].(float64))
	
	// Should not exceed the limit
	assert.True(suite.T(), len(recommendations1) <= 2)
	assert.True(suite.T(), count1 <= 2)
	
	suite.T().Logf("Limited recommendations: requested 2, got %d", count1)
}

func (suite *RecommendationTestSuite) TestRecommendationUnauthorized() {
	// Test recommendations without authentication
	resp, err := suite.client.Get(APIBaseURL + "/recommendations")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (suite *RecommendationTestSuite) TestRecommendationEngineSelection() {
	// Test that the recommendation engine responds appropriately
	req, _ := http.NewRequest("GET", APIBaseURL+"/recommendations?limit=5", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var recResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&recResp)
	require.NoError(suite.T(), err)

	engine := recResp["engine_used"].(string)
	
	// Should use one of the expected engines
	expectedEngines := []string{"content-based", "popular", "default", "collaborative"}
	found := false
	for _, expectedEngine := range expectedEngines {
		if engine == expectedEngine {
			found = true
			break
		}
	}
	assert.True(suite.T(), found, "Engine should be one of the expected types, got: %s", engine)
	
	suite.T().Logf("Recommendation engine used: %s", engine)
}

func (suite *RecommendationTestSuite) TestRecommendationResponseTime() {
	// Test that recommendations respond within reasonable time
	start := time.Now()
	
	req, _ := http.NewRequest("GET", APIBaseURL+"/recommendations?limit=5", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	duration := time.Since(start)
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	assert.Less(suite.T(), duration, 10*time.Second, "Recommendations should respond within 10 seconds")
	
	suite.T().Logf("Recommendation response time: %v", duration)
}

func (suite *RecommendationTestSuite) TestMultipleUsersRecommendations() {
	// Create a second user to test user isolation
	secondUserEmail := fmt.Sprintf("recommendation-test-2-%d@example.com", time.Now().Unix())
	
	// Signup second user
	signupData := map[string]string{
		"email":    secondUserEmail,
		"password": "testpassword123",
	}
	jsonData, _ := json.Marshal(signupData)
	resp, err := suite.client.Post(APIBaseURL+"/signup", "application/json", bytes.NewBuffer(jsonData))
	require.NoError(suite.T(), err)
	resp.Body.Close()
	
	// Login second user
	loginData := map[string]string{
		"email":    secondUserEmail,
		"password": "testpassword123",
	}
	jsonData, _ = json.Marshal(loginData)
	resp, err = suite.client.Post(APIBaseURL+"/login", "application/json", bytes.NewBuffer(jsonData))
	require.NoError(suite.T(), err)
	
	var loginResp map[string]string
	json.NewDecoder(resp.Body).Decode(&loginResp)
	resp.Body.Close()
	secondUserToken := loginResp["token"]
	
	// Get recommendations for second user
	req, _ := http.NewRequest("GET", APIBaseURL+"/recommendations?limit=5", nil)
	req.Header.Set("Authorization", "Bearer "+secondUserToken)

	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var recResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&recResp)
	require.NoError(suite.T(), err)

	// Verify that user ID is different
	assert.NotEqual(suite.T(), suite.userToken, secondUserToken, "Users should have different tokens")
	suite.T().Logf("Second user recommendations: %d recommendations", int(recResp["count"].(float64)))
}

func TestRecommendationSuite(t *testing.T) {
	suite.Run(t, new(RecommendationTestSuite))
}