package client

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"time"
)

// ResilienceConfig defines the retry behavior
type ResilienceConfig struct {
	// Initial backoff duration
	InitialBackoff time.Duration
	// Maximum backoff duration
	MaxBackoff time.Duration
	// Backoff multiplier (exponential growth)
	BackoffMultiplier float64
	// Maximum number of retries
	MaxRetries int
	// Health check interval
	HealthCheckInterval time.Duration
	// Operation timeout
	OperationTimeout time.Duration
}

// DefaultResilienceConfig returns sensible defaults
func DefaultResilienceConfig() ResilienceConfig {
	return ResilienceConfig{
		InitialBackoff:      1 * time.Second,
		MaxBackoff:          30 * time.Second,
		BackoffMultiplier:   2.0,
		MaxRetries:          3,
		HealthCheckInterval: 10 * time.Second,
		OperationTimeout:    15 * time.Second,
	}
}

// ResilientAdapter wraps a ClientAdapter with retry logic and health checks
type ResilientAdapter struct {
	adapter   ClientAdapter
	config    ResilienceConfig
	lastError error
	isHealthy bool
	lastCheck time.Time
}

// NewResilientAdapter creates a new resilient adapter
func NewResilientAdapter(adapter ClientAdapter, config ResilienceConfig) *ResilientAdapter {
	return &ResilientAdapter{
		adapter:   adapter,
		config:    config,
		isHealthy: true,
		lastCheck: time.Time{}, // Zero time means never checked
	}
}

// IsRetryableError determines if an error should trigger a retry
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Context errors
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, context.Canceled) {
		return false
	}

	// Network-level errors that are retryable
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	// Check error message for common retryable patterns
	errStr := err.Error()
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"EOF",
		"i/o timeout",
		"no such host",
		"connection declined",
		"connection refused",
		"temporarily unavailable",
		"try again",
	}

	for _, pattern := range retryablePatterns {
		if containsSubstring(errStr, pattern) {
			return true
		}
	}

	return false
}

// IsAuthError determines if an error is authentication-related (non-retryable)
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	authStrings := []string{
		"unauthorized",
		"forbidden",
		"401",
		"403",
		"invalid credentials",
		"authentication failed",
	}

	for _, s := range authStrings {
		if containsSubstring(errStr, s) {
			return true
		}
	}

	return false
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// HealthCheck verifies if the adapter is connected
func (r *ResilientAdapter) HealthCheck(ctx context.Context) bool {
	// If we checked recently, return cached result
	if !r.lastCheck.IsZero() && time.Since(r.lastCheck) < r.config.HealthCheckInterval {
		return r.isHealthy
	}

	// Perform a real health check
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// List torrents is a simple operation that indicates if connected
	_, err := r.adapter.ListTorrents(ctx)

	r.lastCheck = time.Now()
	r.isHealthy = err == nil
	if err != nil {
		r.lastError = err
	}

	return r.isHealthy
}

// RetryWithBackoff executes a function with exponential backoff on retryable errors
func (r *ResilientAdapter) RetryWithBackoff(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Create a timeout for this operation
		opCtx, cancel := context.WithTimeout(ctx, r.config.OperationTimeout)
		err := fn(opCtx)
		cancel()

		if err == nil {
			// Success
			r.isHealthy = true
			r.lastError = nil
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryableError(err) {
			// Non-retryable error, return immediately
			if IsAuthError(err) {
				r.isHealthy = false
			}
			return err
		}

		// If this was the last attempt, return the error
		if attempt >= r.config.MaxRetries {
			r.isHealthy = false
			r.lastError = err
			return fmt.Errorf("operation failed after %d retries: %w", r.config.MaxRetries, err)
		}

		// Calculate backoff with exponential increase and jitter
		backoff := time.Duration(float64(r.config.InitialBackoff) * math.Pow(r.config.BackoffMultiplier, float64(attempt)))
		if backoff > r.config.MaxBackoff {
			backoff = r.config.MaxBackoff
		}

		// Add jitter (±10%)
		jitterMax := int64(backoff / 10)
		var jitter time.Duration
		if jitterMax > 0 {
			jitter = time.Duration(rand.Int63n(jitterMax))
		}
		if rand.Intn(2) == 0 {
			backoff -= jitter
		} else {
			backoff += jitter
		}

		// Wait before retrying
		select {
		case <-time.After(backoff):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return lastErr
}

// Wrapped methods

func (r *ResilientAdapter) Connect(ctx context.Context) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.Connect(opCtx)
	})
}

func (r *ResilientAdapter) Disconnect(ctx context.Context) error {
	return r.adapter.Disconnect(ctx)
}

func (r *ResilientAdapter) IsConnected() bool {
	return r.adapter.IsConnected() && r.isHealthy
}

func (r *ResilientAdapter) ListTorrents(ctx context.Context) ([]Torrent, error) {
	var result []Torrent
	err := r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		torrents, err := r.adapter.ListTorrents(opCtx)
		if err == nil {
			result = torrents
		}
		return err
	})
	return result, err
}

func (r *ResilientAdapter) PauseTorrent(ctx context.Context, id string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.PauseTorrent(opCtx, id)
	})
}

func (r *ResilientAdapter) ResumeTorrent(ctx context.Context, id string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.ResumeTorrent(opCtx, id)
	})
}

func (r *ResilientAdapter) RemoveTorrent(ctx context.Context, id string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.RemoveTorrent(opCtx, id)
	})
}

func (r *ResilientAdapter) RemoveTorrentWithData(ctx context.Context, id string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.RemoveTorrentWithData(opCtx, id)
	})
}

func (r *ResilientAdapter) AddTorrent(ctx context.Context, input string, category string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.AddTorrent(opCtx, input, category)
	})
}

func (r *ResilientAdapter) GetCategories(ctx context.Context) ([]string, error) {
	var result []string
	err := r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		categories, err := r.adapter.GetCategories(opCtx)
		if err == nil {
			result = categories
		}
		return err
	})
	return result, err
}

func (r *ResilientAdapter) PauseAll(ctx context.Context) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.PauseAll(opCtx)
	})
}

func (r *ResilientAdapter) ResumeAll(ctx context.Context) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.ResumeAll(opCtx)
	})
}

func (r *ResilientAdapter) GetSpeedLimitEnabled(ctx context.Context) (bool, error) {
	var result bool
	err := r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		enabled, err := r.adapter.GetSpeedLimitEnabled(opCtx)
		if err == nil {
			result = enabled
		}
		return err
	})
	return result, err
}

func (r *ResilientAdapter) SetSpeedLimitEnabled(ctx context.Context, enabled bool) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.SetSpeedLimitEnabled(opCtx, enabled)
	})
}

func (r *ResilientAdapter) GetSpeedLimits(ctx context.Context) (int, int, error) {
	var downKBs, upKBs int
	err := r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		down, up, err := r.adapter.GetSpeedLimits(opCtx)
		if err == nil {
			downKBs = down
			upKBs = up
		}
		return err
	})
	return downKBs, upKBs, err
}

func (r *ResilientAdapter) GetTorrentDetail(ctx context.Context, id string) (*TorrentDetail, error) {
	var result *TorrentDetail
	err := r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		detail, err := r.adapter.GetTorrentDetail(opCtx, id)
		if err == nil {
			result = detail
		}
		return err
	})
	return result, err
}

func (r *ResilientAdapter) GetTorrentFiles(ctx context.Context, id string) ([]TorrentFile, error) {
	var result []TorrentFile
	err := r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		files, err := r.adapter.GetTorrentFiles(opCtx, id)
		if err == nil {
			result = files
		}
		return err
	})
	return result, err
}

func (r *ResilientAdapter) SetTorrentName(ctx context.Context, id string, newName string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.SetTorrentName(opCtx, id, newName)
	})
}

func (r *ResilientAdapter) SetCategory(ctx context.Context, id string, category string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.SetCategory(opCtx, id, category)
	})
}

func (r *ResilientAdapter) SetTags(ctx context.Context, id string, tags []string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.SetTags(opCtx, id, tags)
	})
}

func (r *ResilientAdapter) SetSavePath(ctx context.Context, id string, path string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.SetSavePath(opCtx, id, path)
	})
}

func (r *ResilientAdapter) SetFilePriorities(ctx context.Context, id string, fileIndices []int) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.SetFilePriorities(opCtx, id, fileIndices)
	})
}

func (r *ResilientAdapter) SetLabels(ctx context.Context, id string, labels []string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.SetLabels(opCtx, id, labels)
	})
}

func (r *ResilientAdapter) RecheckTorrent(ctx context.Context, id string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.RecheckTorrent(opCtx, id)
	})
}

func (r *ResilientAdapter) ReannounceTorrent(ctx context.Context, id string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.ReannounceTorrent(opCtx, id)
	})
}

func (r *ResilientAdapter) GetMagnetURI(ctx context.Context, id string) (string, error) {
	var result string
	err := r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		uri, err := r.adapter.GetMagnetURI(opCtx, id)
		if err == nil {
			result = uri
		}
		return err
	})
	return result, err
}

func (r *ResilientAdapter) SetQueuePriority(ctx context.Context, id string, action string) error {
	return r.RetryWithBackoff(ctx, func(opCtx context.Context) error {
		return r.adapter.SetQueuePriority(opCtx, id, action)
	})
}

// LastError returns the last error that occurred
func (r *ResilientAdapter) LastError() error {
	return r.lastError
}
