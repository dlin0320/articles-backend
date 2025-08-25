package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dustin/articles-backend/config"
	"github.com/rs/zerolog"
)

type Logger struct {
	logger zerolog.Logger
}

// NewLogger creates a structured logger with validation and defaults
func NewLogger(cfg *config.LoggingConfig) (*Logger, error) {
	// Set defaults for empty config values
	level := cfg.Level
	if level == "" {
		level = "info"
	}

	format := cfg.Format
	if format == "" {
		format = "json"
	}

	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "articles-backend"
	}

	// Validate log level early to fail fast
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level '%s': %v", level, err)
	}

	// Configure output based on format
	var output io.Writer
	if format == "console" {
		// Console format for development - human-readable to stdout
		output = zerolog.ConsoleWriter{
			Out:     os.Stdout,
			NoColor: false,
		}
	} else {
		// JSON format for production - write to both stdout and log file
		// Use project root for logs directory
		logDir := "./logs"
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %v", err)
		}

		logFile := fmt.Sprintf("%s/%s-%s.log", logDir, serviceName, time.Now().Format("2006-01-02"))
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %v", err)
		}

		// Write to both stdout and file
		output = io.MultiWriter(os.Stdout, file)
	}

	logger := zerolog.New(output).
		Level(logLevel).
		With().
		Timestamp().
		Str("service", serviceName).
		Logger()

	return &Logger{logger: logger}, nil
}

func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

func (l *Logger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

func (l *Logger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

func (l *Logger) Fatal(msg string) {
	l.logger.Fatal().Msg(msg)
}

// WithComponent returns a logger instance with component context
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("component", component).Logger(),
	}
}
