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
	
	if len(info.Episodes) > 0 {
		fmt.Printf("  Episodes:      %v\n", info.Episodes)
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
	
	fmt.Printf("  Confidence:    %.1f%%\n", info.Confidence*100)
}
