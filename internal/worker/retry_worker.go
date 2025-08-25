package worker

import (
	"fmt"
	"time"

	"github.com/dustin/articles-backend/config"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/robfig/cron/v3"
)

// RetryFunc defines the function signature for retry operations
type RetryFunc func() error

// RetryWorker runs scheduled retry operations with configurable intervals
type RetryWorker struct {
	name          string
	cron          *cron.Cron
	retryFunc     RetryFunc
	retryInterval time.Duration
	logger        *logger.Logger
	entryID       cron.EntryID
}

// NewRetryWorker creates a cron-scheduled worker with validation and defaults
func NewRetryWorker(cfg *config.WorkerConfig, name string, retryFunc RetryFunc, logger *logger.Logger) (*RetryWorker, error) {
	// Set defaults for nil or empty config values
	var retryInterval time.Duration = 5 * time.Minute
	if cfg != nil && cfg.RetryInterval != "" {
		duration, err := time.ParseDuration(cfg.RetryInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid retry interval '%s': %v", cfg.RetryInterval, err)
		}
		retryInterval = duration
	}

	return &RetryWorker{
		name:          name,
		cron:          cron.New(),
		retryFunc:     retryFunc,
		retryInterval: retryInterval,
		logger:        logger.WithComponent("retry-worker"),
	}, nil
}

// Start schedules and begins the retry worker
func (w *RetryWorker) Start() error {
	intervalStr := w.durationToCronExpression(w.retryInterval)
	w.logger.Info(fmt.Sprintf("Starting retry worker: %s (every %v)", w.name, w.retryInterval))

	entryID, err := w.cron.AddFunc(intervalStr, func() {
		w.logger.Debug("Executing retry operation for worker: " + w.name)

		if err := w.retryFunc(); err != nil {
			w.logger.Error("Retry operation failed for worker " + w.name + ": " + err.Error())
		} else {
			w.logger.Info("Retry operation completed successfully for worker: " + w.name)
		}
	})

	if err != nil {
		w.logger.Error("Failed to schedule retry worker " + w.name + ": " + err.Error())
		return err
	}

	w.entryID = entryID
	w.cron.Start()

	w.logger.Info("Retry worker started successfully: " + w.name)

	return nil
}

// Stop gracefully shuts down the retry worker
func (w *RetryWorker) Stop() error {
	w.logger.Info("Stopping retry worker: " + w.name)

	// Remove the scheduled entry
	if w.entryID > 0 {
		w.cron.Remove(w.entryID)
	}

	ctx := w.cron.Stop()
	<-ctx.Done() // Wait for graceful shutdown

	w.logger.Info("Retry worker stopped: " + w.name)

	return nil
}

// IsRunning checks if the worker has active cron entries
func (w *RetryWorker) IsRunning() bool {
	return len(w.cron.Entries()) > 0
}

// durationToCronExpression converts duration to cron format with fallback
func (w *RetryWorker) durationToCronExpression(duration time.Duration) string {
	minutes := int(duration.Minutes())
	hours := int(duration.Hours())

	if hours > 0 && minutes%60 == 0 {
		return fmt.Sprintf("0 */%d * * *", hours)
	} else if minutes > 0 && minutes < 60 {
		return fmt.Sprintf("*/%d * * * *", minutes)
	}

	// Fallback for unsupported durations
	w.logger.Warn(fmt.Sprintf("Unsupported retry interval %v, defaulting to 5 minutes", duration))
	return "*/5 * * * *"
}
