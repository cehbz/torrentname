// Package torrentname provides parsing of torrent names into structured metadata
package torrentname

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TorrentInfo contains all metadata parsed from a torrent name
type TorrentInfo struct {
	Title        string   `json:"title"`
	Year         int      `json:"year,omitempty"`
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
	yearPattern         = regexp.MustCompile(`[\.\s\(](\d{4})[\.\s\)]`)
	seasonPattern       = regexp.MustCompile(`(?i)S(\d{1,2})`)
	seasonAltPattern    = regexp.MustCompile(`(?i)Season[\.\s]?(\d{1,2})`)
	episodePattern      = regexp.MustCompile(`(?i)S\d{1,2}E(\d{1,3})`)
	episodeOnlyPattern  = regexp.MustCompile(`(?i)E(\d{1,3})`)
	multiEpisodePattern = regexp.MustCompile(`(?i)S\d{1,2}E(\d{1,3})(?:-?E?(\d{1,3}))?`)
	altEpisodePattern   = regexp.MustCompile(`(?i)(\d{1,2})x(\d{1,3})`)
	datePattern         = regexp.MustCompile(`(\d{4})[\.\-](\d{2})[\.\-](\d{2})`)
	
	// Quality patterns
	resolutionPattern = regexp.MustCompile(`(?i)(2160p|4K|1080p|720p|480p|360p)`)
	sourcePattern     = regexp.MustCompile(`(?i)(BluRay|Blu-Ray|BDRip|BRRip|WEB-DL|WEBDL|WEBRip|WEB|HDTV|HDRip|DVDRip|DVD|CAM|TS|TC|SCR|DVDSCR|HC|HDCAM)`)
	codecPattern      = regexp.MustCompile(`(?i)(x264|h264|x265|h265|HEVC|AVC|MPEG4|DIVX|XVID|VP9|AV1)`)
	audioPattern      = regexp.MustCompile(`(?i)(DTS-HD|DTS|TrueHD|Atmos|DD\+?|EAC3|AC3|AAC|FLAC|MP3)`)
	
	// Edition patterns
	editionPattern = regexp.MustCompile(`(?i)(Directors?[\.\s]?Cut|Extended|Unrated|Remastered|Theatrical|Ultimate[\.\s]?Edition|Special[\.\s]?Edition|Collectors?[\.\s]?Edition|International|Criterion)`)
	
	// Status patterns
	completePattern  = regexp.MustCompile(`(?i)(Complete|COMPLETE)`)
	properPattern    = regexp.MustCompile(`(?i)(PROPER|REPACK)`)
	hardcodedPattern = regexp.MustCompile(`(?i)(HC|HARDCODED)`)
	
	// Language patterns
	languagePattern = regexp.MustCompile(`(?i)(ENGLISH|FRENCH|SPANISH|GERMAN|ITALIAN|DANISH|DUTCH|JAPANESE|CANTONESE|MANDARIN|RUSSIAN|POLISH|VIETNAMESE|SWEDISH|NORWEGIAN|FINNISH|TURKISH|PORTUGUESE|MULTI)`)
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
	if match := containerPattern.FindStringSubmatch(name); match != nil {
		info.Container = strings.ToLower(match[1])
		// Remove extension for further parsing
		name = name[:strings.LastIndex(name, match[0])]
	}
	
	// Extract year
	if match := yearPattern.FindStringSubmatch(name); match != nil {
		info.Year, _ = strconv.Atoi(match[1])
	}
	
	// Check for date-based episodes (common for daily shows)
	dateMatch := datePattern.FindStringSubmatch(name)
	
	// Extract season and episodes
	info.parseTVInfo(name, dateMatch != nil)
	
	// Extract quality information
	info.parseQuality(name)
	
	// Extract edition
	if match := editionPattern.FindStringSubmatch(name); match != nil {
		info.Edition = cleanString(match[1])
	}
	
	// Extract status flags
	info.IsComplete = completePattern.MatchString(name) || btnSeasonPack.MatchString(name)
	info.IsProper = properPattern.MatchString(name)
	info.IsRepack = strings.Contains(strings.ToUpper(name), "REPACK")
	info.IsHardcoded = hardcodedPattern.MatchString(name)
	
	// Extract language and subtitles
	info.parseLanguage(name)
	
	// Extract release group (usually at the end)
	info.ReleaseGroup = extractReleaseGroup(name)
	
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
	// Multi-episode pattern (S01E01-E05 or S01E01-05)
	if match := multiEpisodePattern.FindStringSubmatch(name); match != nil {
		if seasonMatch := seasonPattern.FindStringSubmatch(name); seasonMatch != nil {
			info.Season, _ = strconv.Atoi(seasonMatch[1])
		}
		
		start, _ := strconv.Atoi(match[1])
		end := start
		if match[2] != "" {
			end, _ = strconv.Atoi(match[2])
		}
		
		for i := start; i <= end && i < start+100; i++ { // Sanity limit
			info.Episodes = append(info.Episodes, i)
		}
		return
	}
	
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
		info.Resolution = strings.ToUpper(match[1])
		if info.Resolution == "4K" {
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
		subLanguages := regexp.MustCompile(`(?i)(ENG|FRE|SPA|GER|ITA|DAN|DUT|JAP|CHI|RUS|POL|VIE|SWE|NOR|FIN|TUR|POR)[\.\s]?SUBS`).FindAllStringSubmatch(name, -1)
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
	
	// Common patterns for release groups
	patterns := []string{
		`-([a-zA-Z0-9]+)$`,              // -GROUP at end
		`\[([a-zA-Z0-9]+)\]$`,           // [GROUP] at end
		`\.([a-zA-Z0-9]+)$`,             // .GROUP at end
		`\s([a-zA-Z0-9]+)$`,             // GROUP at end
	}
	
	for _, pattern := range patterns {
		if match := regexp.MustCompile(pattern).FindStringSubmatch(name); match != nil {
			group := match[1]
			// Validate it looks like a release group (not a quality tag)
			if !isQualityTag(group) && len(group) >= 2 {
				return group
			}
		}
	}
	
	return ""
}

func extractTitle(name string, info *TorrentInfo) string {
	title := name
	
	// Remove everything after year if found
	if info.Year > 0 && yearPattern.MatchString(title) {
		if idx := yearPattern.FindStringIndex(title); idx != nil {
			title = title[:idx[0]]
		}
	}
	
	// Remove TV info
	title = seasonPattern.ReplaceAllString(title, "")
	title = episodePattern.ReplaceAllString(title, "")
	title = altEpisodePattern.ReplaceAllString(title, "")
	
	// Remove quality info
	title = resolutionPattern.ReplaceAllString(title, "")
	title = sourcePattern.ReplaceAllString(title, "")
	title = codecPattern.ReplaceAllString(title, "")
	title = audioPattern.ReplaceAllString(title, "")
	
	// Remove status info
	title = completePattern.ReplaceAllString(title, "")
	title = properPattern.ReplaceAllString(title, "")
	
	// Remove release group if found
	if info.ReleaseGroup != "" {
		title = regexp.MustCompile(`[\.\-\s\[]`+regexp.QuoteMeta(info.ReleaseGroup)+`[\]\s]*$`).ReplaceAllString(title, "")
	}
	
	// Clean up
	return cleanString(title)
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
	points := 0.0
	maxPoints := 0.0
	
	// Title (required)
	maxPoints += 2.0
	if info.Title != "" {
		points += 2.0
	}
	
	// Year (important for movies)
	maxPoints += 1.0
	if info.Year > 1900 && info.Year <= getCurrentYear() {
		points += 1.0
	}
	
	// TV info (important for series)
	if info.Season > 0 || len(info.Episodes) > 0 {
		maxPoints += 1.0
		points += 1.0
	}
	
	// Quality info
	maxPoints += 1.0
	if info.Resolution != "" || info.Source != "" {
		points += 1.0
	}
	
	// Release group
	maxPoints += 0.5
	if info.ReleaseGroup != "" {
		points += 0.5
	}
	
	info.Confidence = points / maxPoints
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
