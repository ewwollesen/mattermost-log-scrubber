package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mattermost-log-scrubber/cli"
	"mattermost-log-scrubber/config"
	"mattermost-log-scrubber/constants"
	"mattermost-log-scrubber/scrubber"
)

func main() {
	if err := runApplication(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runApplication handles the main application logic
func runApplication() error {
	// Parse command line flags
	flags := cli.ParseFlags()

	// Setup configuration
	settings, err := setupApplication(flags)
	if err != nil {
		return err
	}

	// Resolve file paths
	resolveFilePaths(&settings)

	// Show configuration info
	showConfigInfo(settings)

	// Run scrubbing process
	return runScrubbing(settings)
}

// setupApplication handles configuration loading and validation
func setupApplication(flags config.CLIFlags) (config.ResolvedSettings, error) {
	// Get config file path
	configPath, userSpecifiedConfig := cli.GetConfigPath(flags)

	// Load config file if it exists
	var configFile *config.Config
	if _, err := os.Stat(configPath); err == nil {
		configFile, err = config.LoadConfig(configPath)
		if err != nil {
			return config.ResolvedSettings{}, fmt.Errorf("loading config file '%s': %w", configPath, err)
		}
	} else if userSpecifiedConfig {
		return config.ResolvedSettings{}, fmt.Errorf("specified config file '%s' does not exist", configPath)
	}

	// Resolve settings from CLI and config
	settings := config.ResolveSettings(flags, configFile)
	
	// Only show config file message if config values are actually being used
	if configFile != nil && isConfigFileUsed(flags) {
		fmt.Printf("Using config file at %s\n", configPath)
	}

	// Validate settings
	if err := config.ValidateSettings(settings); err != nil {
		return settings, err
	}

	return settings, nil
}

// isConfigFileUsed checks if essential CLI flags are missing and config file would provide them
func isConfigFileUsed(flags config.CLIFlags) bool {
	// Only show message if required flags are missing (input file or scrub level)
	inputProvided := flags.InputFile != "" || flags.Input != ""
	levelProvided := flags.Level != 0 || flags.LevelLong != 0
	
	return !inputProvided || !levelProvided
}

// resolveFilePaths sets default file paths if not specified
func resolveFilePaths(settings *config.ResolvedSettings) {
	// Set default output path if not specified
	if settings.OutputPath == "" {
		ext := filepath.Ext(settings.InputPath)
		base := strings.TrimSuffix(settings.InputPath, ext)
		settings.OutputPath = base + constants.ScrubSuffix + ext
	}
	
	// Add .gz extension if compression is enabled and not already present
	if settings.CompressOutputFile && !strings.HasSuffix(settings.OutputPath, constants.ExtGZ) {
		settings.OutputPath += constants.ExtGZ
	}

	// Set default audit path if not specified
	if settings.AuditPath == "" {
		ext := filepath.Ext(settings.InputPath)
		base := strings.TrimSuffix(settings.InputPath, ext)
		if settings.AuditFileType == constants.AuditTypeJSON {
			settings.AuditPath = base + constants.AuditSuffix + constants.ExtJSON
		} else {
			settings.AuditPath = base + constants.AuditSuffix + constants.ExtCSV
		}
	}
}

// showConfigInfo displays the current configuration
func showConfigInfo(settings config.ResolvedSettings) {
	fmt.Printf("Input file: %s\n", settings.InputPath)
	fmt.Printf("Output file: %s\n", settings.OutputPath)
	fmt.Printf("Audit file: %s\n", settings.AuditPath)
	fmt.Printf("Scrubbing level: %d\n", settings.ScrubLevel)
	fmt.Printf("Compress output: %t\n", settings.CompressOutputFile)
	fmt.Printf("Dry run: %t\n", settings.DryRun)
}

// runScrubbing executes the scrubbing process
func runScrubbing(settings config.ResolvedSettings) error {
	// Initialize scrubber
	s := scrubber.NewScrubber(settings.ScrubLevel, settings.Verbose)

	// Process the file
	actualOutputPath, err := s.ProcessFile(settings.InputPath, settings.OutputPath, settings.DryRun, settings.CompressOutputFile, settings.OverwriteAction)
	if err != nil {
		return fmt.Errorf("processing file: %w", err)
	}

	// Update settings with actual output path used
	settings.OutputPath = actualOutputPath

	// Write output
	return writeOutput(s, settings)
}

// writeOutput handles audit file writing and success messages
func writeOutput(s *scrubber.Scrubber, settings config.ResolvedSettings) error {
	var actualAuditPath string
	
	// Write audit file if not dry run
	if !settings.DryRun {
		var err error
		if settings.AuditFileType == constants.AuditTypeJSON {
			actualAuditPath, err = s.WriteAuditFileJSON(settings.AuditPath, settings.OverwriteAction)
			if err != nil {
				return fmt.Errorf("writing JSON audit file: %w", err)
			}
		} else {
			actualAuditPath, err = s.WriteAuditFile(settings.AuditPath, settings.OverwriteAction)
			if err != nil {
				return fmt.Errorf("writing CSV audit file: %w", err)
			}
		}
	}

	// Show completion message
	if settings.DryRun {
		fmt.Println("Dry run completed successfully. No files were modified.")
	} else {
		fmt.Printf("Log scrubbing completed successfully. Output written to: %s\n", settings.OutputPath)
		fmt.Printf("Audit log written to: %s\n", actualAuditPath)
	}

	return nil
}