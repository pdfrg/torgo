package tui

import (
	"fmt"
	"math"
	"time"
)

// FormatBytes converts bytes to human-readable format (B, KB, MB, GB, TB)
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	exp := 0
	div := int64(1)
	for b := bytes; b >= unit; b /= unit {
		div *= unit
		exp++
		if exp >= 4 {
			break
		}
	}

	value := float64(bytes) / float64(div)

	switch exp {
	case 1:
		return fmt.Sprintf("%.1f KB", value)
	case 2:
		if value >= 100 {
			return fmt.Sprintf("%.0f MB", value)
		}
		return fmt.Sprintf("%.1f MB", value)
	case 3:
		if value >= 10 {
			return fmt.Sprintf("%.1f GB", value)
		}
		return fmt.Sprintf("%.2f GB", value)
	default:
		return fmt.Sprintf("%.2f TB", value)
	}
}

// FormatSpeed converts bytes/sec to human-readable speed (B/s, KB/s, MB/s, etc)
func FormatSpeed(bytesPerSec int64) string {
	if bytesPerSec == 0 {
		return "0 B/s"
	}

	const unit = 1024
	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	size := float64(bytesPerSec)

	for _, unitName := range units {
		if size < 1024.0 {
			if unitName == "B/s" {
				return fmt.Sprintf("%d %s", int64(size), unitName)
			}
			return fmt.Sprintf("%.1f %s", size, unitName)
		}
		size /= unit
	}

	return fmt.Sprintf("%.1f GB/s", size)
}

// FormatRatio converts a ratio float to a readable string
func FormatRatio(ratio float64) string {
	if ratio < 0 {
		return "N/A"
	}
	if ratio == math.Inf(1) {
		return "∞"
	}
	return fmt.Sprintf("%.2f", ratio)
}

// FormatETA estimates time remaining based on remaining bytes and speed
// Returns a human-readable string like "2h 15m", "5m 30s", etc.
// Returns "Done" if percent >= 1.0
// Returns "Calculating..." if speed is 0
func FormatETA(downloadedBytes, totalBytes, speedBytesPerSec int64) string {
	// Already complete
	if downloadedBytes >= totalBytes {
		return "Done"
	}

	// No speed - can't estimate
	if speedBytesPerSec <= 0 {
		return "Calculating..."
	}

	remainingBytes := totalBytes - downloadedBytes
	secondsRemaining := remainingBytes / speedBytesPerSec

	return FormatDuration(time.Duration(secondsRemaining) * time.Second)
}

// FormatDuration converts a duration to a human-readable string
// Examples: "2h 15m", "5m 30s", "45s", "0s"
func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	totalSeconds := int64(d.Seconds())
	if totalSeconds == 0 {
		return "0s"
	}

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}

	if minutes > 0 {
		if seconds > 0 {
			return fmt.Sprintf("%dm %ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	}

	return fmt.Sprintf("%ds", seconds)
}

// FormatPercent formats a percentage (0.0-1.0) as a string
func FormatPercent(percent float64) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}
	return fmt.Sprintf("%.0f%%", percent*100)
}

// FormatPeerCount formats seed/leech counts with optional total in parentheses.
// Examples: "5(12)", "5", "N/A" for negative numbers
func FormatPeerCount(connected, total int64) string {
	if connected < 0 {
		return "N/A"
	}
	if total > 0 && total != connected {
		return fmt.Sprintf("%d(%d)", connected, total)
	}
	return fmt.Sprintf("%d", connected)
}

// TruncateString truncates a string to maxLen with ellipsis if needed
// Accounts for multi-byte unicode characters
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// Simple truncation (doesn't account for rune width)
	if maxLen <= 3 {
		return "…"
	}

	return s[:maxLen-1] + "…"
}

// FormatProgressLine creates a formatted progress line
// Example: "Progress: ████░░░░ 50% (2.5GB / 5.0GB)"
func FormatProgressLine(progressBar string, downloaded, total int64) string {
	return fmt.Sprintf("Progress: %s %s (%s / %s)",
		progressBar,
		FormatPercent(float64(downloaded)/float64(total)),
		FormatBytes(downloaded),
		FormatBytes(total),
	)
}

// FormatStatusLine creates a formatted status line
// Example: "Status: Downloading  ↓ 2.5MB/s ↑ 0.3MB/s  Ratio: 0.85"
func FormatStatusLine(status string, downloadSpeed, uploadSpeed, ratio int64) string {
	ratioStr := FormatRatio(float64(ratio) / 100.0) // Assuming ratio is in hundredths
	return fmt.Sprintf("Status: %-12s  ↓ %s ↑ %s  Ratio: %s",
		status,
		FormatSpeed(downloadSpeed),
		FormatSpeed(uploadSpeed),
		ratioStr,
	)
}

// FormatInfoLine creates a formatted info line
// Example: "Seeds: 12(30)  Peers: 5(20)  ETA: 2h 15m"
func FormatInfoLine(seeds, totalSeeds, peers, totalPeers int64, eta string) string {
	return fmt.Sprintf("Seeds: %s  Peers: %s  ETA: %s",
		FormatPeerCount(seeds, totalSeeds),
		FormatPeerCount(peers, totalPeers),
		eta,
	)
}
