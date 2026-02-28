package tui

import (
	"net/url"
	"os"
	"strings"
)

// ValidateInput validates magnet links, URLs, and file paths
// Returns error message if invalid, or empty string if valid
func ValidateInput(input string) string {
	input = strings.TrimSpace(input)

	if input == "" {
		return "" // Empty is valid at this stage, will be caught on submit
	}

	// Check if it's a magnet link
	if strings.HasPrefix(input, "magnet:") {
		return validateMagnet(input)
	}

	// Check if it's a URL (http/https)
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return validateURL(input)
	}

	// Otherwise, treat as file path
	return validateFilePath(input)
}

// validateMagnet validates magnet link format
func validateMagnet(magnet string) string {
	// Basic magnet link validation: should start with magnet: and contain ?
	if !strings.Contains(magnet, "?") {
		return "Invalid magnet link format"
	}

	// Extract the query part (after ?)
	parts := strings.SplitN(magnet, "?", 2)
	if len(parts) < 2 {
		return "Invalid magnet link format"
	}

	queryPart := parts[1]

	// Split by & to get individual parameters
	params := strings.Split(queryPart, "&")
	if len(params) == 0 {
		return "Invalid magnet link format"
	}

	// A valid magnet should have at least xt= parameter
	hasXt := false
	for _, param := range params {
		if strings.HasPrefix(param, "xt=") {
			hasXt = true
			break
		}
	}

	if !hasXt {
		return "Magnet link missing xt parameter"
	}

	return ""
}

// validateURL validates HTTP/HTTPS URL format
func validateURL(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "Invalid URL format"
	}

	// Check if scheme is http or https (case-insensitive)
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return "Only HTTP and HTTPS URLs are supported"
	}

	// Check if host is present
	if parsedURL.Host == "" {
		return "Invalid URL: missing host"
	}

	return ""
}

// validateFilePath validates if the file path exists
func validateFilePath(path string) string {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "Could not determine home directory"
		}
		path = home + path[1:]
	}

	// Check if file exists
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "File not found"
		}
		return "Error accessing file: " + err.Error()
	}

	// Optionally check if it has .torrent extension
	if !strings.HasSuffix(strings.ToLower(path), ".torrent") {
		return "File must be a .torrent file"
	}

	return ""
}

// IsValidForSubmit performs stricter validation for form submission
func IsValidForSubmit(input string) string {
	input = strings.TrimSpace(input)

	if input == "" {
		return "Please enter a magnet link, URL, or .torrent file path"
	}

	// Use the regular validation
	return ValidateInput(input)
}

// ShouldHighlightError returns true if error message should be shown in red
// This helps distinguish between "not ready yet" (no error shown) vs actual errors
func ShouldHighlightError(errorMsg string) bool {
	return errorMsg != ""
}
