package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cehbz/torrentname"
)

func main() {
	// Example torrent names
	examples := []string{
		"The.Matrix.1999.1080p.BluRay.x264-SPARKS",
		"Breaking.Bad.S01E01.Pilot.1080p.BluRay.x264-ROVERS",
		"Game.of.Thrones.S08.COMPLETE.1080p.BluRay.x264-ROVERS[rartv]",
		"The.Mandalorian.S02E01-E08.2160p.WEB-DL.DDP5.1.Atmos.HDR.HEVC-MZABI",
		"Parasite.2019.KOREAN.1080p.BluRay.x264.DTS-FGT",
		"The.Lord.of.the.Rings.The.Fellowship.of.the.Ring.2001.EXTENDED.1080p.BluRay.x265-RARBG.mkv",
	}

	// Check if arguments provided
	if len(os.Args) > 1 {
		examples = os.Args[1:]
	}

	fmt.Println("Torrent Name Parser Examples")
	fmt.Println("============================")

	for _, name := range examples {
		fmt.Printf("\nParsing: %s\n", name)
		fmt.Println(strings.Repeat("-", len(name)+9))

		// Parse the torrent name
		info := torrentname.Parse(name)

		// Display results
		displayInfo(info)

		// Show confidence calculation logic
		fmt.Printf("  (Confidence is calculated as: +40 for Year/Season, +20 for Resolution, +10 for Source, +10 for ReleaseGroup, +1 for each minor field)\n")

		// Example with tracker hints
		if len(os.Args) == 1 {
			fmt.Println("\nWith BTN hint:")
			infoWithHint := torrentname.ParseWithHints(name, "BTN")
			if infoWithHint.IsComplete && !info.IsComplete {
				fmt.Println("  ✓ Detected as complete season pack")
			}
		}
	}

	// JSON output example
	if len(os.Args) == 1 {
		fmt.Println("\n\nJSON Output Example:")
		fmt.Println("===================")
		sampleInfo := torrentname.Parse("The.Matrix.1999.1080p.BluRay.x264-SPARKS")
		jsonData, err := json.MarshalIndent(sampleInfo, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(jsonData))

		// Title normalization and matching examples
		fmt.Println("\n\nTitle Normalization and Matching Examples:")
		fmt.Println("==========================================")

		// Normalization replaces all non-alphanumeric characters with spaces, removes common words, and collapses whitespace.
		title1 := "The.Matrix.1999.1080p.BluRay.x264-SPARKS"
		title2 := "The Matrix"
		title3 := "Matrix Reloaded"

		normalized1 := torrentname.NormalizeTitle(title1)
		normalized2 := torrentname.NormalizeTitle(title2)
		normalized3 := torrentname.NormalizeTitle(title3)

		fmt.Printf("Original: %q\n", title1)
		fmt.Printf("Normalized: %q\n", normalized1)
		fmt.Printf("Original: %q\n", title2)
		fmt.Printf("Normalized: %q\n", normalized2)
		fmt.Printf("Original: %q\n", title3)
		fmt.Printf("Normalized: %q\n", normalized3)

		// Similarity uses the Dice coefficient; default threshold is 0.8
		fmt.Printf("\nMatching %q and %q (threshold 0.8): %v\n", title1, title2, torrentname.MatchTitles(title1, title2, 0.8))
		fmt.Printf("Matching %q and %q (threshold 0.8): %v\n", title1, title3, torrentname.MatchTitles(title1, title3, 0.8))
		fmt.Printf("Matching %q and %q (threshold 0.3): %v\n", title1, title3, torrentname.MatchTitles(title1, title3, 0.3))
	}
}

func displayInfo(info *torrentname.TorrentInfo) {
	fmt.Printf("  Title:         %s\n", info.Title)

	if info.Year > 0 {
		fmt.Printf("  Year:          %d\n", info.Year)
	}

	if info.Season > 0 {
		fmt.Printf("  Season:        %d\n", info.Season)
	}

	if info.Episode > 0 {
		fmt.Printf("  Episode:       %d\n", info.Episode)
	}

	if info.Resolution != "" {
		fmt.Printf("  Resolution:    %s\n", info.Resolution)
	}

	if info.Source != "" {
		fmt.Printf("  Source:        %s\n", info.Source)
	}

	if info.Codec != "" {
		fmt.Printf("  Codec:         %s\n", info.Codec)
	}

	if info.Audio != "" {
		fmt.Printf("  Audio:         %s\n", info.Audio)
	}

	if info.ReleaseGroup != "" {
		fmt.Printf("  Release Group: %s\n", info.ReleaseGroup)
	}

	if info.Container != "" {
		fmt.Printf("  Container:     %s\n", info.Container)
	}

	if info.Language != "" {
		fmt.Printf("  Language:      %s\n", info.Language)
	}

	if len(info.Subtitles) > 0 {
		fmt.Printf("  Subtitles:     %v\n", info.Subtitles)
	}

	if info.Edition != "" {
		fmt.Printf("  Edition:       %s\n", info.Edition)
	}

	// Status flags
	var flags []string
	if info.IsComplete {
		flags = append(flags, "COMPLETE")
	}
	if info.IsProper {
		flags = append(flags, "PROPER")
	}
	if info.IsRepack {
		flags = append(flags, "REPACK")
	}
	if info.IsHardcoded {
		flags = append(flags, "HARDCODED")
	}

	if len(flags) > 0 {
		fmt.Printf("  Flags:         %v\n", flags)
	}

	fmt.Printf("  Confidence:    %d%%\n", info.Confidence)
}
