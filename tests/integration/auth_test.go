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

type AuthTestSuite struct {
	suite.Suite
	client    *http.Client
	userEmail string
	userToken string
}

func (suite *AuthTestSuite) SetupSuite() {
	suite.client = &http.Client{Timeout: 30 * time.Second}
	suite.userEmail = fmt.Sprintf("test-%d-%d@example.com", time.Now().Unix(), time.Now().UnixNano())
	
	// Create user for login tests
	suite.createTestUser()
	suite.loginTestUser()
}

func (suite *AuthTestSuite) createTestUser() {
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

func (suite *AuthTestSuite) loginTestUser() {
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

func (suite *AuthTestSuite) TestHealthCheck() {
	resp, err := suite.client.Get(APIBaseURL + "/health")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "healthy", health["status"])
	assert.NotEmpty(suite.T(), health["timestamp"])
}

func (suite *AuthTestSuite) TestUserSignup() {
	// Create a different user for this test since main user is created in setup
	testEmail := fmt.Sprintf("signup-test-%d-%d@example.com", time.Now().Unix(), time.Now().UnixNano())
	
	signupData := map[string]string{
		"email":    testEmail,
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

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var user map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&user)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), testEmail, user["email"])
	assert.NotEmpty(suite.T(), user["id"])
	assert.NotEmpty(suite.T(), user["created_at"])
}

func (suite *AuthTestSuite) TestUserLogin_AfterSignup() {
	// Test successful login (already done in setup, so just verify we have a token)
	assert.NotEmpty(suite.T(), suite.userToken)
	
	// Test login again to verify it works multiple times
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

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var loginResp map[string]string
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	require.NoError(suite.T(), err)

	assert.NotEmpty(suite.T(), loginResp["token"])
}

func (suite *AuthTestSuite) TestInvalidLogin() {
	loginData := map[string]string{
		"email":    "nonexistent@example.com",
		"password": "wrongpassword",
	}

	jsonData, _ := json.Marshal(loginData)
	resp, err := suite.client.Post(
		APIBaseURL+"/login",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (suite *AuthTestSuite) TestUserMe_AfterLogin() {
	req, _ := http.NewRequest("GET", APIBaseURL+"/me", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var user map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&user)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), suite.userEmail, user["email"])
	assert.NotEmpty(suite.T(), user["id"])
}

func (suite *AuthTestSuite) TestUnauthorizedAccess() {
	// Test accessing protected endpoint without token
	resp, err := suite.client.Get(APIBaseURL + "/me")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (suite *AuthTestSuite) TestInvalidToken() {
	req, _ := http.NewRequest("GET", APIBaseURL+"/me", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}