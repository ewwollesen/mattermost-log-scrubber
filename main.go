package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mattermost-log-scrubber/scrubber"
)

const version = "0.1.0"

func main() {
	var inputFile = flag.String("i", "", "Input log file path (required)")
	var input = flag.String("input", "", "Input log file path (required)")
	var outputFile = flag.String("o", "", "Output file path (optional)")
	var output = flag.String("output", "", "Output file path (optional)")
	var level = flag.Int("l", 0, "Scrubbing level 1-3 (required)")
	var levelLong = flag.Int("level", 0, "Scrubbing level 1-3 (required)")
	var dryRun = flag.Bool("dry-run", false, "Preview changes without writing output")
	var verbose = flag.Bool("v", false, "Verbose output")
	var verboseLong = flag.Bool("verbose", false, "Verbose output")
	var showVersion = flag.Bool("version", false, "Show version and exit")

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("mattermost-log-scrubber v%s\n", version)
		os.Exit(0)
	}

	// Handle short and long flag variants
	inputPath := *inputFile
	if inputPath == "" {
		inputPath = *input
	}

	outputPath := *outputFile
	if outputPath == "" {
		outputPath = *output
	}

	scrubbingLevel := *level
	if scrubbingLevel == 0 {
		scrubbingLevel = *levelLong
	}

	isVerbose := *verbose || *verboseLong

	// Validate required flags
	if inputPath == "" {
		fmt.Fprintf(os.Stderr, "Error: Input file path is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if scrubbingLevel < 1 || scrubbingLevel > 3 {
		fmt.Fprintf(os.Stderr, "Error: Scrubbing level must be 1, 2, or 3\n")
		flag.Usage()
		os.Exit(1)
	}

	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file '%s' does not exist\n", inputPath)
		os.Exit(1)
	}

	// Set default output path if not specified
	if outputPath == "" {
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		outputPath = base + "_scrubbed" + ext
	}

	if isVerbose {
		fmt.Printf("Input file: %s\n", inputPath)
		fmt.Printf("Output file: %s\n", outputPath)
		fmt.Printf("Scrubbing level: %d\n", scrubbingLevel)
		fmt.Printf("Dry run: %t\n", *dryRun)
	}

	// Initialize scrubber
	s := scrubber.NewScrubber(scrubbingLevel, isVerbose)

	// Process the file
	err := s.ProcessFile(inputPath, outputPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing file: %v\n", err)
		os.Exit(1)
	}

	if *dryRun {
		fmt.Println("Dry run completed successfully. No files were modified.")
	} else {
		fmt.Printf("Log scrubbing completed successfully. Output written to: %s\n", outputPath)
	}
}