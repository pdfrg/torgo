package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// QBittorrentAdapter implements ClientAdapter for qBittorrent
type QBittorrentAdapter struct {
	host      string
	port      int
	username  string
	password  string
	connected bool
	client    *http.Client
}

// qbTorrent represents the qBittorrent API torrent response
type qbTorrent struct {
	Hash        string  `json:"hash"`
	Name        string  `json:"name"`
	Progress    float64 `json:"progress"` // 0-1
	DlSpeed     float64 `json:"dlspeed"`  // bytes/sec
	UpSpeed     float64 `json:"upspeed"`  // bytes/sec
	State       string  `json:"state"`    // uploading, downloading, etc.
	NumSeeds    int     `json:"num_seeds"`
	NumLeechs   int     `json:"num_leechs"`
	TotalSize   int64   `json:"total_size"`
	Downloaded  int64   `json:"downloaded"`
	Uploaded    int64   `json:"uploaded"`
	Category    string  `json:"category"`
	Tags        string  `json:"tags"` // Comma-separated
	Comment     string  `json:"comment"`
	SavePath    string  `json:"save_path"`
	ContentPath string  `json:"content_path"`
	ETA         int64   `json:"eta"` // seconds (8640000 = infinite)
	MagnetURI   string  `json:"magnet_uri"`
}

// qbFile represents a file in a torrent (from /api/v2/torrents/files)
type qbFile struct {
	Index    int     `json:"index"`
	Name     string  `json:"name"`
	Size     int64   `json:"size"`
	Progress float64 `json:"progress"` // 0-1
	Priority int     `json:"priority"`
}

// NewQBittorrentAdapter creates a new qBittorrent adapter
func NewQBittorrentAdapter(host string, port int, username, password string) *QBittorrentAdapter {
	jar, _ := cookiejar.New(nil)
	qa := &QBittorrentAdapter{
		host:     host,
		port:     port,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Jar:     jar,
		},
	}
	return qa
}

// Connect authenticates with the qBittorrent instance
func (qa *QBittorrentAdapter) Connect(ctx context.Context) error {
	loginURL := qa.getBaseURL() + "/api/v2/auth/login"
	formData := url.Values{}
	formData.Set("username", qa.username)
	formData.Set("password", qa.password)

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	bodyStr := strings.TrimSpace(string(body))

	switch resp.StatusCode {
	case http.StatusOK:
		// qBittorrent returns "Ok." on success and "Fails." on failure (both with HTTP 200)
		if bodyStr != "Ok." {
			return fmt.Errorf("login failed: %s", bodyStr)
		}
	case http.StatusNoContent:
		// qBittorrent 5.x returns 204 No Content on success (no body)
	default:
		return fmt.Errorf("login failed: %s (status %d)", bodyStr, resp.StatusCode)
	}

	qa.connected = true
	return nil
}

// Disconnect closes the connection
func (qa *QBittorrentAdapter) Disconnect(ctx context.Context) error {
	qa.connected = false
	return nil
}

// IsConnected returns whether the adapter is connected
func (qa *QBittorrentAdapter) IsConnected() bool {
	return qa.connected
}

// ListTorrents fetches all torrents from qBittorrent
func (qa *QBittorrentAdapter) ListTorrents(ctx context.Context) ([]Torrent, error) {
	listURL := qa.getBaseURL() + "/api/v2/torrents/info"

	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list request: %w", err)
	}

	resp, err := qa.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("list torrents failed: status %d", resp.StatusCode)
	}

	var qbTorrents []qbTorrent
	if err := json.NewDecoder(resp.Body).Decode(&qbTorrents); err != nil {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("failed to decode torrent list: %w", err)
	}
	_ = resp.Body.Close()

	torrents := make([]Torrent, len(qbTorrents))
	for i, qb := range qbTorrents {
		torrents[i] = qa.mapTorrent(qb)
	}
	return torrents, nil
}

// PauseTorrent pauses a single torrent
func (qa *QBittorrentAdapter) PauseTorrent(ctx context.Context, id string) error {
	return qa.pauseResume(ctx, "pause", id)
}

// ResumeTorrent resumes a single torrent
func (qa *QBittorrentAdapter) ResumeTorrent(ctx context.Context, id string) error {
	return qa.pauseResume(ctx, "resume", id)
}

// RemoveTorrent removes a torrent without deleting files
func (qa *QBittorrentAdapter) RemoveTorrent(ctx context.Context, id string) error {
	return qa.delete(ctx, id, false)
}

// RemoveTorrentWithData removes a torrent and deletes files
func (qa *QBittorrentAdapter) RemoveTorrentWithData(ctx context.Context, id string) error {
	return qa.delete(ctx, id, true)
}

// AddTorrent adds a torrent from magnet link, URL, or .torrent file with optional category
func (qa *QBittorrentAdapter) AddTorrent(ctx context.Context, input string, category string) error {
	addURL := qa.getBaseURL() + "/api/v2/torrents/add"

	// Check if it's a magnet link or URL
	if strings.HasPrefix(input, "magnet:") || strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		// Add magnet link or URL
		formData := url.Values{}
		formData.Set("urls", input)
		if category != "" {
			formData.Set("category", category)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", addURL, strings.NewReader(formData.Encode()))
		if err != nil {
			return fmt.Errorf("failed to create add request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := qa.client.Do(req)
		if err != nil {
			return fmt.Errorf("add failed: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("add failed: status %d", resp.StatusCode)
		}
		return nil
	}

	// Handle file path
	file, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("failed to open torrent file: %w", err)
	}
	fileBytes, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		return fmt.Errorf("failed to read torrent file: %w", err)
	}

	formData := url.Values{}
	formData.Set("filedata", string(fileBytes))
	if category != "" {
		formData.Set("category", category)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", addURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create add request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("add torrent file failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("add torrent file failed: status %d", resp.StatusCode)
	}
	return nil
}

// GetCategories fetches available categories from qBittorrent
func (qa *QBittorrentAdapter) GetCategories(ctx context.Context) ([]string, error) {
	syncURL := qa.getBaseURL() + "/api/v2/sync/maindata"

	req, err := http.NewRequestWithContext(ctx, "GET", syncURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := qa.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch categories: status %d", resp.StatusCode)
	}

	var result struct {
		Categories map[string]interface{} `json:"categories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract category names and sort them
	categories := make([]string, 0, len(result.Categories))
	for name := range result.Categories {
		categories = append(categories, name)
	}

	sort.Strings(categories)
	return categories, nil
}

// PauseAll pauses all torrents
func (qa *QBittorrentAdapter) PauseAll(ctx context.Context) error {
	return qa.pauseResume(ctx, "pause", "all")
}

// ResumeAll resumes all torrents
func (qa *QBittorrentAdapter) ResumeAll(ctx context.Context) error {
	return qa.pauseResume(ctx, "resume", "all")
}

// Helper methods

func (qa *QBittorrentAdapter) getBaseURL() string {
	return fmt.Sprintf("http://%s:%d", qa.host, qa.port)
}

func (qa *QBittorrentAdapter) pauseResume(ctx context.Context, action, hash string) error {
	// qBittorrent v5+ uses "stop" instead of "pause"
	endpoint := action
	if action == "pause" {
		endpoint = "stop"
	}
	if action == "resume" {
		endpoint = "start"
	}

	actionURL := qa.getBaseURL() + fmt.Sprintf("/api/v2/torrents/%s", endpoint)
	formData := url.Values{}
	formData.Set("hashes", hash)

	req, err := http.NewRequestWithContext(ctx, "POST", actionURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create %s request: %w", action, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", action, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s failed: status %d", action, resp.StatusCode)
	}
	return nil
}

func (qa *QBittorrentAdapter) delete(ctx context.Context, hash string, deleteFiles bool) error {
	deleteURL := qa.getBaseURL() + "/api/v2/torrents/delete"
	formData := url.Values{}
	formData.Set("hashes", hash)
	formData.Set("deleteFiles", fmt.Sprintf("%v", deleteFiles))

	req, err := http.NewRequestWithContext(ctx, "POST", deleteURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete failed: status %d", resp.StatusCode)
	}
	return nil
}

func (qa *QBittorrentAdapter) mapTorrent(qb qbTorrent) Torrent {
	status := qa.mapStatus(qb.State)
	return Torrent{
		ID:         qb.Hash,
		Name:       qb.Name,
		Progress:   uint8(qb.Progress * 100),
		SpeedDown:  qb.DlSpeed,
		SpeedUp:    qb.UpSpeed,
		Status:     status,
		Seeds:      qb.NumSeeds,
		Leechs:     qb.NumLeechs,
		Size:       qb.TotalSize,
		Downloaded: qb.Downloaded,
		Uploaded:   qb.Uploaded,
		Category:   qb.Category,
		ETA:        qb.ETA,
		MagnetURI:  qb.MagnetURI,
	}
}

func (qa *QBittorrentAdapter) mapStatus(qbState string) TorrentStatus {
	switch qbState {
	// Actively downloading
	case "downloading", "metaDL", "forcedDL", "allocating", "checkingDL", "checkingResumeData", "moving":
		return StatusDownloading

	// Queued for download (blocked, needs slot)
	case "queuedDL":
		return StatusQueuedDL

	// Stalled during download (no peers)
	case "stalledDL":
		return StatusStalledDL

	// Paused state (both download and upload paused)
	case "pausedDL", "pausedUP":
		return StatusPaused

	// Seeding (uploading, at 100%)
	case "uploading", "forcedUP", "checkingUP", "queuedUP", "stalledUP":
		return StatusSeeding

	// Stopped while downloading (qB v5)
	case "stoppedDL":
		return StatusPaused

	// Fully downloaded and stopped
	case "stoppedUP":
		return StatusCompleted

	// Error states
	case "error", "missingFiles":
		return StatusError

	// Unknown/unmapped states treat as error
	default:
		return StatusError
	}
}

// GetSpeedLimitEnabled returns whether alternative speed limit is enabled
func (qa *QBittorrentAdapter) GetSpeedLimitEnabled(ctx context.Context) (bool, error) {
	syncURL := qa.getBaseURL() + "/api/v2/sync/maindata"

	req, err := http.NewRequestWithContext(ctx, "GET", syncURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create sync request: %w", err)
	}

	resp, err := qa.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("get sync failed: status %d", resp.StatusCode)
	}

	var syncData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&syncData); err != nil {
		return false, fmt.Errorf("failed to decode sync data: %w", err)
	}

	// Check server_state for use_alt_speed_limits
	if serverState, ok := syncData["server_state"].(map[string]interface{}); ok {
		if val, ok := serverState["use_alt_speed_limits"]; ok {
			if enabled, ok := val.(bool); ok {
				return enabled, nil
			}
		}
	}
	return false, nil
}

// SetSpeedLimitEnabled toggles alternative speed limit (doesn't matter what `enabled` value is - it just toggles)
func (qa *QBittorrentAdapter) SetSpeedLimitEnabled(ctx context.Context, enabled bool) error {
	toggleURL := qa.getBaseURL() + "/api/v2/transfer/toggleSpeedLimitsMode"

	req, err := http.NewRequestWithContext(ctx, "POST", toggleURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create toggle request: %w", err)
	}

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("toggle speed limits request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("toggle speed limits failed: status %d - %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetSpeedLimits returns the alternative speed limits in KB/s
func (qa *QBittorrentAdapter) GetSpeedLimits(ctx context.Context) (downKBs, upKBs int, err error) {
	syncURL := qa.getBaseURL() + "/api/v2/sync/maindata"

	req, err := http.NewRequestWithContext(ctx, "GET", syncURL, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create sync request: %w", err)
	}

	resp, err := qa.client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("get sync failed: status %d", resp.StatusCode)
	}

	var syncData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&syncData); err != nil {
		return 0, 0, fmt.Errorf("failed to decode sync data: %w", err)
	}

	// Extract speed limits from server_state (in bytes/sec, convert to KB/s)
	if serverState, ok := syncData["server_state"].(map[string]interface{}); ok {
		if val, ok := serverState["dl_rate_limit"].(float64); ok {
			downKBs = int(val / 1024)
		}
		if val, ok := serverState["up_rate_limit"].(float64); ok {
			upKBs = int(val / 1024)
		}
	}
	return downKBs, upKBs, nil
}

// GetTorrentDetail fetches detailed information about a specific torrent
func (qa *QBittorrentAdapter) GetTorrentDetail(ctx context.Context, id string) (*TorrentDetail, error) {
	infoURL := qa.getBaseURL() + "/api/v2/torrents/info?hashes=" + id

	req, err := http.NewRequestWithContext(ctx, "GET", infoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := qa.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get torrent info failed: status %d", resp.StatusCode)
	}

	var qbTorrents []qbTorrent
	if err := json.NewDecoder(resp.Body).Decode(&qbTorrents); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(qbTorrents) == 0 {
		return nil, fmt.Errorf("torrent not found")
	}

	qb := qbTorrents[0]

	// Parse tags (comma-separated string)
	tags := []string{}
	if qb.Tags != "" {
		tags = strings.Split(qb.Tags, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
	}

	return &TorrentDetail{
		ID:          qb.Hash,
		Name:        qb.Name,
		Category:    qb.Category,
		Tags:        tags,
		Comments:    qb.Comment,
		SavePath:    qb.SavePath,
		TotalSize:   qb.TotalSize,
		Downloaded:  qb.Downloaded,
		ContentPath: qb.ContentPath,
	}, nil
}

// GetTorrentFiles fetches the list of files in a torrent
func (qa *QBittorrentAdapter) GetTorrentFiles(ctx context.Context, id string) ([]TorrentFile, error) {
	filesURL := qa.getBaseURL() + "/api/v2/torrents/files?hash=" + id

	req, err := http.NewRequestWithContext(ctx, "GET", filesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := qa.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get files failed: status %d", resp.StatusCode)
	}

	var qbFiles []qbFile
	if err := json.NewDecoder(resp.Body).Decode(&qbFiles); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	files := make([]TorrentFile, len(qbFiles))
	for i, qbFile := range qbFiles {
		files[i] = TorrentFile{
			Index:      qbFile.Index,
			Name:       qbFile.Name,
			Size:       qbFile.Size,
			Downloaded: int64(float64(qbFile.Size) * qbFile.Progress),
			Priority:   qbFile.Priority,
		}
	}

	return files, nil
}

// SetTorrentName renames a torrent in qBittorrent
func (qa *QBittorrentAdapter) SetTorrentName(ctx context.Context, id string, newName string) error {
	renameURL := qa.getBaseURL() + "/api/v2/torrents/rename"

	form := url.Values{}
	form.Set("hash", id)
	form.Set("newName", newName)

	req, err := http.NewRequestWithContext(ctx, "POST", renameURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rename failed: status %d", resp.StatusCode)
	}

	return nil
}

// SetCategory changes the category of a torrent in qBittorrent
func (qa *QBittorrentAdapter) SetCategory(ctx context.Context, id string, category string) error {
	setURL := qa.getBaseURL() + "/api/v2/torrents/setCategory"

	form := url.Values{}
	form.Set("hashes", id)
	form.Set("category", category)

	req, err := http.NewRequestWithContext(ctx, "POST", setURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("set category failed: status %d", resp.StatusCode)
	}

	return nil
}

// SetTags updates tags for a torrent in qBittorrent
func (qa *QBittorrentAdapter) SetTags(ctx context.Context, id string, tags []string) error {
	setURL := qa.getBaseURL() + "/api/v2/torrents/addTags"

	// First, remove all existing tags for this torrent
	removeURL := qa.getBaseURL() + "/api/v2/torrents/removeTags"
	form := url.Values{}
	form.Set("hashes", id)
	form.Set("tags", "")

	req, err := http.NewRequestWithContext(ctx, "POST", removeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	resp.Body.Close()

	// Then add new tags
	form = url.Values{}
	form.Set("hashes", id)
	form.Set("tags", strings.Join(tags, ","))

	req, err = http.NewRequestWithContext(ctx, "POST", setURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("set tags failed: status %d", resp.StatusCode)
	}

	return nil
}

// SetSavePath changes the save/download location for a torrent in qBittorrent
func (qa *QBittorrentAdapter) SetSavePath(ctx context.Context, id string, path string) error {
	// Check if path exists first
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}

	setURL := qa.getBaseURL() + "/api/v2/torrents/setLocation"

	form := url.Values{}
	form.Set("hashes", id)
	form.Set("location", path)

	req, err := http.NewRequestWithContext(ctx, "POST", setURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("set location failed: status %d", resp.StatusCode)
	}

	return nil
}

// SetFilePriorities sets which files to download in a qBittorrent torrent
func (qa *QBittorrentAdapter) SetFilePriorities(ctx context.Context, id string, fileIndices []int) error {
	// qBittorrent 5.0 API: need to get all files first to determine which to skip
	filesURL := qa.getBaseURL() + "/api/v2/torrents/files?hash=" + id
	req, err := http.NewRequestWithContext(ctx, "GET", filesURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get files failed: status %d", resp.StatusCode)
	}

	var qbFiles []qbFile
	if err := json.NewDecoder(resp.Body).Decode(&qbFiles); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Create a set of indices to download for quick lookup
	downloadSet := make(map[int]bool)
	for _, idx := range fileIndices {
		downloadSet[idx] = true
	}

	// Separate files into two groups: those to download (priority 1) and those to skip (priority 0)
	// Use the index field from the API response as recommended since 2.8.2
	var toDownload []string
	var toSkip []string

	for _, file := range qbFiles {
		if downloadSet[file.Index] {
			toDownload = append(toDownload, fmt.Sprintf("%d", file.Index))
		} else {
			toSkip = append(toSkip, fmt.Sprintf("%d", file.Index))
		}
	}

	// qBittorrent 5.0+ API requires separate calls for each priority level
	// First, set files to skip (priority 0)
	if len(toSkip) > 0 {
		form := url.Values{}
		form.Set("hash", id)
		form.Set("id", strings.Join(toSkip, "|"))
		form.Set("priority", "0")

		priorityURL := qa.getBaseURL() + "/api/v2/torrents/filePrio"
		req, err = http.NewRequestWithContext(ctx, "POST", priorityURL, strings.NewReader(form.Encode()))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err = qa.client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("set skip file priorities failed: status %d", resp.StatusCode)
		}
	}

	// Then, set files to download (priority 1)
	if len(toDownload) > 0 {
		form := url.Values{}
		form.Set("hash", id)
		form.Set("id", strings.Join(toDownload, "|"))
		form.Set("priority", "1")

		priorityURL := qa.getBaseURL() + "/api/v2/torrents/filePrio"
		req, err = http.NewRequestWithContext(ctx, "POST", priorityURL, strings.NewReader(form.Encode()))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err = qa.client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("set download file priorities failed: status %d", resp.StatusCode)
		}
	}

	return nil
}

// SetLabels updates tags for a qBittorrent torrent (qBittorrent doesn't have labels, so we use tags)
func (qa *QBittorrentAdapter) SetLabels(ctx context.Context, id string, labels []string) error {
	// In qBittorrent, labels are represented as tags, so we delegate to SetTags
	return qa.SetTags(ctx, id, labels)
}

// RecheckTorrent forces a hash recheck of a torrent
func (qa *QBittorrentAdapter) RecheckTorrent(ctx context.Context, id string) error {
	actionURL := qa.getBaseURL() + "/api/v2/torrents/recheck"
	formData := url.Values{}
	formData.Set("hashes", id)

	req, err := http.NewRequestWithContext(ctx, "POST", actionURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create recheck request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("recheck request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("recheck failed: status %d", resp.StatusCode)
	}
	return nil
}

// ReannounceTorrent forces a tracker reannounce for a torrent
func (qa *QBittorrentAdapter) ReannounceTorrent(ctx context.Context, id string) error {
	actionURL := qa.getBaseURL() + "/api/v2/torrents/reannounce"
	formData := url.Values{}
	formData.Set("hashes", id)

	req, err := http.NewRequestWithContext(ctx, "POST", actionURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create reannounce request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("reannounce request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("reannounce failed: status %d", resp.StatusCode)
	}
	return nil
}

// GetMagnetURI returns the magnet URI for a torrent
func (qa *QBittorrentAdapter) GetMagnetURI(ctx context.Context, id string) (string, error) {
	infoURL := qa.getBaseURL() + "/api/v2/torrents/info?hashes=" + id

	req, err := http.NewRequestWithContext(ctx, "GET", infoURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := qa.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get torrent info failed: status %d", resp.StatusCode)
	}

	var qbTorrents []qbTorrent
	if err := json.NewDecoder(resp.Body).Decode(&qbTorrents); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(qbTorrents) == 0 {
		return "", fmt.Errorf("torrent not found")
	}

	return qbTorrents[0].MagnetURI, nil
}

// SetQueuePriority changes the queue position of a torrent
func (qa *QBittorrentAdapter) SetQueuePriority(ctx context.Context, id string, action string) error {
	var endpoint string
	switch action {
	case "top":
		endpoint = "/api/v2/torrents/topPrio"
	case "bottom":
		endpoint = "/api/v2/torrents/bottomPrio"
	case "up":
		endpoint = "/api/v2/torrents/increasePrio"
	case "down":
		endpoint = "/api/v2/torrents/decreasePrio"
	default:
		return fmt.Errorf("invalid queue priority action: %s", action)
	}

	actionURL := qa.getBaseURL() + endpoint
	formData := url.Values{}
	formData.Set("hashes", id)

	req, err := http.NewRequestWithContext(ctx, "POST", actionURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create queue priority request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qa.client.Do(req)
	if err != nil {
		return fmt.Errorf("queue priority request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("torrent queueing is not enabled")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("queue priority failed: status %d", resp.StatusCode)
	}
	return nil
}
