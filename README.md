# torrentname

[![Go Reference](https://pkg.go.dev/badge/github.com/cehbz/torrentname.svg)](https://pkg.go.dev/github.com/cehbz/torrentname)
[![Go Report Card](https://goreportcard.com/badge/github.com/cehbz/torrentname)](https://goreportcard.com/report/github.com/cehbz/torrentname)

A Go library for parsing torrent names and extracting metadata. Supports movies, TV shows, and various quality/source formats commonly used by release groups.

## Features

- **Comprehensive parsing** of torrent names into structured data
- **Movie support**: Title, year, quality, source, codec, audio format
- **TV show support**: Series name, season, episode(s), complete packs
- **Quality detection**: Resolution (480p-2160p/4K), source (BluRay, WEB-DL, etc), codec (x264, x265/HEVC)
- **Additional metadata**: Release group, edition (Extended, Director's Cut), language, subtitles
- **Status flags**: PROPER, REPACK, COMPLETE, HARDCODED
- **Tracker hints**: Special handling for BTN, PTP, HDB formats
- **Confidence scoring**: Indicates parsing reliability

## Installation

```bash
go get github.com/cehbz/torrentname
```

## Usage

### Basic Parsing

```go
package main

import (
    "fmt"
    "github.com/cehbz/torrentname"
)

func main() {
    info := torrentname.Parse("The.Matrix.1999.1080p.BluRay.x264-SPARKS")
    
    fmt.Printf("Title: %s\n", info.Title)           // The Matrix
    fmt.Printf("Year: %d\n", info.Year)             // 1999
    fmt.Printf("Resolution: %s\n", info.Resolution) // 1080p
    fmt.Printf("Source: %s\n", info.Source)         // BluRay
    fmt.Printf("Codec: %s\n", info.Codec)           // H264
    fmt.Printf("Group: %s\n", info.ReleaseGroup)    // SPARKS
}
```

### TV Show Parsing

```go
// Single episode
info := torrentname.Parse("Breaking.Bad.S01E01.Pilot.1080p.BluRay.x264-ROVERS")
fmt.Printf("Title: %s\n", info.Title)       // Breaking Bad
fmt.Printf("Season: %d\n", info.Season)     // 1
fmt.Printf("Episodes: %v\n", info.Episodes) // [1]

// Multi-episode
info = torrentname.Parse("The.Wire.S01E01-E03.720p.HDTV.x264")
fmt.Printf("Episodes: %v\n", info.Episodes) // [1 2 3]

// Complete season
info = torrentname.Parse("Game.of.Thrones.S08.COMPLETE.1080p.BluRay.x264")
fmt.Printf("Complete: %v\n", info.IsComplete) // true
```

### Tracker-Specific Parsing

Some trackers have unique naming conventions. Use `ParseWithHints` for better accuracy:

```go
// BTN uses "S01 Complete" format
info := torrentname.ParseWithHints(
    "Breaking.Bad.S01.Complete.720p.BluRay.x264-DEMAND",
    "BTN",
)
fmt.Printf("Complete: %v\n", info.IsComplete) // true

// HDBits entries get higher confidence scores
info = torrentname.ParseWithHints(
    "The.Dark.Knight.2008.1080p.BluRay.DTS.x264-ESiR",
    "HDBits",
)
fmt.Printf("Confidence: %.1f%%\n", info.Confidence * 100) // 100.0%
```

### Extended Information

```go
info := torrentname.Parse("The.Lord.of.the.Rings.2001.EXTENDED.1080p.BluRay.x265")
fmt.Printf("Edition: %s\n", info.Edition) // Extended

info = torrentname.Parse("Parasite.2019.KOREAN.1080p.BluRay.x264.DTS-FGT")
fmt.Printf("Language: %s\n", info.Language) // Korean
fmt.Printf("Audio: %s\n", info.Audio)       // DTS

info = torrentname.Parse("Movie.Title.2020.1080p.HC.WEBRip.SUBS")
fmt.Printf("Hardcoded: %v\n", info.IsHardcoded)     // true
fmt.Printf("Subtitles: %v\n", len(info.Subtitles))  // 1
```

## Supported Formats

### Video Quality
- **Resolution**: 2160p, 4K, 1080p, 720p, 480p, 360p
- **Source**: BluRay, WEB-DL, WEBRip, HDTV, DVDRip, CAM, TS, TC, SCR
- **Codec**: x264, H264, x265, H265, HEVC, AVC, MPEG4, DIVX, XVID, VP9, AV1

### Audio
- DTS-HD, DTS, TrueHD, Atmos, DD+, DD, EAC3, AC3, AAC, FLAC, MP3

### Special Editions
- Director's Cut, Extended, Unrated, Remastered, Theatrical, Ultimate Edition, Special Edition

### Languages
- English, French, Spanish, German, Italian, Danish, Dutch, Japanese, Cantonese, Mandarin, Russian, Polish, Vietnamese, Swedish, Norwegian, Finnish, Turkish, Portuguese, Multi

## Data Structure

```go
type TorrentInfo struct {
    Title        string   // Clean title without metadata
    Year         int      // Release year (movies) or series start year
    Season       int      // Season number (0 if not applicable)
    Episodes     []int    // Episode numbers (empty for movies)
    Resolution   string   // 2160p, 1080p, 720p, etc.
    Source       string   // BluRay, WEB-DL, HDTV, etc.
    Codec        string   // H264, H265, etc.
    Audio        string   // DTS, AC3, AAC, etc.
    ReleaseGroup string   // Release group name
    Container    string   // mkv, mp4, avi, etc.
    Language     string   // Primary language
    Subtitles    []string // Subtitle languages
    IsComplete   bool     // Complete season/series pack
    IsProper     bool     // PROPER release
    IsRepack     bool     // REPACK release  
    IsHardcoded  bool     // Hardcoded subtitles
    Edition      string   // Special edition info
    Confidence   float64  // Parsing confidence (0.0-1.0)
}
```

## Confidence Score

The parser assigns a confidence score (0.0-1.0) based on how much metadata was successfully extracted:

- **1.0**: Full metadata (title, year/season, quality, source, group)
- **0.8**: Most metadata present
- **0.6**: Basic metadata (title + some quality info)
- **0.4**: Minimal metadata (title only)

## Running the Example

```bash
# Run with default examples
go run example/main.go

# Parse specific torrent names
go run example/main.go "The.Matrix.1999.1080p.BluRay.x264" "Breaking.Bad.S01E01.720p"
```

## Testing

```bash
go test -v
go test -bench=.
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. Areas for improvement:

- Additional tracker-specific formats
- More language detection
- Better handling of anime naming conventions
- Support for more quality/source formats

## License

MIT License - see LICENSE file for details
