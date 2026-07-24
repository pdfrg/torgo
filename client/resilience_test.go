package client

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockAdapter is a test implementation of ClientAdapter
type MockAdapter struct {
	connectErr  error
	listErr     error
	callCount   int
	failUntil   int // Fail first N calls, then succeed
	isConnected bool
	torrents    []Torrent
}

func (m *MockAdapter) Connect(ctx context.Context) error {
	if m.connectErr != nil {
		return m.connectErr
	}
	m.isConnected = true
	return nil
}

func (m *MockAdapter) Disconnect(ctx context.Context) error {
	m.isConnected = false
	return nil
}

func (m *MockAdapter) IsConnected() bool {
	return m.isConnected
}

func (m *MockAdapter) ListTorrents(ctx context.Context) ([]Torrent, error) {
	m.callCount++
	if m.callCount <= m.failUntil {
		return nil, errors.New("connection refused")
	}
	return m.torrents, m.listErr
}

func (m *MockAdapter) PauseTorrent(ctx context.Context, id string) error {
	return nil
}

func (m *MockAdapter) ResumeTorrent(ctx context.Context, id string) error {
	return nil
}

func (m *MockAdapter) RemoveTorrent(ctx context.Context, id string) error {
	return nil
}

func (m *MockAdapter) RemoveTorrentWithData(ctx context.Context, id string) error {
	return nil
}

func (m *MockAdapter) AddTorrent(ctx context.Context, input string, category string) error {
	return nil
}

func (m *MockAdapter) GetCategories(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockAdapter) PauseAll(ctx context.Context) error {
	return nil
}

func (m *MockAdapter) ResumeAll(ctx context.Context) error {
	return nil
}

func (m *MockAdapter) GetSpeedLimitEnabled(ctx context.Context) (bool, error) {
	return false, nil
}

func (m *MockAdapter) SetSpeedLimitEnabled(ctx context.Context, enabled bool) error {
	return nil
}

func (m *MockAdapter) GetSpeedLimits(ctx context.Context) (int, int, error) {
	return 0, 0, nil
}

func (m *MockAdapter) GetTorrentDetail(ctx context.Context, id string) (*TorrentDetail, error) {
	return nil, nil
}

func (m *MockAdapter) GetTorrentFiles(ctx context.Context, id string) ([]TorrentFile, error) {
	return nil, nil
}

func (m *MockAdapter) SetTorrentName(ctx context.Context, id string, newName string) error {
	return nil
}

func (m *MockAdapter) SetCategory(ctx context.Context, id string, category string) error {
	return nil
}

func (m *MockAdapter) SetTags(ctx context.Context, id string, tags []string) error {
	return nil
}

func (m *MockAdapter) SetSavePath(ctx context.Context, id string, path string) error {
	return nil
}

func (m *MockAdapter) SetFilePriorities(ctx context.Context, id string, fileIndices []int) error {
	return nil
}

func (m *MockAdapter) SetLabels(ctx context.Context, id string, labels []string) error {
	return nil
}

func (m *MockAdapter) RecheckTorrent(ctx context.Context, id string) error {
	return nil
}

func (m *MockAdapter) ReannounceTorrent(ctx context.Context, id string) error {
	return nil
}

func (m *MockAdapter) GetMagnetURI(ctx context.Context, id string) (string, error) {
	return "magnet:?xt=urn:btih:test", nil
}

func (m *MockAdapter) SetQueuePriority(ctx context.Context, id string, action string) error {
	return nil
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "deadline exceeded",
			err:       context.DeadlineExceeded,
			retryable: true,
		},
		{
			name:      "context cancelled",
			err:       context.Canceled,
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.retryable {
				t.Errorf("expected retryable=%v, got %v", tt.retryable, result)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		isAuth bool
	}{
		{
			name:   "nil error",
			err:    nil,
			isAuth: false,
		},
		{
			name:   "normal error",
			err:    errors.New("connection failed"),
			isAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.err)
			if result != tt.isAuth {
				t.Errorf("expected isAuth=%v, got %v", tt.isAuth, result)
			}
		})
	}
}

func TestResilientAdapterRetryOnFailure(t *testing.T) {
	mockAdapter := &MockAdapter{
		failUntil: 2, // Fail first 2 calls, succeed on 3rd
		torrents:  []Torrent{{ID: "1", Name: "Test"}},
	}

	config := ResilienceConfig{
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxRetries:        3,
		OperationTimeout:  1 * time.Second,
	}

	resilient := NewResilientAdapter(mockAdapter, config)
	ctx := context.Background()

	torrents, err := resilient.ListTorrents(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(torrents) != 1 || torrents[0].ID != "1" {
		t.Errorf("expected 1 torrent, got %d", len(torrents))
	}

	if mockAdapter.callCount != 3 {
		t.Errorf("expected 3 calls to adapter, got %d", mockAdapter.callCount)
	}
}

func TestResilientAdapterMaxRetriesExceeded(t *testing.T) {
	mockAdapter := &MockAdapter{
		failUntil: 10, // Always fail
		listErr:   errors.New("connection refused"),
	}

	config := ResilienceConfig{
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxRetries:        2,
		OperationTimeout:  1 * time.Second,
	}

	resilient := NewResilientAdapter(mockAdapter, config)
	ctx := context.Background()

	_, err := resilient.ListTorrents(ctx)

	if err == nil {
		t.Error("expected error when max retries exceeded")
	}

	// Should attempt: 1 (initial) + 2 (retries) = 3 times
	if mockAdapter.callCount != 3 {
		t.Errorf("expected 3 calls to adapter, got %d", mockAdapter.callCount)
	}
}

func TestHealthCheck(t *testing.T) {
	mockAdapter := &MockAdapter{
		isConnected: true,
		torrents:    []Torrent{},
	}

	config := DefaultResilienceConfig()
	config.HealthCheckInterval = 100 * time.Millisecond

	resilient := NewResilientAdapter(mockAdapter, config)
	ctx := context.Background()

	// First health check
	healthy := resilient.HealthCheck(ctx)
	if !healthy {
		t.Error("expected healthy connection")
	}

	callCount := mockAdapter.callCount

	// Immediate second check should use cache
	healthy = resilient.HealthCheck(ctx)
	if !healthy {
		t.Error("expected cached healthy result")
	}

	if mockAdapter.callCount != callCount {
		t.Error("expected cached result, should not call adapter again")
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third check should hit the adapter
	healthy = resilient.HealthCheck(ctx)
	if !healthy {
		t.Error("expected healthy connection")
	}

	if mockAdapter.callCount <= callCount {
		t.Error("expected adapter to be called after cache expiration")
	}
}

func TestIsConnectedStatus(t *testing.T) {
	mockAdapter := &MockAdapter{
		isConnected: true,
	}

	config := DefaultResilienceConfig()
	resilient := NewResilientAdapter(mockAdapter, config)

	if !resilient.IsConnected() {
		t.Error("expected IsConnected to return true")
	}

	// Simulate health check failure
	resilient.isHealthy = false

	if resilient.IsConnected() {
		t.Error("expected IsConnected to return false when unhealthy")
	}
}
