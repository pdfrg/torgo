package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TransmissionAdapter implements ClientAdapter for Transmission
type TransmissionAdapter struct {
	host      string
	port      int
	username  string
	password  string
	connected bool
	client    *http.Client
	sessionID string // X-Transmission-Session-Id header
}

// trTorrent represents a Transmission torrent
type trTorrent struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	PercentDone     float64 `json:"percentDone"` // 0-1
	RateDownload    float64 `json:"rateDownload"`
	RateUpload      float64 `json:"rateUpload"`
	Status          int     `json:"status"` // 0=stopped, 1=check waiting, 2=checking, 3=downloading, 4=seeding, 5=seed waiting, 6=stopped
	PeersSendingToUs int     `json:"peersSendingToUs"`
	PeersGettingFromUs int  `json:"peersGettingFromUs"`
	TotalSize       int64   `json:"totalSize"`
	DownloadedEver  int64   `json:"downloadedEver"`
	UploadedEver    int64   `json:"uploadedEver"`
	Hash            string  `json:"hashString"`
	DownloadDir     string  `json:"downloadDir"`
	Labels          []string `json:"labels"`
	Comment         string  `json:"comment"`
	Files           []trFile `json:"files"`
	FileStats       []trFileStat `json:"fileStats"`
}

// trFile represents a file in a Transmission torrent
type trFile struct {
	Name   string `json:"name"`
	Length int64  `json:"length"`
}

// trFileStat represents the stats for a file in a Transmission torrent
type trFileStat struct {
	BytesCompleted int64  `json:"bytesCompleted"`
	Wanted         bool   `json:"wanted"`
	Priority       int    `json:"priority"` // -1=low, 0=normal, 1=high
}

// trResponse is the wrapper for Transmission RPC responses
type trResponse struct {
	Result    string      `json:"result"`
	Arguments trArguments `json:"arguments"`
}

// trArguments contains the response arguments
type trArguments struct {
	Torrents []trTorrent `json:"torrents"`
}

// NewTransmissionAdapter creates a new Transmission adapter
func NewTransmissionAdapter(host string, port int, username, password string) *TransmissionAdapter {
	return &TransmissionAdapter{
		host:     host,
		port:     port,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Connect tests connection to the Transmission instance
func (ta *TransmissionAdapter) Connect(ctx context.Context) error {
	// First request to get session ID
	payload := `{"method":"session-get"}`
	req, err := ta.buildRPCRequest(ctx, payload)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := ta.client.Do(req)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer resp.Body.Close()

	// Extract session ID from response header
	if sessionID := resp.Header.Get("X-Transmission-Session-Id"); sessionID != "" {
		ta.sessionID = sessionID
	}

	// 409 is expected on first request (need session ID)
	if resp.StatusCode == http.StatusConflict {
		// Retry with session ID
		req, err = ta.buildRPCRequest(ctx, payload)
		if err != nil {
			return fmt.Errorf("failed to build retry request: %w", err)
		}

		resp, err = ta.client.Do(req)
		if err != nil {
			return fmt.Errorf("retry failed: %w", err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("connection failed: %s (status %d)", string(body), resp.StatusCode)
	}

	ta.connected = true
	return nil
}

// Disconnect closes the connection
func (ta *TransmissionAdapter) Disconnect(ctx context.Context) error {
	ta.connected = false
	return nil
}

// IsConnected returns whether the adapter is connected
func (ta *TransmissionAdapter) IsConnected() bool {
	return ta.connected
}

// ListTorrents fetches all torrents from Transmission
func (ta *TransmissionAdapter) ListTorrents(ctx context.Context) ([]Torrent, error) {
	payload := `{
		"method":"torrent-get",
		"arguments":{
			"fields":["id","name","percentDone","rateDownload","rateUpload","status",
				"peersSendingToUs","peersGettingFromUs","totalSize","downloadedEver","uploadedEver","hashString"]
		}
	}`

	resp, err := ta.sendRPC(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("torrent-get failed: %w", err)
	}

	torrents := make([]Torrent, len(resp.Arguments.Torrents))
	for i, tr := range resp.Arguments.Torrents {
		torrents[i] = ta.mapTorrent(tr)
	}
	return torrents, nil
}

// PauseTorrent pauses a single torrent
func (ta *TransmissionAdapter) PauseTorrent(ctx context.Context, id string) error {
	return ta.torrentAction(ctx, "torrent-stop", id)
}

// ResumeTorrent resumes a single torrent
func (ta *TransmissionAdapter) ResumeTorrent(ctx context.Context, id string) error {
	return ta.torrentAction(ctx, "torrent-start", id)
}

// RemoveTorrent removes a torrent without deleting files
func (ta *TransmissionAdapter) RemoveTorrent(ctx context.Context, id string) error {
	return ta.torrentRemove(ctx, id, false)
}

// RemoveTorrentWithData removes a torrent and deletes files
func (ta *TransmissionAdapter) RemoveTorrentWithData(ctx context.Context, id string) error {
	return ta.torrentRemove(ctx, id, true)
}

// AddTorrent adds a torrent from magnet link, URL, or .torrent file with optional labels
func (ta *TransmissionAdapter) AddTorrent(ctx context.Context, input string, label string) error {
	var payload string
	var args string

	if strings.HasPrefix(input, "magnet:") || strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		// Add magnet link or URL
		escapedInput := strings.ReplaceAll(input, "\"", "\\\"")
		args = fmt.Sprintf(`"filename":"%s"`, escapedInput)
	} else {
		// Handle file path
		fileBytes, err := os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("failed to read torrent file: %w", err)
		}
		encoded := base64.StdEncoding.EncodeToString(fileBytes)
		args = fmt.Sprintf(`"metainfo":"%s"`, encoded)
	}

	// Build arguments with optional labels
	if label != "" {
		escapedLabel := strings.ReplaceAll(label, "\"", "\\\"")
		payload = fmt.Sprintf(`{
			"method":"torrent-add",
			"arguments":{%s,"labels":["%s"]}
		}`, args, escapedLabel)
	} else {
		payload = fmt.Sprintf(`{
			"method":"torrent-add",
			"arguments":{%s}
		}`, args)
	}

	resp, err := ta.sendRPC(ctx, payload)
	if err != nil {
		return fmt.Errorf("torrent-add failed: %w", err)
	}

	if resp.Result != "success" {
		return fmt.Errorf("torrent-add failed: %s", resp.Result)
	}

	return nil
}

// GetCategories returns available labels from Transmission
func (ta *TransmissionAdapter) GetCategories(ctx context.Context) ([]string, error) {
	// Transmission doesn't have a direct way to get all labels
	// We need to fetch all torrents and extract unique labels
	payload := `{
		"method":"torrent-get",
		"arguments":{"fields":["labels"]}
	}`

	resp, err := ta.sendRPC(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch torrents: %w", err)
	}

	if resp.Result != "success" {
		return nil, fmt.Errorf("failed to fetch torrents: %s", resp.Result)
	}

	// Extract unique labels from all torrents
	labelSet := make(map[string]struct{})
	for range resp.Arguments.Torrents {
		// TODO: Extract labels once trTorrent struct is updated to include Labels field
	}

	// Convert to sorted slice
	labels := make([]string, 0, len(labelSet))
	for label := range labelSet {
		if label != "" { // Skip empty labels
			labels = append(labels, label)
		}
	}
	sort.Strings(labels)
	return labels, nil
}

// PauseAll pauses all torrents
func (ta *TransmissionAdapter) PauseAll(ctx context.Context) error {
	payload := `{"method":"torrent-stop","arguments":{"ids":"recently-active"}}`
	resp, err := ta.sendRPC(ctx, payload)
	if err != nil {
		return fmt.Errorf("torrent-stop-all failed: %w", err)
	}
	if resp.Result != "success" {
		return fmt.Errorf("torrent-stop-all failed: %s", resp.Result)
	}
	return nil
}

// ResumeAll resumes all torrents
func (ta *TransmissionAdapter) ResumeAll(ctx context.Context) error {
	payload := `{"method":"torrent-start","arguments":{"ids":"recently-active"}}`
	resp, err := ta.sendRPC(ctx, payload)
	if err != nil {
		return fmt.Errorf("torrent-start-all failed: %w", err)
	}
	if resp.Result != "success" {
		return fmt.Errorf("torrent-start-all failed: %s", resp.Result)
	}
	return nil
}

// Helper methods

func (ta *TransmissionAdapter) getRPCURL() string {
	return fmt.Sprintf("http://%s:%d/transmission/rpc", ta.host, ta.port)
}

func (ta *TransmissionAdapter) buildRPCRequest(ctx context.Context, payload string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", ta.getRPCURL(), strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if ta.sessionID != "" {
		req.Header.Set("X-Transmission-Session-Id", ta.sessionID)
	}

	// Add basic auth if credentials provided
	if ta.username != "" || ta.password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(ta.username + ":" + ta.password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	return req, nil
}

func (ta *TransmissionAdapter) sendRPC(ctx context.Context, payload string) (*trResponse, error) {
	req, err := ta.buildRPCRequest(ctx, payload)
	if err != nil {
		return nil, err
	}

	resp, err := ta.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle session ID updates
	if sessionID := resp.Header.Get("X-Transmission-Session-Id"); sessionID != "" {
		ta.sessionID = sessionID
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("RPC failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var trResp trResponse
	if err := decodeJSON(resp.Body, &trResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &trResp, nil
}

func (ta *TransmissionAdapter) torrentAction(ctx context.Context, method, id string) error {
	payload := fmt.Sprintf(`{"method":"%s","arguments":{"ids":[%s]}}`, method, id)
	resp, err := ta.sendRPC(ctx, payload)
	if err != nil {
		return fmt.Errorf("%s failed: %w", method, err)
	}
	if resp.Result != "success" {
		return fmt.Errorf("%s failed: %s", method, resp.Result)
	}
	return nil
}

func (ta *TransmissionAdapter) torrentRemove(ctx context.Context, id string, deleteData bool) error {
	deleteFlag := "false"
	if deleteData {
		deleteFlag = "true"
	}
	payload := fmt.Sprintf(`{
		"method":"torrent-remove",
		"arguments":{"ids":[%s],"delete-local-data":%s}
	}`, id, deleteFlag)

	resp, err := ta.sendRPC(ctx, payload)
	if err != nil {
		return fmt.Errorf("torrent-remove failed: %w", err)
	}
	if resp.Result != "success" {
		return fmt.Errorf("torrent-remove failed: %s", resp.Result)
	}
	return nil
}

func (ta *TransmissionAdapter) mapTorrent(tr trTorrent) Torrent {
	return Torrent{
		ID:         strconv.FormatInt(tr.ID, 10),
		Name:       tr.Name,
		Progress:   uint8(tr.PercentDone * 100),
		SpeedDown:  tr.RateDownload,
		SpeedUp:    tr.RateUpload,
		Status:     ta.mapStatus(tr.Status),
		Seeds:      tr.PeersSendingToUs,
		Leechs:     tr.PeersGettingFromUs,
		Size:       tr.TotalSize,
		Downloaded: tr.DownloadedEver,
		Uploaded:   tr.UploadedEver,
	}
}

func (ta *TransmissionAdapter) mapStatus(trStatus int) TorrentStatus {
	switch trStatus {
	case 0: // TR_STATUS_STOPPED
		return StatusPaused
	case 1, 2: // TR_STATUS_CHECK_WAIT, TR_STATUS_CHECK
		return StatusDownloading
	case 3: // TR_STATUS_DOWNLOAD
		return StatusDownloading
	case 4, 5: // TR_STATUS_SEED, TR_STATUS_SEED_WAIT
		return StatusSeeding
	default:
		return StatusError
	}
}

// decodeJSON decodes JSON from a reader
func decodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

// GetSpeedLimitEnabled returns whether speed limit mode is enabled
func (ta *TransmissionAdapter) GetSpeedLimitEnabled(ctx context.Context) (bool, error) {
	// For Transmission, we'll use a more flexible approach by parsing session-get
	payload := `{
		"method":"session-get",
		"arguments":{}
	}`

	req, err := ta.buildRPCRequest(ctx, payload)
	if err != nil {
		return false, fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := ta.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("session-get request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("session-get failed: status %d", resp.StatusCode)
	}

	var sessionResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return false, fmt.Errorf("failed to decode session response: %w", err)
	}

	// Check for speed-limit-down-enabled in arguments
	if args, ok := sessionResp["arguments"].(map[string]interface{}); ok {
		if speedLimit, ok := args["speed-limit-down-enabled"].(bool); ok {
			return speedLimit, nil
		}
	}
	return false, nil
}

// SetSpeedLimitEnabled enables or disables speed limit mode
func (ta *TransmissionAdapter) SetSpeedLimitEnabled(ctx context.Context, enabled bool) error {
	payload := fmt.Sprintf(`{
		"method":"session-set",
		"arguments":{"speed-limit-down-enabled":%v,"speed-limit-up-enabled":%v}
	}`, enabled, enabled)

	resp, err := ta.sendRPC(ctx, payload)
	if err != nil {
		return fmt.Errorf("session-set failed: %w", err)
	}
	if resp.Result != "success" {
		return fmt.Errorf("session-set failed: %s", resp.Result)
	}
	return nil
}

// GetSpeedLimits returns the speed limits in KB/s
func (ta *TransmissionAdapter) GetSpeedLimits(ctx context.Context) (downKBs, upKBs int, err error) {
	payload := `{
		"method":"session-get",
		"arguments":{}
	}`

	req, err := ta.buildRPCRequest(ctx, payload)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := ta.client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("session-get request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("session-get failed: status %d", resp.StatusCode)
	}

	var sessionResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return 0, 0, fmt.Errorf("failed to decode session response: %w", err)
	}

	// Extract speed limits
	if args, ok := sessionResp["arguments"].(map[string]interface{}); ok {
		if down, ok := args["speed-limit-down"].(float64); ok {
			downKBs = int(down)
		}
		if up, ok := args["speed-limit-up"].(float64); ok {
			upKBs = int(up)
		}
	}
	return downKBs, upKBs, nil
}

// GetTorrentDetail fetches detailed information about a specific torrent
func (ta *TransmissionAdapter) GetTorrentDetail(ctx context.Context, id string) (*TorrentDetail, error) {
	payload := fmt.Sprintf(`{
		"method":"torrent-get",
		"arguments":{
			"ids":[%s],
			"fields":["name","downloadDir","labels","comment","files","fileStats","totalSize","downloadedEver"]
		}
	}`, id)

	resp, err := ta.sendRPC(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("torrent-get failed: %w", err)
	}

	if len(resp.Arguments.Torrents) == 0 {
		return nil, fmt.Errorf("torrent not found")
	}

	tr := resp.Arguments.Torrents[0]

	// Combine labels into a single category (Transmission uses multiple labels, but we'll show them comma-separated)
	category := ""
	if len(tr.Labels) > 0 {
		category = tr.Labels[0]
	}

	return &TorrentDetail{
		ID:          strconv.FormatInt(tr.ID, 10),
		Name:        tr.Name,
		Category:    category,
		Tags:        tr.Labels, // In Transmission, labels are used like tags
		Comments:    tr.Comment,
		SavePath:    tr.DownloadDir,
		TotalSize:   tr.TotalSize,
		Downloaded:  tr.DownloadedEver,
	}, nil
}

// GetTorrentFiles fetches the list of files in a torrent
func (ta *TransmissionAdapter) GetTorrentFiles(ctx context.Context, id string) ([]TorrentFile, error) {
	payload := fmt.Sprintf(`{
		"method":"torrent-get",
		"arguments":{
			"ids":[%s],
			"fields":["files","fileStats"]
		}
	}`, id)

	resp, err := ta.sendRPC(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("torrent-get failed: %w", err)
	}

	if len(resp.Arguments.Torrents) == 0 {
		return nil, fmt.Errorf("torrent not found")
	}

	tr := resp.Arguments.Torrents[0]

	// Build file list from files and fileStats
	files := make([]TorrentFile, len(tr.Files))
	for i, f := range tr.Files {
		downloaded := int64(0)
		wanted := true
		if i < len(tr.FileStats) {
			downloaded = tr.FileStats[i].BytesCompleted
			wanted = tr.FileStats[i].Wanted
		}

		priority := 0 // Download (1 = normal in Transmission)
		if !wanted {
			priority = 0 // Do not download (0 = wanted=false)
		}

		files[i] = TorrentFile{
			Index:      i,
			Name:       f.Name,
			Size:       f.Length,
			Downloaded: downloaded,
			Priority:   priority,
		}
	}

	return files, nil
}

// SetTorrentName renames a torrent (not supported in Transmission)
func (ta *TransmissionAdapter) SetTorrentName(ctx context.Context, id string, newName string) error {
	return fmt.Errorf("SetTorrentName not supported in Transmission")
}

// SetCategory changes the category/label of a torrent (Transmission uses download directory instead)
func (ta *TransmissionAdapter) SetCategory(ctx context.Context, id string, category string) error {
	// Transmission doesn't have categories like qBittorrent, so we ignore this
	// The user would need to manually organize by directory
	return nil
}

// SetTags updates tags for a torrent (not a core Transmission feature)
func (ta *TransmissionAdapter) SetTags(ctx context.Context, id string, tags []string) error {
	// Transmission doesn't have tags like qBittorrent
	return nil
}

// SetSavePath changes the save/download location for a torrent in Transmission
func (ta *TransmissionAdapter) SetSavePath(ctx context.Context, id string, path string) error {
	payload := fmt.Sprintf(`{
		"method":"torrent-set",
		"arguments":{
			"ids":[%s],
			"downloadDir":"%s"
		}
	}`, id, path)

	resp, err := ta.sendRPC(ctx, payload)
	if err != nil {
		return fmt.Errorf("torrent-set failed: %w", err)
	}

	if resp.Result != "success" {
		return fmt.Errorf("set location failed: %s", resp.Result)
	}

	return nil
}
