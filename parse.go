// Package torrentname provides parsing of torrent names into structured metadata
package torrentname

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TorrentInfo contains all metadata parsed from a torrent name
type TorrentInfo struct {
	Title        string   `json:"title"`
	Year         int      `json:"year,omitempty"`
	Date         string   `json:"date,omitempty"` // For daily shows (YYYY.MM.DD format)
	Season       int      `json:"season,omitempty"`
	Episodes     []int    `json:"episodes,omitempty"`
	Resolution   string   `json:"resolution,omitempty"`
	Source       string   `json:"source,omitempty"`
	Codec        string   `json:"codec,omitempty"`
	Audio        string   `json:"audio,omitempty"`
	ReleaseGroup string   `json:"release_group,omitempty"`
	Container    string   `json:"container,omitempty"`
	Language     string   `json:"language,omitempty"`
	Subtitles    []string `json:"subtitles,omitempty"`
	IsComplete   bool     `json:"is_complete,omitempty"`
	IsProper     bool     `json:"is_proper,omitempty"`
	IsRepack     bool     `json:"is_repack,omitempty"`
	IsHardcoded  bool     `json:"is_hardcoded,omitempty"`
	Edition      string   `json:"edition,omitempty"` // Director's Cut, Extended, etc.
	Confidence   float64  `json:"confidence"`        // 0.0 to 1.0
}

// Common patterns
var (
	yearPattern       = regexp.MustCompile(`\b(19\d{2}|20\d{2})\b`)
	seasonPattern     = regexp.MustCompile(`(?i)S(\d{1,2})`)
	seasonAltPattern  = regexp.MustCompile(`(?i)Season[\.\s]?(\d{1,2})`)
	episodePattern    = regexp.MustCompile(`(?i)S\d{1,2}E(\d{1,3})`)
	altEpisodePattern = regexp.MustCompile(`(?i)(\d{1,2})x(\d{1,3})`)
	datePattern       = regexp.MustCompile(`(\d{4})[\.\-](\d{2})[\.\-](\d{2})`)

	// Quality patterns
	resolutionPattern = regexp.MustCompile(`(?i)(2160p|4K|1080p|720p|480p|360p)`)
	sourcePattern     = regexp.MustCompile(`(?i)\b(BLURAY|BLU-RAY|WEB-DL|WEBDL|WEBRIP|WEB|HDTV|CAM|TC|DVD|BRRIP|BDRIP)\b`)
	codecPattern      = regexp.MustCompile(`(?i)\b(H264|X264|AVC|H265|X265|HEVC|MPEG2|MPEG4)\b`)
	audioPattern      = regexp.MustCompile(`(?i)\b(AAC|AC3|DTS|FLAC|TRUEHD|MP3|OGG|WAV)\b`)

	// Edition patterns - only match when they're standalone metadata
	editionPattern = regexp.MustCompile(`(?i)\b(Directors?\.?\s?Cut|Extended\.?\s?Cut|Extended|Unrated|Rated|Theatrical|Final\.?\s?Cut)\b`)

	// Status patterns - only match when they're standalone metadata
	completePattern  = regexp.MustCompile(`(?i)\b(Complete|COMPLETE)\b`)
	properPattern    = regexp.MustCompile(`(?i)\b(PROPER|REPACK)\b`)
	hardcodedPattern = regexp.MustCompile(`(?i)\b(HC|HARDCODED)\b`)

	// Language patterns
	languagePattern = regexp.MustCompile(`(?i)\b(ENGLISH|FRENCH|SPANISH|GERMAN|ITALIAN|DANISH|DUTCH|JAPANESE|CANTONESE|MANDARIN|RUSSIAN|POLISH|VIETNAMESE|SWEDISH|NORWEGIAN|FINNISH|TURKISH|PORTUGUESE|KOREAN|MULTI)\b`)
	subsPattern     = regexp.MustCompile(`(?i)(SUBS|SUBBED|SUB)`)

	// Container patterns
	containerPattern = regexp.MustCompile(`(?i)\.(mkv|mp4|avi|mov|wmv|flv|webm)`)

	// Tracker-specific patterns
	btnSeasonPack = regexp.MustCompile(`(?i)S(\d{1,2})[\.\s]?Complete`)
	ptnYearRange  = regexp.MustCompile(`(\d{4})-(\d{4})`)
)

// Parse analyzes a torrent name and extracts metadata
func Parse(name string) *TorrentInfo {
	info := &TorrentInfo{
		Confidence: 1.0,
	}

	// Extract container first (it's usually at the end)
	if matches := containerPattern.FindAllStringSubmatch(name, -1); len(matches) > 0 {
		last := matches[len(matches)-1]
		info.Container = strings.ToLower(last[1])
		// Remove extension for further parsing
		name = name[:strings.LastIndex(name, last[0])]
	}

	// Check for date-based episodes (common for daily shows)
	dateMatch := datePattern.FindStringSubmatch(name)

	// Extract season and episodes
	info.parseTVInfo(name, dateMatch != nil)

	// Extract quality information
	info.parseQuality(name)

	// Extract status flags - only if they appear after quality information or release year
	hasQualityInfo := resolutionPattern.MatchString(name) || sourcePattern.MatchString(name) || codecPattern.MatchString(name) || audioPattern.MatchString(name)

	// Only consider the last year as the release year for enabling metadata
	yearMatchesForMetadata := yearPattern.FindAllStringSubmatch(name, -1)
	hasReleaseYear := len(yearMatchesForMetadata) > 0

	if hasQualityInfo || hasReleaseYear {
		// Only consider these as metadata if quality info or release year is present
		info.IsComplete = completePattern.MatchString(name) || btnSeasonPack.MatchString(name)
		info.IsProper = properPattern.MatchString(name) && !strings.Contains(strings.ToUpper(name), "REPACK")
		info.IsRepack = strings.Contains(strings.ToUpper(name), "REPACK")
		info.IsHardcoded = hardcodedPattern.MatchString(name)

		// Extract edition only if quality info or release year is present
		if match := editionPattern.FindStringSubmatch(name); match != nil {
			edition := cleanString(match[1])
			info.Edition = strings.Title(strings.ToLower(edition))
		}
	} else {
		// No quality info or release year, so these are likely part of titles, not metadata
		info.IsComplete = false
		info.IsProper = false
		info.IsRepack = false
		info.IsHardcoded = false
		info.Edition = ""
	}

	// Extract language and subtitles
	info.parseLanguage(name)

	// Extract release group (usually at the end)
	info.ReleaseGroup = extractReleaseGroup(name)

	// Extract year - use the last year found as release year (do this after title extraction)
	if len(yearMatchesForMetadata) > 0 {
		// If there are multiple year-like numbers, use the last one
		if len(yearMatchesForMetadata) > 1 {
			lastYearMatch := yearMatchesForMetadata[len(yearMatchesForMetadata)-1]
			if year, err := strconv.Atoi(lastYearMatch[1]); err == nil && year >= 1895 && year <= getCurrentYear() {
				info.Year = year
			}
		} else {
			// Only one year-like number, check if it's the first word
			firstWord := strings.Split(name, ".")[0]
			if yearMatchesForMetadata[0][1] != firstWord {
				if year, err := strconv.Atoi(yearMatchesForMetadata[0][1]); err == nil && year >= 1895 && year <= getCurrentYear() {
					info.Year = year
				}
			}
		}
	}

	// Extract title
	info.Title = extractTitle(name, info)

	// Calculate confidence based on what we found
	info.calculateConfidence()

	return info
}

// ParseWithHints parses with tracker-specific hints
func ParseWithHints(name string, tracker string) *TorrentInfo {
	info := Parse(name)

	// Apply tracker-specific adjustments
	switch strings.ToLower(tracker) {
	case "btn", "broadcasthenet":
		// BTN uses "Season X Complete" format
		if match := btnSeasonPack.FindStringSubmatch(name); match != nil {
			info.Season, _ = strconv.Atoi(match[1])
			info.IsComplete = true
		}

	case "ptp", "passthepopcorn":
		// PTP sometimes uses year ranges for collections
		if match := ptnYearRange.FindStringSubmatch(name); match != nil {
			info.Year, _ = strconv.Atoi(match[1])
			// Could store end year in a new field if needed
		}

	case "hdb", "hdbits":
		// HDBits has very standardized naming
		info.Confidence = min(info.Confidence*1.1, 1.0)
	}

	return info
}

func (info *TorrentInfo) parseTVInfo(name string, hasDate bool) {
	// Standard episode pattern (S01E01)
	if match := episodePattern.FindStringSubmatch(name); match != nil {
		if seasonMatch := seasonPattern.FindStringSubmatch(name); seasonMatch != nil {
			info.Season, _ = strconv.Atoi(seasonMatch[1])
		}
		ep, _ := strconv.Atoi(match[1])
		info.Episodes = []int{ep}
		return
	}

	// Alternative format (1x01)
	if match := altEpisodePattern.FindStringSubmatch(name); match != nil {
		info.Season, _ = strconv.Atoi(match[1])
		ep, _ := strconv.Atoi(match[2])
		info.Episodes = []int{ep}
		return
	}

	// Date-based episodes (common for daily shows)
	if hasDate {
		// Extract full date for daily shows
		if match := datePattern.FindStringSubmatch(name); match != nil {
			// Store the full date (YYYY.MM.DD format)
			info.Date = fmt.Sprintf("%s.%s.%s", match[1], match[2], match[3])
			// Also set the year for compatibility
			if year, err := strconv.Atoi(match[1]); err == nil && year >= 1895 && year <= getCurrentYear() {
				info.Year = year
			}
		}
		return
	}

	// Season only
	if match := seasonPattern.FindStringSubmatch(name); match != nil {
		info.Season, _ = strconv.Atoi(match[1])
		return
	}

	// Alternative season format
	if match := seasonAltPattern.FindStringSubmatch(name); match != nil {
		info.Season, _ = strconv.Atoi(match[1])
		return
	}
}

func (info *TorrentInfo) parseQuality(name string) {
	// Resolution
	if match := resolutionPattern.FindStringSubmatch(name); match != nil {
		info.Resolution = strings.ToLower(match[1])
		if info.Resolution == "4k" {
			info.Resolution = "2160p"
		}
	}

	// Source
	if match := sourcePattern.FindStringSubmatch(name); match != nil {
		source := match[1]
		// Normalize source names
		switch strings.ToUpper(source) {
		case "BLURAY", "BLU-RAY":
			info.Source = "BluRay"
		case "WEB-DL", "WEBDL":
			info.Source = "WEB-DL"
		case "WEBRIP", "WEB":
			info.Source = "WEBRip"
		default:
			info.Source = strings.ToUpper(source)
		}
	}

	// Codec
	if match := codecPattern.FindStringSubmatch(name); match != nil {
		codec := strings.ToUpper(match[1])
		// Normalize codec names
		switch codec {
		case "H264", "X264", "AVC":
			info.Codec = "H264"
		case "H265", "X265", "HEVC":
			info.Codec = "H265"
		default:
			info.Codec = codec
		}
	}

	// Audio
	if match := audioPattern.FindStringSubmatch(name); match != nil {
		info.Audio = strings.ToUpper(match[1])
	}
}

func (info *TorrentInfo) parseLanguage(name string) {
	// Language
	if match := languagePattern.FindStringSubmatch(name); match != nil {
		info.Language = strings.Title(strings.ToLower(match[1]))
	}

	// Subtitles
	if subsPattern.MatchString(name) {
		// Try to find specific subtitle languages
		subLanguages := regexp.MustCompile(`(?i)(ENG|FRE|SPA|GER|ITA|DAN|DUT|JAP|CHI|RUS|POL|VIE|SWE|NOR|FIN|TUR|POR|KOR)[\.\s]?SUBS`).FindAllStringSubmatch(name, -1)
		for _, match := range subLanguages {
			info.Subtitles = append(info.Subtitles, match[1])
		}

		// If no specific languages found, just note that it has subtitles
		if len(info.Subtitles) == 0 {
			info.Subtitles = []string{"Unknown"}
		}
	}
}

func extractReleaseGroup(name string) string {
	// Remove file extension
	name = regexp.MustCompile(`\.[a-zA-Z0-9]+$`).ReplaceAllString(name, "")

	// Allow for dash-separated group at end, optionally followed by bracketed tags
	pattern := `-([a-zA-Z0-9]+)(\[[^\]]+\])?$` // -GROUP or -GROUP[bracket]

	if match := regexp.MustCompile(pattern).FindStringSubmatch(name); match != nil {
		group := match[1]
		// Validate it looks like a release group (not a quality tag)
		if !isQualityTag(group) && len(group) >= 2 {
			return group
		}
	}

	return ""
}

func extractTitle(name string, info *TorrentInfo) string {
	title := name

	// Find the earliest index of clearly metadata patterns only
	// Don't use status/edition patterns here as they can be part of titles
	indices := []int{}
	patterns := []*regexp.Regexp{
		seasonPattern, seasonAltPattern, episodePattern, altEpisodePattern,
		resolutionPattern, sourcePattern, codecPattern, audioPattern,
		languagePattern, datePattern,
	}

	for _, pat := range patterns {
		if idx := pat.FindStringIndex(title); idx != nil {
			indices = append(indices, idx[0])
		}
	}

	// Also consider the release year position if it's set
	if info.Year > 0 {
		yearStr := strconv.Itoa(info.Year)
		yearRegex := regexp.MustCompile(`\b` + regexp.QuoteMeta(yearStr) + `\b`)
		matches := yearRegex.FindAllStringIndex(title, -1)
		if len(matches) > 0 {
			// Use the last occurrence of the release year (in case there are multiple)
			last := matches[len(matches)-1]
			indices = append(indices, last[0])
		}
	}

	if len(indices) > 0 {
		minIdx := indices[0]
		for _, idx := range indices {
			if idx < minIdx {
				minIdx = idx
			}
		}
		title = title[:minIdx]
	}

	// Clean up the title
	title = cleanString(title)

	return strings.TrimSpace(title)
}

func cleanString(s string) string {
	// Replace dots and underscores with spaces
	s = strings.ReplaceAll(s, ".", " ")
	s = strings.ReplaceAll(s, "_", " ")

	// Remove brackets and their contents (often contains metadata)
	s = regexp.MustCompile(`\[[^\]]+\]`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\([^\)]+\)$`).ReplaceAllString(s, "")

	// Clean up extra spaces
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
}

func isQualityTag(s string) bool {
	qualityTags := []string{
		"1080p", "720p", "480p", "2160p", "4K",
		"BluRay", "WEBRip", "HDTV", "WEB",
		"x264", "x265", "H264", "H265",
		"AAC", "AC3", "DTS", "FLAC",
		"PROPER", "REPACK",
	}

	upper := strings.ToUpper(s)
	for _, tag := range qualityTags {
		if strings.ToUpper(tag) == upper {
			return true
		}
	}
	return false
}

func (info *TorrentInfo) calculateConfidence() {
	conf := 0.0
	// Required fields
	if info.Title != "" {
		conf += 0.4
	}
	if info.Year != 0 {
		conf += 0.2
	}
	if info.Resolution != "" {
		conf += 0.1
	}
	if info.Source != "" {
		conf += 0.1
	}
	if info.Codec != "" {
		conf += 0.1
	}
	if info.ReleaseGroup != "" {
		conf += 0.1
	}
	// Optional fields
	if info.Season != 0 {
		conf += 0.05
	}
	if len(info.Episodes) > 0 {
		conf += 0.05
	}
	if info.Container != "" {
		conf += 0.05
	}
	if info.Language != "" {
		conf += 0.05
	}
	if info.Edition != "" {
		conf += 0.05
	}
	if info.IsComplete || info.IsProper || info.IsRepack || info.IsHardcoded {
		conf += 0.05
	}
	// Clamp to nearest of 1.0, 0.8, 0.4, 0.1
	choices := []float64{1.0, 0.8, 0.4, 0.1}
	closest := choices[0]
	minDiff := 1.0
	for _, c := range choices {
		diff := conf - c
		if diff < 0 {
			diff = -diff
		}
		if diff < minDiff {
			minDiff = diff
			closest = c
		}
	}
	info.Confidence = closest
}

func getCurrentYear() int {
	return time.Now().Year()
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
