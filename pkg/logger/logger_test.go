package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/dustin/articles-backend/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_BasicLogging(t *testing.T) {
	// Capture output for testing
	var buf bytes.Buffer
	testLogger := zerolog.New(&buf).Level(zerolog.DebugLevel)

	logger := &Logger{logger: testLogger}

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestLogger_LogLevelFiltering(t *testing.T) {
	// Test that log level filtering works
	var buf bytes.Buffer
	testLogger := zerolog.New(&buf).Level(zerolog.WarnLevel) // Only warn and above

	logger := &Logger{logger: testLogger}

	logger.Debug("debug message") // Should be filtered out
	logger.Info("info message")   // Should be filtered out
	logger.Warn("warn message")   // Should appear
	logger.Error("error message") // Should appear

	output := buf.String()

	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestNewLogger_ConsoleFormat(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:       "info",
		Format:      "console",
		ServiceName: "test-service",
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestNewLogger_JSONFormatWithDefaults(t *testing.T) {
	// Clean up any existing test logs directory
	testLogsDir := "./logs"
	defer os.RemoveAll(testLogsDir)

	cfg := &config.LoggingConfig{
		Level:       "", // Should default to "info"
		Format:      "", // Should default to "json"
		ServiceName: "", // Should default to "articles-backend"
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)
	assert.NotNil(t, logger)

	// Verify logs directory was created
	assert.DirExists(t, testLogsDir)
}

func TestNewLogger_InvalidLogLevel(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:       "invalid-level",
		Format:      "console",
		ServiceName: "test-service",
	}

	logger, err := NewLogger(cfg)
	assert.Error(t, err)
	assert.Nil(t, logger)
	assert.Contains(t, err.Error(), "invalid log level")
}

func TestNewLogger_FileLogging(t *testing.T) {
	testLogsDir := "./test-logs"
	defer os.RemoveAll(testLogsDir)

	// Temporarily change working directory for this test
	originalDir, _ := os.Getwd()
	tempDir, err := os.MkdirTemp("", "logger-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	cfg := &config.LoggingConfig{
		Level:       "debug",
		Format:      "json",
		ServiceName: "test-service",
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	// Log a test message
	logger.Info("test log message")

	// Check if log file was created
	logsDir := "./logs"
	assert.DirExists(t, logsDir)

	// Find log files
	files, err := filepath.Glob(filepath.Join(logsDir, "test-service-*.log"))
	require.NoError(t, err)
	assert.NotEmpty(t, files, "No log files found")

	// Read log file content
	if len(files) > 0 {
		content, err := os.ReadFile(files[0])
		require.NoError(t, err)
		assert.Contains(t, string(content), "test log message")
		assert.Contains(t, string(content), `"service":"test-service"`)
		assert.Contains(t, string(content), `"level":"info"`)
	}
}

func TestNewLogger_ConfigurationDefaults(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         *config.LoggingConfig
		expectError bool
	}{
		{
			name: "empty config uses defaults",
			cfg: &config.LoggingConfig{
				Level:       "",
				Format:      "",
				ServiceName: "",
			},
			expectError: false,
		},
		{
			name: "partial config with level only",
			cfg: &config.LoggingConfig{
				Level:       "debug",
				Format:      "",
				ServiceName: "",
			},
			expectError: false,
		},
		{
			name: "all fields provided",
			cfg: &config.LoggingConfig{
				Level:       "warn",
				Format:      "console",
				ServiceName: "custom-service",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up logs directory for each test
			defer os.RemoveAll("./logs")

			logger, err := NewLogger(tc.cfg)
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
			}
		})
	}
}

func TestLogger_WithComponent(t *testing.T) {
	var buf bytes.Buffer
	testLogger := zerolog.New(&buf).Level(zerolog.InfoLevel)

	logger := &Logger{logger: testLogger}
	componentLogger := logger.WithComponent("test-component")

	componentLogger.Info("component message")

	output := buf.String()
	assert.Contains(t, output, "component message")
	assert.Contains(t, output, `"component":"test-component"`)
}

func TestLogger_AllLogLevels(t *testing.T) {
	var buf bytes.Buffer
	testLogger := zerolog.New(&buf).Level(zerolog.DebugLevel)

	logger := &Logger{logger: testLogger}

	testCases := []struct {
		level   string
		message string
		logFunc func(string)
	}{
		{"debug", "debug test message", logger.Debug},
		{"info", "info test message", logger.Info},
		{"warn", "warn test message", logger.Warn},
		{"error", "error test message", logger.Error},
	}

	for _, tc := range testCases {
		t.Run(tc.level, func(t *testing.T) {
			buf.Reset()
			tc.logFunc(tc.message)

			output := buf.String()
			assert.Contains(t, output, tc.message)
			assert.Contains(t, output, `"level":"`+tc.level+`"`)
		})
	}
}

func TestLogger_TimestampAndService(t *testing.T) {
	testLogsDir := "./logs"
	defer os.RemoveAll(testLogsDir)

	cfg := &config.LoggingConfig{
		Level:       "info",
		Format:      "console",
		ServiceName: "timestamp-test-service",
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	var buf bytes.Buffer
	// Override the logger output for testing
	testLogger := zerolog.New(&buf).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Str("service", "timestamp-test-service").
		Logger()
	logger.logger = testLogger

	logger.Info("timestamp test message")

	output := buf.String()
	assert.Contains(t, output, "timestamp test message")
	assert.Contains(t, output, `"service":"timestamp-test-service"`)
	// Check for timestamp field (will be in RFC3339 format)
	assert.Contains(t, output, `"time":"`)
}
