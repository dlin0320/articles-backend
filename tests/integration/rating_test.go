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

type RatingTestSuite struct {
	suite.Suite
	client    *http.Client
	userEmail string
	userToken string
	articleID string
}

func (suite *RatingTestSuite) SetupSuite() {
	suite.client = &http.Client{Timeout: 30 * time.Second}
	suite.userEmail = fmt.Sprintf("rating-test-%d@example.com", time.Now().Unix())
	
	// Setup user and article for testing
	suite.createTestUser()
	suite.loginTestUser()
	suite.createTestArticle()
}

func (suite *RatingTestSuite) createTestUser() {
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

func (suite *RatingTestSuite) loginTestUser() {
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

func (suite *RatingTestSuite) createTestArticle() {
	articleData := map[string]string{
		"url": fmt.Sprintf("https://example.com/rating-test-article-%d", time.Now().UnixNano()),
	}

	jsonData, _ := json.Marshal(articleData)
	req, _ := http.NewRequest("POST", APIBaseURL+"/articles", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	require.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var article map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&article)
	require.NoError(suite.T(), err)
	suite.articleID = article["id"].(string)
}

func (suite *RatingTestSuite) TestCreateRating() {
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	ratingData := map[string]int{"score": 5}
	jsonData, _ := json.Marshal(ratingData)
	
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var rating map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&rating)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), suite.articleID, rating["article_id"])
	assert.Equal(suite.T(), float64(5), rating["score"])
	assert.NotEmpty(suite.T(), rating["user_id"])
	assert.NotEmpty(suite.T(), rating["created_at"])
	assert.NotEmpty(suite.T(), rating["updated_at"])
}

func (suite *RatingTestSuite) TestCreateRatingUnauthorized() {
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	ratingData := map[string]int{"score": 4}
	jsonData, _ := json.Marshal(ratingData)
	
	resp, err := suite.client.Post(
		fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (suite *RatingTestSuite) TestCreateInvalidRating() {
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	// Test invalid score (outside 1-5 range)
	ratingData := map[string]int{"score": 6}
	jsonData, _ := json.Marshal(ratingData)
	
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	// Test score of 0
	ratingData = map[string]int{"score": 0}
	jsonData, _ = json.Marshal(ratingData)
	
	req, _ = http.NewRequest("POST", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
}

func (suite *RatingTestSuite) TestGetRating() {
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	// First create a rating
	ratingData := map[string]int{"score": 5}
	jsonData, _ := json.Marshal(ratingData)
	
	createReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), bytes.NewBuffer(jsonData))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+suite.userToken)

	createResp, err := suite.client.Do(createReq)
	require.NoError(suite.T(), err)
	defer createResp.Body.Close()
	require.Equal(suite.T(), http.StatusOK, createResp.StatusCode)

	// Now get the rating
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var rating map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&rating)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), suite.articleID, rating["article_id"])
	assert.Equal(suite.T(), float64(5), rating["score"])
	assert.NotEmpty(suite.T(), rating["user_id"])
}

func (suite *RatingTestSuite) TestGetNonexistentRating() {
	fakeArticleID := "00000000-0000-0000-0000-000000000000"
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, fakeArticleID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
}

func (suite *RatingTestSuite) TestUpdateRating() {
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	// Update the rating to a different score
	ratingData := map[string]int{"score": 3}
	jsonData, _ := json.Marshal(ratingData)
	
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode) // Should be OK for updates

	var rating map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&rating)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), suite.articleID, rating["article_id"])
	assert.Equal(suite.T(), float64(3), rating["score"])
	
	// Verify the rating was updated by retrieving it
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var updatedRating map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&updatedRating)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), float64(3), updatedRating["score"])
}

func (suite *RatingTestSuite) TestDeleteRating() {
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	// First create a rating to delete
	ratingData := map[string]int{"score": 4}
	jsonData, _ := json.Marshal(ratingData)
	
	createReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), bytes.NewBuffer(jsonData))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+suite.userToken)

	createResp, err := suite.client.Do(createReq)
	require.NoError(suite.T(), err)
	defer createResp.Body.Close()
	require.Equal(suite.T(), http.StatusOK, createResp.StatusCode)

	// Now delete the rating
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Verify the rating was deleted by trying to retrieve it
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
}

func (suite *RatingTestSuite) TestDeleteNonexistentRating() {
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

func (suite *RatingTestSuite) TestRatingUnauthorized() {
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	// Test GET without auth
	resp, err := suite.client.Get(fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

	// Test DELETE without auth
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), nil)
	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (suite *RatingTestSuite) TestRatingValidScoreRange() {
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	// Test all valid scores (1-5)
	validScores := []int{1, 2, 3, 4, 5}
	
	for _, score := range validScores {
		ratingData := map[string]int{"score": score}
		jsonData, _ := json.Marshal(ratingData)
		
		req, _ := http.NewRequest("POST", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+suite.userToken)

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)
		resp.Body.Close()

		// First rating should be 201 Created, subsequent ones should be 200 OK (updates)
		assert.True(suite.T(), resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
			"Score %d should be accepted (got status %d)", score, resp.StatusCode)
	}
}

func (suite *RatingTestSuite) TestCrossUserArticleAccessDenied() {
	// Test that users cannot rate articles they don't own (should get 404)
	if suite.articleID == "" {
		suite.T().Skip("No article ID available")
		return
	}

	// Create a second user
	secondUserEmail := fmt.Sprintf("rating-test-2-%d@example.com", time.Now().Unix())
	
	signupData := map[string]string{
		"email":    secondUserEmail,
		"password": "testpassword123",
	}
	jsonData, _ := json.Marshal(signupData)
	resp, err := suite.client.Post(APIBaseURL+"/signup", "application/json", bytes.NewBuffer(jsonData))
	require.NoError(suite.T(), err)
	resp.Body.Close()
	
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

	// First user successfully rates their own article
	ratingData1 := map[string]int{"score": 5}
	jsonData1, _ := json.Marshal(ratingData1)
	req1, _ := http.NewRequest("POST", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), bytes.NewBuffer(jsonData1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+suite.userToken)
	resp1, err := suite.client.Do(req1)
	require.NoError(suite.T(), err)
	resp1.Body.Close()
	assert.True(suite.T(), resp1.StatusCode == http.StatusCreated || resp1.StatusCode == http.StatusOK)

	// Second user attempts to rate first user's article - should be denied with 404
	ratingData2 := map[string]int{"score": 2}
	jsonData2, _ := json.Marshal(ratingData2)
	req2, _ := http.NewRequest("POST", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), bytes.NewBuffer(jsonData2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+secondUserToken)
	resp2, err := suite.client.Do(req2)
	require.NoError(suite.T(), err)
	resp2.Body.Close()
	
	// This should fail with 404 because users can only rate their own articles
	assert.Equal(suite.T(), http.StatusNotFound, resp2.StatusCode)

	// Verify first user can still read their own rating
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)
	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var rating1 map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&rating1)
	assert.Equal(suite.T(), float64(5), rating1["score"])

	// Verify second user cannot read the first user's rating (404)
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/articles/%s/rate", APIBaseURL, suite.articleID), nil)
	req.Header.Set("Authorization", "Bearer "+secondUserToken)
	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Should return 404 since second user cannot access first user's article
	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
}

func TestRatingSuite(t *testing.T) {
	suite.Run(t, new(RatingTestSuite))
}