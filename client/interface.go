package client

import (
	"context"
)

// TorrentStatus represents the state of a torrent
type TorrentStatus string

const (
	StatusDownloading TorrentStatus = "downloading"
	StatusQueuedDL    TorrentStatus = "queuedDL"   // Queued for download
	StatusStalledDL   TorrentStatus = "stalledDL"  // Downloading but stalled (no peers)
	StatusPaused      TorrentStatus = "paused"
	StatusSeeding     TorrentStatus = "seeding"
	StatusCompleted   TorrentStatus = "completed"  // Fully downloaded, ready for removal
	StatusError       TorrentStatus = "error"
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

// TorrentFile represents a file in a torrent
type TorrentFile struct {
	Index      int
	Name       string // Relative path (e.g., "dir/subdir/file.txt")
	Size       int64  // File size in bytes
	Downloaded int64  // Bytes downloaded
	Priority   int    // 0=do not download, 1=normal, 6=high, 7=maximal (qBittorrent only)
}

// TorrentDetail contains detailed information about a torrent
type TorrentDetail struct {
	ID           string         // Torrent ID/Hash
	Name         string         // Torrent name
	Category     string         // Category/Label
	Tags         []string       // Tags (qBittorrent only)
	Comments     string         // Torrent comments (from metadata)
	SavePath     string         // Download location
	Files        []TorrentFile  // All files in torrent
	TotalSize    int64          // Total size of all files
	Downloaded   int64          // Total downloaded bytes
	ContentPath  string         // Actual content path (qBittorrent)
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

	// AddTorrent adds a torrent from a magnet link, URL, or file path with optional category/label
	// input: magnet:// URI, http(s):// URL, or path to .torrent file
	// category: optional category/label name (empty string to skip)
	AddTorrent(ctx context.Context, input string, category string) error

	// GetCategories returns available categories/labels for organizing torrents
	GetCategories(ctx context.Context) ([]string, error)

	// PauseAll pauses all torrents
	PauseAll(ctx context.Context) error

	// ResumeAll resumes all paused torrents
	ResumeAll(ctx context.Context) error

	// GetSpeedLimitEnabled returns whether the speed limit is currently enabled
	GetSpeedLimitEnabled(ctx context.Context) (bool, error)

	// SetSpeedLimitEnabled enables or disables the speed limit
	SetSpeedLimitEnabled(ctx context.Context, enabled bool) error

	// GetSpeedLimits returns the download and upload speed limits (KB/s), or 0 if not set
	GetSpeedLimits(ctx context.Context) (downKBs, upKBs int, err error)

	// GetTorrentDetail returns detailed information about a specific torrent
	GetTorrentDetail(ctx context.Context, id string) (*TorrentDetail, error)

	// GetTorrentFiles returns the list of files in a torrent
	GetTorrentFiles(ctx context.Context, id string) ([]TorrentFile, error)

	// SetTorrentName renames a torrent (qBittorrent only, returns error for Transmission)
	SetTorrentName(ctx context.Context, id string, newName string) error

	// SetCategory changes the category/label of a torrent
	SetCategory(ctx context.Context, id string, category string) error

	// SetTags updates the tags for a torrent (qBittorrent only, no-op for Transmission)
	SetTags(ctx context.Context, id string, tags []string) error

	// SetSavePath changes the save/download location for a torrent
	SetSavePath(ctx context.Context, id string, path string) error
}
