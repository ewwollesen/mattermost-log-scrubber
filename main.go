package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mattermost-log-scrubber/scrubber"
)

const version = "0.3.1"

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "A Golang application that scrubs identifying information from Mattermost log files.\n\n")
	fmt.Fprintf(os.Stderr, "Required flags:\n")
	fmt.Fprintf(os.Stderr, "  -i, --input string    Input log file path\n")
	fmt.Fprintf(os.Stderr, "  -l, --level int       Scrubbing level (1, 2, or 3)\n\n")
	fmt.Fprintf(os.Stderr, "Optional flags:\n")
	fmt.Fprintf(os.Stderr, "  -o, --output string   Output file path (default: <input>_scrubbed.<ext>)\n")
	fmt.Fprintf(os.Stderr, "  -a, --audit string    Audit file path for tracking mappings (default: <input>_audit.csv)\n")
	fmt.Fprintf(os.Stderr, "  --dry-run             Preview changes without writing output\n")
	fmt.Fprintf(os.Stderr, "  -v, --verbose         Verbose output\n")
	fmt.Fprintf(os.Stderr, "  --version             Show version and exit\n")
	fmt.Fprintf(os.Stderr, "  -h, --help            Show this help message\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  %s -i mattermost.log -l 1\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --input mattermost.log --level 2 --output clean.log\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -i mattermost.log -l 3 --dry-run --verbose\n", os.Args[0])
}

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
	var auditFile = flag.String("a", "", "Audit file path for tracking mappings (optional)")
	var auditFileLong = flag.String("audit", "", "Audit file path for tracking mappings (optional)")
	var showVersion = flag.Bool("version", false, "Show version and exit")
	var showHelp = flag.Bool("h", false, "Show help message")
	var showHelpLong = flag.Bool("help", false, "Show help message")

	// Set custom usage function
	flag.Usage = printUsage

	flag.Parse()

	// Handle help flag
	if *showHelp || *showHelpLong {
		printUsage()
		os.Exit(0)
	}

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

	auditPath := *auditFile
	if auditPath == "" {
		auditPath = *auditFileLong
	}

	// Validate required flags
	if inputPath == "" {
		fmt.Fprintf(os.Stderr, "Error: Input file path is required\n\n")
		printUsage()
		os.Exit(1)
	}

	if scrubbingLevel < 1 || scrubbingLevel > 3 {
		fmt.Fprintf(os.Stderr, "Error: Scrubbing level must be 1, 2, or 3\n\n")
		printUsage()
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

	// Set default audit path if not specified
	if auditPath == "" {
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		auditPath = base + "_audit.csv"
	}

	// Always show basic info
	fmt.Printf("Input file: %s\n", inputPath)
	fmt.Printf("Output file: %s\n", outputPath)
	fmt.Printf("Audit file: %s\n", auditPath)
	fmt.Printf("Scrubbing level: %d\n", scrubbingLevel)
	fmt.Printf("Dry run: %t\n", *dryRun)

	// Initialize scrubber
	s := scrubber.NewScrubber(scrubbingLevel, isVerbose)

	// Process the file
	err := s.ProcessFile(inputPath, outputPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing file: %v\n", err)
		os.Exit(1)
	}

	// Write audit file if not dry run
	if !*dryRun {
		err = s.WriteAuditFile(auditPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing audit file: %v\n", err)
			os.Exit(1)
		}
	}

	if *dryRun {
		fmt.Println("Dry run completed successfully. No files were modified.")
	} else {
		fmt.Printf("Log scrubbing completed successfully. Output written to: %s\n", outputPath)
		fmt.Printf("Audit log written to: %s\n", auditPath)
	}
}