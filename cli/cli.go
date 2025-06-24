package cli

import (
	"flag"
	"fmt"
	"os"

	"mattermost-log-scrubber/config"
	"mattermost-log-scrubber/constants"
)

// ParseFlags parses command line flags and returns flag values
func ParseFlags() config.CLIFlags {
	var flags config.CLIFlags

	// Define flags
	flag.StringVar(&flags.InputFile, "i", "", "Input log file path (required)")
	flag.StringVar(&flags.Input, "input", "", "Input log file path (required)")
	flag.StringVar(&flags.OutputFile, "o", "", "Output file path (optional)")
	flag.StringVar(&flags.Output, "output", "", "Output file path (optional)")
	flag.IntVar(&flags.Level, "l", 0, "Scrubbing level 1-3 (required)")
	flag.IntVar(&flags.LevelLong, "level", 0, "Scrubbing level 1-3 (required)")
	flag.StringVar(&flags.ConfigFile, "c", "", "Config file path (default: scrubber_config.json)")
	flag.StringVar(&flags.ConfigLong, "config", "", "Config file path (default: scrubber_config.json)")
	flag.BoolVar(&flags.DryRun, "dry-run", false, "Preview changes without writing output")
	flag.BoolVar(&flags.Verbose, "v", false, "Verbose output")
	flag.BoolVar(&flags.VerboseLong, "verbose", false, "Verbose output")
	flag.StringVar(&flags.AuditFile, "a", "", "Audit file path for tracking mappings (optional)")
	flag.StringVar(&flags.AuditLong, "audit", "", "Audit file path for tracking mappings (optional)")
	flag.StringVar(&flags.AuditType, "audit-type", "", "Audit file format: csv or json (default: csv)")
	flag.StringVar(&flags.OverwriteAction, "overwrite", "", "Action when files exist: prompt, overwrite, timestamp, cancel (default: prompt)")
	flag.StringVar(&flags.MaxFileSize, "max-file-size", "", "Maximum input file size: 150MB, 1GB, etc. (default: 150MB)")
	flag.BoolVar(&flags.Compress, "z", false, "Compress output file with gzip")
	flag.BoolVar(&flags.CompressLong, "compress", false, "Compress output file with gzip")

	// Version and help flags
	var showVersion bool
	var showVersionLong bool
	var showHelp bool
	var showHelpLong bool

	flag.BoolVar(&showVersion, "V", false, "Show version and exit")
	flag.BoolVar(&showVersionLong, "version", false, "Show version and exit")
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.BoolVar(&showHelpLong, "help", false, "Show help message")

	// Set custom usage function
	flag.Usage = PrintUsage

	flag.Parse()

	// Handle help flag
	if showHelp || showHelpLong {
		PrintUsage()
		os.Exit(0)
	}

	// Handle version flag
	if showVersion || showVersionLong {
		fmt.Printf("%s v%s\n", constants.AppName, constants.Version)
		os.Exit(0)
	}

	return flags
}

// PrintUsage prints the application usage information
func PrintUsage() {
	fmt.Fprintf(os.Stderr, "%s\n\n", constants.Description)
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Required flags (unless using config file):\n")
	fmt.Fprintf(os.Stderr, "  -i, --input string    Input log file path\n")
	fmt.Fprintf(os.Stderr, "  -l, --level int       Scrubbing level (1, 2, or 3)\n\n")
	fmt.Fprintf(os.Stderr, "Optional flags:\n")
	fmt.Fprintf(os.Stderr, "  -c, --config string   Config file path (default: %s)\n", constants.DefaultConfigFile)
	fmt.Fprintf(os.Stderr, "  -o, --output string   Output file path (default: <input>%s.<ext>)\n", constants.ScrubSuffix)
	fmt.Fprintf(os.Stderr, "  -a, --audit string    Audit file path for tracking mappings (default: <input>%s.csv)\n", constants.AuditSuffix)
	fmt.Fprintf(os.Stderr, "  --audit-type string   Audit file format: %s or %s (default: %s)\n", constants.AuditTypeCSV, constants.AuditTypeJSON, constants.AuditTypeCSV)
	fmt.Fprintf(os.Stderr, "  --overwrite string    Action when files exist: %s, %s, %s, %s (default: %s)\n", constants.OverwritePrompt, constants.OverwriteOverwrite, constants.OverwriteTimestamp, constants.OverwriteCancel, constants.OverwritePrompt)
	fmt.Fprintf(os.Stderr, "  --max-file-size string Maximum input file size: 150MB, 1GB, etc. (default: 150MB)\n")
	fmt.Fprintf(os.Stderr, "  -z, --compress        Compress output file with gzip\n")
	fmt.Fprintf(os.Stderr, "  --dry-run             Preview changes without writing output\n")
	fmt.Fprintf(os.Stderr, "  -v, --verbose         Verbose output\n")
	fmt.Fprintf(os.Stderr, "  -V, --version         Show version and exit\n")
	fmt.Fprintf(os.Stderr, "  -h, --help            Show this help message\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  %s -i mattermost.log -l 1\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --input mattermost.log --level 2 --output clean.log\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -i mattermost.log -l 3 --dry-run --verbose\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --config %s\n", os.Args[0], constants.DefaultConfigFile)
	fmt.Fprintf(os.Stderr, "  %s -c my_config.json --verbose\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -i mattermost.log -l 2 --audit-type %s\n", os.Args[0], constants.AuditTypeJSON)
	fmt.Fprintf(os.Stderr, "  %s -i mattermost.log -l 1 --compress\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -i mattermost.log -l 1 --overwrite %s\n", os.Args[0], constants.OverwriteTimestamp)
	fmt.Fprintf(os.Stderr, "  %s -i large.log -l 1 --max-file-size 500MB\n", os.Args[0])
}

// GetConfigPath determines the configuration file path from CLI flags
func GetConfigPath(flags config.CLIFlags) (string, bool) {
	configPath := flags.ConfigFile
	if configPath == "" {
		configPath = flags.ConfigLong
	}

	// Check if user explicitly specified a config file
	userSpecifiedConfig := flags.ConfigFile != "" || flags.ConfigLong != ""

	// Set default config path if not specified
	if configPath == "" {
		configPath = constants.DefaultConfigFile
	}

	return configPath, userSpecifiedConfig
}