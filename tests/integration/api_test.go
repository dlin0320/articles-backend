//go:build integration
// +build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServicesRunning verifies that all required services are accessible
func TestServicesRunning(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}
	
	t.Run("API Service Health", func(t *testing.T) {
		resp, err := client.Get(APIBaseURL + "/health")
		require.NoError(t, err, "API service should be accessible at %s", APIBaseURL)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "API health check should return 200")
		t.Logf("✅ API service is running at %s", APIBaseURL)
	})
	
	t.Run("Embedding Service Health", func(t *testing.T) {
		resp, err := client.Get(EmbeddingServiceBaseURL + "/health")
		require.NoError(t, err, "Embedding service should be accessible at %s", EmbeddingServiceBaseURL)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Embedding service health check should return 200")
		t.Logf("✅ Embedding service is running at %s", EmbeddingServiceBaseURL)
	})
}

// TestEndToEndFlow tests a complete user journey through the system
func TestEndToEndFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}
	
	client := &http.Client{Timeout: 60 * time.Second}
	userEmail := fmt.Sprintf("e2e-test-%d@example.com", time.Now().Unix())
	
	t.Log("🧪 Starting End-to-End Integration Test")
	t.Logf("👤 Test user: %s", userEmail)
	
	// This test will be implemented to run a full user journey
	// For now, just verify the services are ready
	t.Run("Services Ready", func(t *testing.T) {
		// API Health
		resp, err := client.Get(APIBaseURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		// Embedding Service Health  
		resp, err = client.Get(EmbeddingServiceBaseURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		t.Log("✅ All services are ready for end-to-end testing")
	})
}
