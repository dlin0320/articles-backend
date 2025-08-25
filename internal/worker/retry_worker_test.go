package worker

import (
	"testing"
	"time"

	"github.com/dustin/articles-backend/config"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRetryWorker(t *testing.T) {
	mockFunc := func() error { return nil }
	logCfg := &config.LoggingConfig{
		Level:       "info",
		Format:      "console",
		ServiceName: "test-worker",
	}
	log, err := logger.NewLogger(logCfg)
	require.NoError(t, err)

	workerCfg := config.WorkerConfig{
		RetryInterval: "5m",
	}

	worker, err := NewRetryWorker(&workerCfg, "test-worker", mockFunc, log)

	assert.NoError(t, err)
	assert.NotNil(t, worker)
	assert.Equal(t, "test-worker", worker.name)
	assert.NotNil(t, worker.cron)
	assert.NotNil(t, worker.retryFunc)
	assert.Equal(t, 5*time.Minute, worker.retryInterval)
	assert.NotNil(t, worker.logger)
}

func TestRetryWorker_Start_Stop(t *testing.T) {
	callCount := 0
	mockFunc := func() error {
		callCount++
		return nil
	}
	logCfg := &config.LoggingConfig{
		Level:       "info",
		Format:      "console",
		ServiceName: "test-worker",
	}
	log, err := logger.NewLogger(logCfg)
	require.NoError(t, err)

	workerCfg := config.WorkerConfig{RetryInterval: "5m"}
	worker, err := NewRetryWorker(&workerCfg, "test-worker", mockFunc, log)
	require.NoError(t, err)

	// Start the worker
	err = worker.Start()
	assert.NoError(t, err)

	// Verify it's running
	assert.True(t, worker.IsRunning())

	// Stop the worker
	err = worker.Stop()
	assert.NoError(t, err)

	// Verify it's stopped
	assert.False(t, worker.IsRunning())
}

func TestRetryWorker_IsRunning(t *testing.T) {
	mockFunc := func() error { return nil }
	logCfg := &config.LoggingConfig{
		Level:       "info",
		Format:      "console",
		ServiceName: "test-worker",
	}
	log, err := logger.NewLogger(logCfg)
	require.NoError(t, err)

	workerCfg := config.WorkerConfig{RetryInterval: "5m"}
	worker, err := NewRetryWorker(&workerCfg, "test-worker", mockFunc, log)
	require.NoError(t, err)

	// Initially not running
	assert.False(t, worker.IsRunning())

	// Start and check
	err = worker.Start()
	assert.NoError(t, err)
	assert.True(t, worker.IsRunning())

	// Stop and check
	err = worker.Stop()
	assert.NoError(t, err)
	assert.False(t, worker.IsRunning())
}

func TestRetryWorker_InvalidConfig(t *testing.T) {
	mockFunc := func() error { return nil }
	logCfg := &config.LoggingConfig{
		Level:       "info",
		Format:      "console",
		ServiceName: "test-worker",
	}
	log, err := logger.NewLogger(logCfg)
	require.NoError(t, err)

	// Test invalid retry interval
	workerCfg := config.WorkerConfig{
		RetryInterval: "invalid-duration",
	}

	_, err = NewRetryWorker(&workerCfg, "test-worker", mockFunc, log)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid retry interval")

	// Test valid config with retry interval
	workerCfg = config.WorkerConfig{
		RetryInterval: "5m",
	}

	worker, err := NewRetryWorker(&workerCfg, "test-worker", mockFunc, log)
	assert.NoError(t, err)
	assert.NotNil(t, worker)
	assert.Equal(t, 5*time.Minute, worker.retryInterval)
}

func TestRetryWorker_EmptyConfig(t *testing.T) {
	mockFunc := func() error { return nil }
	logCfg := &config.LoggingConfig{
		Level:       "info",
		Format:      "console",
		ServiceName: "test-worker",
	}
	log, err := logger.NewLogger(logCfg)
	require.NoError(t, err)

	// Test empty config uses defaults
	workerCfg := config.WorkerConfig{
		RetryInterval: "",
	}

	worker, err := NewRetryWorker(&workerCfg, "test-worker", mockFunc, log)

	assert.NoError(t, err)
	assert.NotNil(t, worker)
	assert.Equal(t, 5*time.Minute, worker.retryInterval)
}

func TestRetryFunc_Type(t *testing.T) {
	// Test that RetryFunc is correctly defined
	var fn RetryFunc = func() error { return nil }

	err := fn()
	assert.NoError(t, err)
}
