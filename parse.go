// Package torrentname provides parsing of torrent names into structured metadata
package torrentname

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Confidence scoring weights
const (
	YearSeasonWeight   = 40
	ResolutionWeight   = 20
	SourceWeight       = 10
	ReleaseGroupWeight = 10
	MinorFieldWeight   = 1
)

// TorrentInfo contains all metadata parsed from a torrent name
type TorrentInfo struct {
	Title        string   `json:"title"`
	Year         int      `json:"year,omitempty"`
	Date         string   `json:"date,omitempty"` // For daily shows (YYYY.MM.DD format)
	Season       int      `json:"season,omitempty"`
	Episode      int      `json:"episode,omitempty"` // Single episode number
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
	Edition      string   `json:"edition,omitempty"`  // Director's Cut, Extended, etc.
	Confidence   int      `json:"confidence"`         // 0 to 100
	Unparsed     string   `json:"unparsed,omitempty"` // Everything after metadata start that isn't metadata
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
	completePattern  = regexp.MustCompile(`(?i)\b(Complete)\b`)
	properPattern    = regexp.MustCompile(`(?i)\b(PROPER)\b`)
	repackPattern    = regexp.MustCompile(`(?i)\b(REPACK)\b`)
	hardcodedPattern = regexp.MustCompile(`(?i)\b(HC|HARDCODED)\b`)

	// Language patterns
	languagePattern = regexp.MustCompile(`(?i)\b(ENGLISH|FRENCH|SPANISH|GERMAN|ITALIAN|DANISH|DUTCH|JAPANESE|CANTONESE|MANDARIN|RUSSIAN|POLISH|VIETNAMESE|SWEDISH|NORWEGIAN|FINNISH|TURKISH|PORTUGUESE|KOREAN|MULTI)\b`)
	subsPattern     = regexp.MustCompile(`(?i)(SUBS|SUBBED|SUB)`)

	// Container patterns
	containerPattern = regexp.MustCompile(`(?i)\.(mkv|mp4|avi|mov|wmv|flv|webm)$`)

	// Release group pattern
	releaseGroupPattern = regexp.MustCompile(`-([a-zA-Z0-9]+)(\[[^\]]+\])?$`)

	// Tracker-specific patterns
	btnSeasonPack     = regexp.MustCompile(`(?i)S(\d{1,2})[\.\s]?Complete`)
	ptnYearRange      = regexp.MustCompile(`(\d{4})-(\d{4})`)
	monoStereoPattern = regexp.MustCompile(`(?i)\b(Mono|Stereo)\b`)
	channelPattern    = regexp.MustCompile(`(?i)\b(1\.0|2\.0|2\.1|3\.0|4\.0|5\.1|6\.0|6\.1|7\.0|7\.1|8\.1|9\.1|10\.2)\b`)
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

	// Extract date early for daily shows (but not year - let metadata boundary detection handle it)
	if match := datePattern.FindString(name); match != "" {
		info.Date = strings.ReplaceAll(match, "-", ".")
		if year, err := strconv.Atoi(match[:4]); err == nil && year >= 1895 && year <= time.Now().Year() {
			info.Year = year
		}
		name = strings.Replace(name, match, "", 1)
	}

	// Find metadata boundary using three-phase approach
	metadataStartPos := findMetadataBoundary(name, info)

	// Extract title using the metadata start position
	info.Title = extractTitleFromPosition(name, metadataStartPos)

	// Extract unparsed content (everything after metadata start that isn't metadata)
	info.Unparsed = extractUnparsedContent(name, metadataStartPos)

	// Calculate confidence based on what we found
	info.calculateConfidence()

	return info
}

// findMetadataBoundary finds all metadata and determines where the title ends
func findMetadataBoundary(name string, info *TorrentInfo) int {
	metadataStartPos := len(name)

	// Phase 1: Definite metadata (back-to-front)
	metadataStartPos = scanDefiniteMetadata(name, info, metadataStartPos)

	// Phase 2: Possible metadata phase 1 (back-to-front, up to current metadata start)
	metadataStartPos = scanPossibleMetadataPhase1(name, info, metadataStartPos)

	// Phase 3: Possible metadata phase 2 (front-to-back, from current metadata start)
	metadataStartPos = scanPossibleMetadataPhase2(name, info, metadataStartPos)

	return metadataStartPos
}

// scanDefiniteMetadata scans for definite metadata from back to front
func scanDefiniteMetadata(name string, info *TorrentInfo, startPos int) int {
	metadataStartPos := startPos

	// Definite metadata patterns
	patterns := []struct {
		pattern *regexp.Regexp
		handler func(string, *TorrentInfo) bool
	}{
		{resolutionPattern, func(match string, info *TorrentInfo) bool {
			if info.Resolution == "" {
				info.Resolution = strings.ToLower(match)
				if info.Resolution == "4k" {
					info.Resolution = "2160p"
				}
				return true
			}
			return false
		}},
		{sourcePattern, func(match string, info *TorrentInfo) bool {
			if info.Source == "" {
				source := match
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
				return true
			}
			return false
		}},
		{codecPattern, func(match string, info *TorrentInfo) bool {
			if info.Codec == "" {
				codec := strings.ToUpper(match)
				// Normalize codec names
				switch codec {
				case "H264", "X264", "AVC":
					info.Codec = "H264"
				case "H265", "X265", "HEVC":
					info.Codec = "H265"
				default:
					info.Codec = codec
				}
				return true
			}
			return false
		}},
		{episodePattern, func(match string, info *TorrentInfo) bool {
			if info.Episode == 0 {
				// Extract season from the same pattern
				if seasonMatch := seasonPattern.FindStringSubmatch(match); seasonMatch != nil {
					info.Season, _ = strconv.Atoi(seasonMatch[1])
				}
				ep, _ := strconv.Atoi(match[strings.LastIndex(match, "E")+1:])
				info.Episode = ep
				return true
			}
			return false
		}},
		{altEpisodePattern, func(match string, info *TorrentInfo) bool {
			if info.Episode == 0 {
				parts := strings.Split(match, "x")
				if len(parts) == 2 {
					info.Season, _ = strconv.Atoi(parts[0])
					ep, _ := strconv.Atoi(parts[1])
					info.Episode = ep
					return true
				}
			}
			return false
		}},
		{seasonPattern, func(match string, info *TorrentInfo) bool {
			if info.Season == 0 {
				info.Season, _ = strconv.Atoi(match[1:])
				return true
			}
			return false
		}},
		{seasonAltPattern, func(match string, info *TorrentInfo) bool {
			if info.Season == 0 {
				info.Season, _ = strconv.Atoi(match[strings.Index(match, "n")+1:])
				return true
			}
			return false
		}},
		{datePattern, func(match string, info *TorrentInfo) bool {
			if info.Date == "" {
				// Store the full date (YYYY.MM.DD format)
				info.Date = strings.ReplaceAll(match, "-", ".")
				// Also set the year for compatibility
				if year, err := strconv.Atoi(match[:4]); err == nil && year >= 1895 && year <= time.Now().Year() {
					info.Year = year
				}
				return true
			}
			return false
		}},
		{btnSeasonPack, func(match string, info *TorrentInfo) bool {
			if info.Season == 0 && !info.IsComplete {
				if submatch := btnSeasonPack.FindStringSubmatch(match); submatch != nil {
					info.Season, _ = strconv.Atoi(submatch[1])
					info.IsComplete = true
					return true
				}
			}
			return false
		}},
	}

	// Find all matches and sort by position (descending for back-to-front scan)
	var matches []struct {
		start, end int
		pattern    int
	}

	for i, p := range patterns {
		allMatches := p.pattern.FindAllStringIndex(name, -1)
		for _, match := range allMatches {
			matches = append(matches, struct {
				start, end int
				pattern    int
			}{match[0], match[1], i})
		}
	}

	// Sort by start position (descending for back-to-front scan)
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[i].start < matches[j].start {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	// Process matches from end to beginning
	for _, match := range matches {
		if match.start >= metadataStartPos {
			continue // Skip if already past our metadata start
		}

		matchText := name[match.start:match.end]
		if patterns[match.pattern].handler(matchText, info) {
			// New metadata found, update start position
			metadataStartPos = match.start
		} else {
			// Duplicate metadata found, terminate scan
			break
		}
	}

	return metadataStartPos
}

// scanPossibleMetadataPhase1 scans for possible metadata from back to front, up to current metadata start
func scanPossibleMetadataPhase1(name string, info *TorrentInfo, startPos int) int {
	metadataStartPos := startPos

	// Debug: Print metadata boundary at start of step 2
	println("DEBUG: Step 2 start - metadata boundary at position:", metadataStartPos, "in:", name)
	if metadataStartPos < len(name) {
		println("DEBUG: Text after boundary:", name[metadataStartPos:])
	} else {
		println("DEBUG: No text after boundary")
	}

	// Temporary slice to collect audio tokens in scan order
	audioTokens := []string{}

	// All possible metadata patterns (including non-extending metadata like audio)
	patterns := []struct {
		pattern *regexp.Regexp
		handler func(string, *TorrentInfo) bool
		isAudio bool // new: mark if this is an audio pattern
	}{
		{yearPattern, func(match string, info *TorrentInfo) bool {
			if info.Year == 0 {
				if year, err := strconv.Atoi(match); err == nil && year >= 1895 && year <= time.Now().Year() {
					info.Year = year
					return true
				}
			}
			return false
		}, false},
		{editionPattern, func(match string, info *TorrentInfo) bool {
			if info.Edition == "" {
				// Normalize multi-word editions by replacing dots with spaces
				norm := strings.ReplaceAll(match, ".", " ")
				info.Edition = strings.Title(strings.ToLower(norm))
				return true
			}
			return false
		}, false},
		{completePattern, func(match string, info *TorrentInfo) bool {
			if !info.IsComplete {
				info.IsComplete = true
				return true
			}
			return false
		}, false},
		{properPattern, func(match string, info *TorrentInfo) bool {
			if !info.IsProper {
				info.IsProper = true
				return true
			}
			return false
		}, false},
		{repackPattern, func(match string, info *TorrentInfo) bool {
			if !info.IsRepack {
				info.IsRepack = true
				return true
			}
			return false
		}, false},
		{hardcodedPattern, func(match string, info *TorrentInfo) bool {
			if !info.IsHardcoded {
				info.IsHardcoded = true
				return true
			}
			return false
		}, false},
		{languagePattern, func(match string, info *TorrentInfo) bool {
			if info.Language == "" {
				info.Language = strings.Title(strings.ToLower(match))
				return true
			}
			return false
		}, false},
		{subsPattern, func(match string, info *TorrentInfo) bool {
			if len(info.Subtitles) == 0 {
				// Try to find specific subtitle languages
				subLanguages := regexp.MustCompile(`(?i)(ENG|FRE|SPA|GER|ITA|DAN|DUT|JAP|CHI|RUS|POL|VIE|SWE|NOR|FIN|TUR|POR|KOR)[\.\s]?SUBS`).FindAllStringSubmatch(match, -1)
				for _, submatch := range subLanguages {
					info.Subtitles = append(info.Subtitles, submatch[1])
				}

				// If no specific languages found, just note that it has subtitles
				if len(info.Subtitles) == 0 {
					info.Subtitles = []string{"Unknown"}
				}
				return true
			}
			return false
		}, false},
		{releaseGroupPattern, func(match string, info *TorrentInfo) bool {
			if info.ReleaseGroup == "" {
				if submatch := releaseGroupPattern.FindStringSubmatch(match); submatch != nil {
					group := submatch[1]
					if !isQualityTag(group) && len(group) >= 2 {
						info.ReleaseGroup = group
						return true
					}
				}
			}
			return false
		}, false},
		{monoStereoPattern, func(match string, info *TorrentInfo) bool {
			// audioTokens handled outside
			return true
		}, true},
		{channelPattern, func(match string, info *TorrentInfo) bool {
			// audioTokens handled outside
			return true
		}, true},
		{audioPattern, func(match string, info *TorrentInfo) bool {
			// audioTokens handled outside
			return true
		}, true},
		{regexp.MustCompile(`(?i)\b(ATMOS|DTS-X|DTS-HD|DTS-HD MA|DTS-ES|DD\+|DD|EAC3)\b`), func(match string, info *TorrentInfo) bool {
			// audioTokens handled outside
			return true
		}, true},
	}

	// Find all matches and sort by position (descending for back-to-front scan)
	var matches []struct {
		start, end int
		pattern    int
	}

	for i, p := range patterns {
		allMatches := p.pattern.FindAllStringIndex(name, -1)
		for _, match := range allMatches {
			matches = append(matches, struct {
				start, end int
				pattern    int
			}{match[0], match[1], i})
		}
	}

	// Sort by start position (descending for back-to-front scan)
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[i].start < matches[j].start {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	// Process matches from end to beginning, up to current metadata start
	for _, match := range matches {
		if match.start < metadataStartPos {
			break // Skip if before our metadata start - all subsequent matches will also be before
		}

		matchText := name[match.start:match.end]
		if patterns[match.pattern].isAudio {
			audioTokens = append(audioTokens, strings.ToUpper(matchText))
		}
		if patterns[match.pattern].handler(matchText, info) {
			// New metadata found, but don't update start position in step 2
		} else {
			// Duplicate metadata found, terminate scan
			break
		}
	}

	// After scan, reverse audioTokens and join
	if len(audioTokens) > 0 {
		for i, j := 0, len(audioTokens)-1; i < j; i, j = i+1, j-1 {
			audioTokens[i], audioTokens[j] = audioTokens[j], audioTokens[i]
		}
		info.Audio = strings.Join(audioTokens, " ")
	}

	return metadataStartPos
}

// scanPossibleMetadataPhase2 scans for possible metadata from current metadata start towards beginning
func scanPossibleMetadataPhase2(name string, info *TorrentInfo, startPos int) int {
	metadataStartPos := startPos

	// Extending metadata patterns (can be found in step 3)
	// These are metadata that can extend the title boundary backwards
	patterns := []struct {
		pattern *regexp.Regexp
		handler func(string, *TorrentInfo) bool
	}{
		{yearPattern, func(match string, info *TorrentInfo) bool {
			if info.Year == 0 {
				if year, err := strconv.Atoi(match); err == nil && year >= 1895 && year <= time.Now().Year() {
					info.Year = year
					return true
				}
			}
			return false
		}},
		{editionPattern, func(match string, info *TorrentInfo) bool {
			if info.Edition == "" {
				// Normalize multi-word editions by replacing dots with spaces
				norm := strings.ReplaceAll(match, ".", " ")
				info.Edition = strings.Title(strings.ToLower(norm))
				return true
			}
			return false
		}},
		{completePattern, func(match string, info *TorrentInfo) bool {
			if !info.IsComplete {
				info.IsComplete = true
				return true
			}
			return false
		}},
		{properPattern, func(match string, info *TorrentInfo) bool {
			if !info.IsProper {
				info.IsProper = true
				return true
			}
			return false
		}},
		{repackPattern, func(match string, info *TorrentInfo) bool {
			if !info.IsRepack {
				info.IsRepack = true
				return true
			}
			return false
		}},
		{hardcodedPattern, func(match string, info *TorrentInfo) bool {
			if !info.IsHardcoded {
				info.IsHardcoded = true
				return true
			}
			return false
		}},
		{languagePattern, func(match string, info *TorrentInfo) bool {
			if info.Language == "" {
				info.Language = strings.Title(strings.ToLower(match))
				return true
			}
			return false
		}},
		{subsPattern, func(match string, info *TorrentInfo) bool {
			if len(info.Subtitles) == 0 {
				// Try to find specific subtitle languages
				subLanguages := regexp.MustCompile(`(?i)(ENG|FRE|SPA|GER|ITA|DAN|DUT|JAP|CHI|RUS|POL|VIE|SWE|NOR|FIN|TUR|POR|KOR)[\.\s]?SUBS`).FindAllStringSubmatch(match, -1)
				for _, submatch := range subLanguages {
					info.Subtitles = append(info.Subtitles, submatch[1])
				}

				// If no specific languages found, just note that it has subtitles
				if len(info.Subtitles) == 0 {
					info.Subtitles = []string{"Unknown"}
				}
				return true
			}
			return false
		}},
		{releaseGroupPattern, func(match string, info *TorrentInfo) bool {
			if info.ReleaseGroup == "" {
				if submatch := releaseGroupPattern.FindStringSubmatch(match); submatch != nil {
					group := submatch[1]
					if !isQualityTag(group) && len(group) >= 2 {
						info.ReleaseGroup = group
						return true
					}
				}
			}
			return false
		}},
	}

	// Find all matches and sort by position (descending for back-to-front scan)
	var matches []struct {
		start, end int
		pattern    int
	}

	for i, p := range patterns {
		allMatches := p.pattern.FindAllStringIndex(name, -1)
		for _, match := range allMatches {
			matches = append(matches, struct {
				start, end int
				pattern    int
			}{match[0], match[1], i})
		}
	}

	// Sort by start position (descending for back-to-front scan)
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[i].start < matches[j].start {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	// Process matches from current metadata start towards beginning (scanning backwards)
	for _, match := range matches {
		if match.start >= metadataStartPos {
			continue // Skip if already past our metadata start
		}

		// Don't consume the first word of the entire name as metadata
		if match.start == 0 {
			break
		}

		// Check if this metadata is adjacent to current metadata start
		if !isAdjacentToMetadataStart(match.start, match.end, metadataStartPos, name) {
			break // Not adjacent, exit scan
		}

		matchText := name[match.start:match.end]
		if patterns[match.pattern].handler(matchText, info) {
			// New metadata found, update start position
			metadataStartPos = match.start
		} else {
			// Duplicate metadata found, terminate scan
			break
		}
	}

	return metadataStartPos
}

// isAdjacentToMetadataStart checks if a metadata position is adjacent to the current metadata start
func isAdjacentToMetadataStart(start, end, metadataStartPos int, name string) bool {
	// If this metadata ends at the metadata start position, it's adjacent
	if end == metadataStartPos {
		return true
	}

	// If this metadata starts at the metadata start position, it's adjacent
	if start == metadataStartPos {
		return true
	}

	// Check if there are only separators between this metadata and the metadata start
	if end < metadataStartPos {
		// This metadata comes before the metadata start
		between := name[end:metadataStartPos]
		return isOnlySeparators(between)
	} else if start > metadataStartPos {
		// This metadata comes after the metadata start (shouldn't happen in phase 2)
		between := name[metadataStartPos:start]
		return isOnlySeparators(between)
	}

	return false
}

// isOnlySeparators returns true if the string contains only separator characters
func isOnlySeparators(s string) bool {
	for _, c := range s {
		if c != '.' && c != ' ' && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

// extractUnparsedContent extracts everything after metadata start that isn't metadata
func extractUnparsedContent(name string, metadataStartPos int) string {
	if metadataStartPos >= len(name) {
		return ""
	}

	afterMetadata := name[metadataStartPos:]

	// Find all metadata patterns in the remaining text
	metadataPatterns := []*regexp.Regexp{
		resolutionPattern, sourcePattern, codecPattern, audioPattern,
		languagePattern, completePattern, properPattern, repackPattern, hardcodedPattern,
		editionPattern, yearPattern, releaseGroupPattern,
		seasonPattern, seasonAltPattern, episodePattern, altEpisodePattern,
		monoStereoPattern, channelPattern,
		// Audio channel enhancements
		regexp.MustCompile(`(?i)\b(ATMOS|DTS-X|DTS-HD|DTS-HD MA|DTS-ES|DD\+|DD|EAC3)\b`),
		// Date component patterns
		regexp.MustCompile(`(?i)\b\d{1,2}\.\d{1,2}\b`), // 10.15, 12.25, etc.
	}

	// Remove all metadata from the unparsed content
	result := afterMetadata
	for _, pattern := range metadataPatterns {
		result = pattern.ReplaceAllString(result, "")
	}

	// Remove leftover episode-only codes like E01, E02, etc.
	result = regexp.MustCompile(`(?i)\bE\d{1,3}\b`).ReplaceAllString(result, "")

	// Clean up extra spaces and separators
	result = strings.ReplaceAll(result, ".", " ")
	result = strings.ReplaceAll(result, "-", " ")
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

	return strings.TrimSpace(result)
}

// isReasonableYear checks if a string is a reasonable year
func isReasonableYear(s string) bool {
	if year, err := strconv.Atoi(s); err == nil {
		return year >= 1895 && year <= time.Now().Year()
	}
	return false
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
		if info.Confidence*11 < 100 {
			info.Confidence = info.Confidence * 11 / 10
		} else {
			info.Confidence = 100
		}
	}

	return info
}

func extractTitle(name string, info *TorrentInfo) string {
	// For backward compatibility, compute metadata start position
	// Find the earliest position of "safe" metadata patterns
	safePatterns := []*regexp.Regexp{
		resolutionPattern, sourcePattern, codecPattern, audioPattern,
		seasonPattern, seasonAltPattern, episodePattern, altEpisodePattern,
		languagePattern, datePattern,
	}

	earliestPos := -1
	for _, pat := range safePatterns {
		if match := pat.FindStringIndex(name); match != nil {
			if earliestPos == -1 || match[0] < earliestPos {
				earliestPos = match[0]
			}
		}
	}

	// Also consider release year position
	yearMatches := yearPattern.FindAllStringIndex(name, -1)
	if len(yearMatches) > 0 {
		// Use the last year as release year (most likely to be the actual release year)
		releaseYearPos := yearMatches[len(yearMatches)-1][0]
		if earliestPos == -1 || releaseYearPos < earliestPos {
			earliestPos = releaseYearPos
		}
	}

	return extractTitleFromPosition(name, earliestPos)
}

func extractTitleFromPosition(name string, metadataStartPos int) string {
	title := name
	if metadataStartPos >= 0 {
		title = title[:metadataStartPos]
	}
	// Trim trailing separators (dot, space, dash, underscore)
	title = strings.TrimRight(title, ". -_")
	return strings.TrimSpace(cleanString(title))
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
	conf := 0
	// Year or Season (or both)
	if info.Year != 0 || info.Season != 0 {
		conf += YearSeasonWeight
	}
	// Resolution
	if info.Resolution != "" {
		conf += ResolutionWeight
	}
	// Source
	if info.Source != "" {
		conf += SourceWeight
	}
	// ReleaseGroup
	if info.ReleaseGroup != "" {
		conf += ReleaseGroupWeight
	}
	// Minor fields (1 point each)
	if info.Episode != 0 {
		conf += MinorFieldWeight
	}
	if info.Codec != "" {
		conf += MinorFieldWeight
	}
	if info.Audio != "" {
		conf += MinorFieldWeight
	}
	if info.Container != "" {
		conf += MinorFieldWeight
	}
	if info.Language != "" {
		conf += MinorFieldWeight
	}
	if info.Edition != "" {
		conf += MinorFieldWeight
	}
	if info.IsComplete {
		conf += MinorFieldWeight
	}
	if info.IsProper {
		conf += MinorFieldWeight
	}
	if info.IsRepack {
		conf += MinorFieldWeight
	}
	if info.IsHardcoded {
		conf += MinorFieldWeight
	}

	// Cap at 100
	if conf > 100 {
		conf = 100
	}
	info.Confidence = conf
}

// NormalizeTitle removes common variations for matching
func NormalizeTitle(title string) string {
	// Replace all non-alphanumeric characters with spaces
	title = regexp.MustCompile(`[^a-zA-Z0-9\s]`).ReplaceAllString(title, " ")

	// Convert to lowercase and split into words
	words := strings.Fields(strings.ToLower(title))

	// Remove common words
	commonWords := map[string]bool{"the": true, "a": true, "an": true, "and": true, "or": true, "of": true}
	filtered := []string{}
	for _, word := range words {
		if !commonWords[word] {
			filtered = append(filtered, word)
		}
	}

	return strings.Join(filtered, " ")
}

// Recommended threshold for title matching using Dice coefficient.
// Titles with similarity >= this value are considered a match.
const TitleMatchThreshold = 0.8

// MatchTitles checks if two titles likely refer to the same content.
// Uses Dice coefficient for similarity and TitleMatchThreshold as the default threshold for a match.
func MatchTitles(title1, title2 string, threshold float64) bool {
	norm1 := NormalizeTitle(title1)
	norm2 := NormalizeTitle(title2)

	// Exact match after normalization
	if norm1 == norm2 {
		return true
	}

	// Calculate similarity ratio (Dice coefficient)
	similarity := calculateSimilarity(norm1, norm2)
	return similarity >= threshold
}

// Simple similarity calculation (Dice coefficient)
func calculateSimilarity(s1, s2 string) float64 {
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)

	// Create sets
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, w := range words1 {
		set1[w] = true
	}
	for _, w := range words2 {
		set2[w] = true
	}

	// Calculate intersection
	intersection := 0
	for w := range set1 {
		if set2[w] {
			intersection++
		}
	}

	// Use Dice coefficient: 2*intersection/(len1+len2)
	total := len(set1) + len(set2)
	if total == 0 {
		return 0
	}

	return 2.0 * float64(intersection) / float64(total)
}
