package tui

import (
	"strings"
)

// CategoryMapping maps common category/label names to emoji icons
// This makes it easy to customize for different preferences
type CategoryMapping struct {
	// map[lowercase_category_name] = emoji
	mappings map[string]string
}

// DefaultCategoryMapping returns the standard category-to-emoji mappings
func DefaultCategoryMapping() CategoryMapping {
	return CategoryMapping{
		mappings: map[string]string{
			// Movies (Radarr)
			"movie":    "🎬",
			"movies":   "🎬",
			"film":     "🎬",
			"films":    "🎬",
			"cinema":   "🎬",
			"feature":  "🎬",
			"radarr":   "🎬",

			// TV Shows (Sonarr)
			"tv":       "📺",
			"tvshow":   "📺",
			"tvshows":  "📺",
			"tv show":  "📺",
			"tv shows": "📺",
			"series":   "📺",
			"show":     "📺",
			"shows":    "📺",
			"episode":  "📺",
			"episodes": "📺",
			"sonarr":   "📺",

			// Adult (Whisparr)
			"whisparr": "🔞",
			"adult":    "🔞",
			"xxx":      "🔞",

			// Comics (Mylar3)
			"mylar":   "💥",
			"comic":   "💥",
			"comics":  "💥",

			// Music & Audio (Lidarr)
			"music":      "🎵",
			"audio":      "🎵",
			"song":       "🎵",
			"songs":      "🎵",
			"album":      "🎵",
			"albums":     "🎵",
			"lidarr":     "🎵",
			"podcast":    "📻",
			"podcasts":   "📻",
			"audiobook":  "📖",
			"audiobooks": "📖",

			// Software & Applications
			"software":   "💻",
			"app":        "💻",
			"apps":       "💻",
			"application": "💻",
			"applications": "💻",
			"program":    "💻",
			"programs":   "💻",
			"tool":       "🔧",
			"tools":      "🔧",
			"utility":    "🔧",
			"utilities":  "🔧",

			// Games
			"game":       "👾",
			"games":      "👾",
			"gaming":     "👾",
			"playstation": "👾",
			"xbox":       "👾",
			"nintendo":   "👾",
			"pc game":    "👾",

			// Books & Documents (Readarr)
			"readarr":   "📚",
			"book":      "📚",
			"books":     "📚",
			"ebook":     "📚",
			"ebooks":    "📚",
			"document":  "📚",
			"documents": "📚",
			"pdf":       "📄",
			"text":      "📄",

			// Images & Photos
			"photo":     "📷",
			"photos":    "📷",
			"image":     "📷",
			"images":    "📷",
			"picture":   "📷",
			"pictures":  "📷",
			"wallpaper": "📷",
			"wallpapers": "📷",

			// Video (non-movie)
			"video":    "🎥",
			"videos":   "🎥",
			"tutorial": "🎥",
			"tutorials": "🎥",
			"stream":   "🎥",

			// Operating Systems
			"linux":     "🐧",
			"ubuntu":    "🐧",
			"debian":    "🐧",
			"fedora":    "🐧",
			"arch":      "🐧",
			"centos":    "🐧",
			"windows":   "🪟",
			"macos":     "🍎",
			"osx":       "🍎",
			"iso":       "💿",
			"distro":    "🐧",
			"distros":   "🐧",

			// Development
			"development": "💾",
			"developer":   "💾",
			"source code": "💾",
			"code":        "💾",
			"git":         "💾",
			"repository":  "💾",

			// Archives & Backups
			"archive":    "📦",
			"archives":   "📦",
			"compressed": "📦",
			"backup":     "💾",
			"backups":    "💾",
			"tar":        "📦",
			"zip":        "📦",
			"rar":        "📦",

			// Education
			"education":  "🎓",
			"course":     "🎓",
			"courses":    "🎓",
			"training":   "🎓",
			"lecture":    "🎓",
			"school":     "🎓",
			"university": "🎓",

			// Anime & Manga
			"anime":  "🌸",
			"manga":  "📖",
			"donghua": "🌸",
			"manhua":  "📖",
			"cartoon": "🎬",

			// Sports (all use soccer ball)
			"sports":     "⚽",
			"sport":      "⚽",
			"soccer":     "⚽",
			"football":   "⚽",
			"basketball": "⚽",
			"hockey":     "⚽",
			"tennis":     "⚽",
			"boxing":     "⚽",
			"mma":        "⚽",
			"racing":     "⚽",
			"cricket":    "⚽",
			"rugby":      "⚽",

			// Fitness & Health
			"fitness":  "💪",
			"workout":  "💪",
			"health":   "⚕️",
			"medical":  "⚕️",
			"yoga":     "🧘",

			// Nature & Animals
			"nature":   "🌿",
			"wildlife": "🦁",
			"animals":  "🦁",
			"pet":      "🐾",
			"pets":     "🐾",

			// Travel
			"travel":    "✈️",
			"tourism":   "✈️",
			"destination": "🗺️",
			"guide":     "🗺️",

			// News & Media
			"news":         "📰",
			"documentary":  "📺",

			// Art & Design
			"art":       "🎨",
			"design":    "🎨",
			"graphic":   "🎨",
			"3d":        "🎨",
			"animation": "🎬",

			// Default fallback
			"default": "📁",
			"misc":    "📁",
			"other":   "📁",
			"various": "📁",
		},
	}
}

// GetIcon returns the emoji icon for a category name
// Returns "📁" (folder) if category not found
func (cm CategoryMapping) GetIcon(category string) string {
	if category == "" {
		return cm.mappings["default"]
	}

	// Normalize: trim whitespace, lowercase
	normalized := strings.ToLower(strings.TrimSpace(category))

	// Try exact match
	if icon, ok := cm.mappings[normalized]; ok {
		return icon
	}

	// Try substring match (e.g., "movies" contains "movie")
	for key, icon := range cm.mappings {
		if strings.Contains(normalized, key) || strings.Contains(key, normalized) {
			return icon
		}
	}

	// Fallback
	return cm.mappings["default"]
}

// GetIcons returns emoji icons for multiple categories/labels
// Used for Transmission which can have multiple labels
func (cm CategoryMapping) GetIcons(categories []string) string {
	icons := ""
	seen := make(map[string]bool)

	for _, cat := range categories {
		icon := cm.GetIcon(cat)
		// Avoid duplicates
		if !seen[icon] {
			icons += icon
			seen[icon] = true
		}
	}

	return icons
}

// AddMapping adds or overrides a category-to-emoji mapping
func (cm *CategoryMapping) AddMapping(category, emoji string) {
	if cm.mappings == nil {
		cm.mappings = make(map[string]string)
	}
	cm.mappings[strings.ToLower(strings.TrimSpace(category))] = emoji
}

// CurrentCategoryMapping holds the active category mapping
var CurrentCategoryMapping = DefaultCategoryMapping()

// SetCategoryMapping changes the active category mapping
func SetCategoryMapping(cm CategoryMapping) {
	CurrentCategoryMapping = cm
}

// GetCategoryIcon returns the emoji icon for a category using the current mapping
func GetCategoryIcon(category string) string {
	return CurrentCategoryMapping.GetIcon(category)
}

// GetCategoryIcons returns emoji icons for multiple categories using the current mapping
func GetCategoryIcons(categories []string) string {
	return CurrentCategoryMapping.GetIcons(categories)
}
