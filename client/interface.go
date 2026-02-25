package client

import (
	"context"
)

// TorrentStatus represents the state of a torrent
type TorrentStatus string

const (
	StatusDownloading TorrentStatus = "downloading"
	StatusSeeding     TorrentStatus = "seeding"
	StatusPaused      TorrentStatus = "paused"
	StatusError       TorrentStatus = "error"
	StatusQueued      TorrentStatus = "queued"
)

// Torrent represents a torrent in the client
type Torrent struct {
	ID        string
	Name      string
	Progress  uint8         // 0-100
	SpeedDown float64       // bytes/sec
	SpeedUp   float64       // bytes/sec
	Status    TorrentStatus
	Seeds     int
	Leechs    int
	Size      int64 // bytes
	Downloaded int64 // bytes
	Uploaded   int64 // bytes
}

// ClientAdapter is the interface all torrent clients must implement
type ClientAdapter interface {
	// Connect tests the connection to the client
	Connect(ctx context.Context) error

	// Disconnect closes the connection
	Disconnect(ctx context.Context) error

	// IsConnected returns whether the client is currently connected
	IsConnected() bool

	// ListTorrents returns all torrents from the client
	ListTorrents(ctx context.Context) ([]Torrent, error)

	// PauseTorrent pauses a single torrent by ID
	PauseTorrent(ctx context.Context, id string) error

	// ResumeTorrent resumes a single torrent by ID
	ResumeTorrent(ctx context.Context, id string) error

	// RemoveTorrent removes a torrent (without deleting files)
	RemoveTorrent(ctx context.Context, id string) error

	// RemoveTorrentWithData removes a torrent and deletes its files
	RemoveTorrentWithData(ctx context.Context, id string) error

	// AddTorrent adds a torrent from a magnet link or file path
	// magnetLink: magnet:// URI or path to .torrent file
	AddTorrent(ctx context.Context, magnetLink string) error

	// PauseAll pauses all torrents
	PauseAll(ctx context.Context) error

	// ResumeAll resumes all paused torrents
	ResumeAll(ctx context.Context) error

	// GetSpeedLimitEnabled returns whether the speed limit is currently enabled
	GetSpeedLimitEnabled(ctx context.Context) (bool, error)

	// SetSpeedLimitEnabled enables or disables the speed limit
	SetSpeedLimitEnabled(ctx context.Context, enabled bool) error
}
