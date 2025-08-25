//go:build integration
// +build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite runs all integration tests in order
type IntegrationTestSuite struct {
	suite.Suite
	client *http.Client
}

func (suite *IntegrationTestSuite) SetupSuite() {
	suite.client = &http.Client{Timeout: 30 * time.Second}
	
	// Wait for services to be ready
	suite.waitForServices()
}

func (suite *IntegrationTestSuite) waitForServices() {
	maxRetries := 30
	retryDelay := 2 * time.Second
	
	suite.T().Log("Waiting for services to be ready...")
	
	// Wait for API service
	for i := 0; i < maxRetries; i++ {
		resp, err := suite.client.Get(APIBaseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			suite.T().Log("✅ Articles API service is ready")
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		if i == maxRetries-1 {
			suite.T().Fatal("❌ Articles API service is not ready after maximum retries")
		}
		time.Sleep(retryDelay)
	}
	
	// Wait for Embedding service
	for i := 0; i < maxRetries; i++ {
		resp, err := suite.client.Get(EmbeddingServiceBaseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			suite.T().Log("✅ Embedding service is ready")
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		if i == maxRetries-1 {
			suite.T().Fatal("❌ Embedding service is not ready after maximum retries")
		}
		time.Sleep(retryDelay)
	}
	
	suite.T().Log("🚀 All services are ready! Starting integration tests...")
}

func (suite *IntegrationTestSuite) TestServiceHealthChecks() {
	// Test API health
	resp, err := suite.client.Get(APIBaseURL + "/health")
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Equal(http.StatusOK, resp.StatusCode)
	
	// Test Embedding service health
	resp, err = suite.client.Get(EmbeddingServiceBaseURL + "/health")
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Equal(http.StatusOK, resp.StatusCode)
}

// TestMain runs all integration test suites
func TestIntegrationSuite(t *testing.T) {
	// Print test information
	fmt.Println("🧪 Running Articles Backend Integration Tests")
	fmt.Println("================================================")
	fmt.Printf("API URL: %s\n", APIBaseURL)
	fmt.Printf("Embedding Service URL: %s\n", EmbeddingServiceBaseURL)
	fmt.Println("================================================")
	
	// Run basic integration suite first
	suite.Run(t, new(IntegrationTestSuite))
	
	// Run all test suites
	fmt.Println("\n🔐 Running Authentication Tests...")
	suite.Run(t, new(AuthTestSuite))
	
	fmt.Println("\n📄 Running Article Management Tests...")
	suite.Run(t, new(ArticleTestSuite))
	
	fmt.Println("\n🌐 Running Multilingual Content Tests...")
	suite.Run(t, new(MultilingualTestSuite))
	
	fmt.Println("\n🤖 Running Embedding Service Tests...")
	suite.Run(t, new(EmbeddingTestSuite))
	
	fmt.Println("\n⭐ Running Rating System Tests...")
	suite.Run(t, new(RatingTestSuite))
	
	fmt.Println("\n💡 Running Recommendation System Tests...")
	suite.Run(t, new(RecommendationTestSuite))
	
	fmt.Println("\n✅ All integration tests completed!")
}