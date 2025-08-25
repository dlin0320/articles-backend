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

type ArticleTestSuite struct {
	suite.Suite
	client    *http.Client
	userEmail string
	userToken string
	articleID string
}

func (suite *ArticleTestSuite) SetupSuite() {
	suite.client = &http.Client{Timeout: 30 * time.Second}
	suite.userEmail = fmt.Sprintf("article-test-%d@example.com", time.Now().Unix())
	
	// Setup user for testing
	suite.createTestUser()
	suite.loginTestUser()
}

func (suite *ArticleTestSuite) createTestUser() {
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

func (suite *ArticleTestSuite) loginTestUser() {
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

func (suite *ArticleTestSuite) TestCreateArticle() {
	// Use unique URL to avoid constraint violations
	articleData := map[string]string{
		"url": fmt.Sprintf("https://example.com/test-article-%d", time.Now().UnixNano()),
	}

	jsonData, _ := json.Marshal(articleData)
	req, _ := http.NewRequest("POST", APIBaseURL+"/articles", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var article map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&article)
	require.NoError(suite.T(), err)

	assert.NotEmpty(suite.T(), article["id"])
	assert.Equal(suite.T(), articleData["url"], article["url"])
	assert.Equal(suite.T(), "pending", article["metadata_status"])
	
	suite.articleID = article["id"].(string)
}

func (suite *ArticleTestSuite) TestCreateArticleUnauthorized() {
	articleData := map[string]string{
		"url": "https://example.com/article",
	}

	jsonData, _ := json.Marshal(articleData)
	resp, err := suite.client.Post(
		APIBaseURL+"/articles",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (suite *ArticleTestSuite) TestCreateArticleInvalidURL() {
	articleData := map[string]string{
		"url": "not-a-valid-url",
	}

	jsonData, _ := json.Marshal(articleData)
	req, _ := http.NewRequest("POST", APIBaseURL+"/articles", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
}

func (suite *ArticleTestSuite) TestListArticles() {
	// First create an article to ensure we have data to list
	articleData := map[string]string{"url": fmt.Sprintf("https://example.com/list-test-%d", time.Now().UnixNano())}
	jsonData, _ := json.Marshal(articleData)
	req, _ := http.NewRequest("POST", APIBaseURL+"/articles", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)
	
	createResp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer createResp.Body.Close()
	require.Equal(suite.T(), http.StatusCreated, createResp.StatusCode)

	// Now list articles
	req, _ = http.NewRequest("GET", APIBaseURL+"/articles?limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	articles := response["articles"].([]interface{})
	assert.True(suite.T(), len(articles) >= 1, "Should have at least one article")
	
	// Check pagination metadata
	assert.NotNil(suite.T(), response["total"])
	assert.NotNil(suite.T(), response["page"])
	assert.NotNil(suite.T(), response["limit"])
	assert.NotNil(suite.T(), response["pages"])
}

func (suite *ArticleTestSuite) TestListArticlesUnauthorized() {
	resp, err := suite.client.Get(APIBaseURL + "/articles")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (suite *ArticleTestSuite) TestListArticlesPagination() {
	req, _ := http.NewRequest("GET", APIBaseURL+"/articles?page=1&limit=2", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	articles := response["articles"].([]interface{})
	assert.True(suite.T(), len(articles) <= 2)
	assert.Equal(suite.T(), float64(1), response["page"])
	assert.Equal(suite.T(), float64(2), response["limit"])
}

func (suite *ArticleTestSuite) TestArticleStructure() {
	// Simple test to verify article creation returns proper structure
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	req, _ := http.NewRequest("GET", APIBaseURL+"/articles", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	articles := response["articles"].([]interface{})
	if len(articles) > 0 {
		article := articles[0].(map[string]interface{})
		// Verify basic article structure without timing dependencies
		assert.NotNil(suite.T(), article["id"])
		assert.NotNil(suite.T(), article["url"])
		assert.NotNil(suite.T(), article["created_at"])
		assert.NotNil(suite.T(), article["user_id"])
		// metadata_status can be pending, processing, success, or failed - all are valid
		if status, ok := article["metadata_status"].(string); ok {
			assert.Contains(suite.T(), []string{"pending", "processing", "success", "failed"}, status)
		}
		suite.T().Logf("✅ Article structure validated: ID=%v, URL=%v, Status=%v", 
			article["id"], article["url"], article["metadata_status"])
	}
}

func (suite *ArticleTestSuite) TestDeleteArticle() {
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	req, _ := http.NewRequest("DELETE", APIBaseURL+"/articles/"+suite.articleID, nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Verify article is deleted
	req, _ = http.NewRequest("GET", APIBaseURL+"/articles", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	articles := response["articles"].([]interface{})
	for _, art := range articles {
		article := art.(map[string]interface{})
		assert.NotEqual(suite.T(), suite.articleID, article["id"])
	}
}

func (suite *ArticleTestSuite) TestDeleteNonexistentArticle() {
	fakeID := "00000000-0000-0000-0000-000000000000"
	req, _ := http.NewRequest("DELETE", APIBaseURL+"/articles/"+fakeID, nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
}

func (suite *ArticleTestSuite) TestDeleteArticleUnauthorized() {
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	req, _ := http.NewRequest("DELETE", APIBaseURL+"/articles/"+suite.articleID, nil)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func TestArticleSuite(t *testing.T) {
	suite.Run(t, new(ArticleTestSuite))
}