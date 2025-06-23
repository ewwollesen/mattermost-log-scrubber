package config

import (
	"encoding/json"
	"fmt"
	"os"

	"mattermost-log-scrubber/constants"
)

// FileSettings contains file-related configuration
type FileSettings struct {
	InputFile     string `json:"InputFile"`
	OutputFile    string `json:"OutputFile"`
	AuditFile     string `json:"AuditFile"`
	AuditFileType string `json:"AuditFileType"`
}

// ScrubSettings contains scrubbing-related configuration
type ScrubSettings struct {
	ScrubLevel int `json:"ScrubLevel"`
}

// OutputSettings contains output-related configuration
type OutputSettings struct {
	Verbose bool `json:"Verbose"`
}

// Config represents the complete configuration structure
type Config struct {
	FileSettings   FileSettings   `json:"FileSettings"`
	ScrubSettings  ScrubSettings  `json:"ScrubSettings"`
	OutputSettings OutputSettings `json:"OutputSettings"`
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// ResolvedSettings contains all resolved configuration values
type ResolvedSettings struct {
	InputPath     string
	OutputPath    string
	AuditPath     string
	AuditFileType string
	ScrubLevel    int
	Verbose       bool
	DryRun        bool
}

// CLIFlags represents command line flag values
type CLIFlags struct {
	InputFile     string
	Input         string
	OutputFile    string
	Output        string
	Level         int
	LevelLong     int
	ConfigFile    string
	ConfigLong    string
	AuditFile     string
	AuditLong     string
	AuditType     string
	Verbose       bool
	VerboseLong   bool
	DryRun        bool
}

// ResolveSettings resolves final configuration values from CLI flags and config file
// CLI flags take precedence over config file values
func ResolveSettings(flags CLIFlags, config *Config) ResolvedSettings {
	settings := ResolvedSettings{}

	// Resolve input path
	settings.InputPath = flags.InputFile
	if settings.InputPath == "" {
		settings.InputPath = flags.Input
	}
	if settings.InputPath == "" && config != nil {
		settings.InputPath = config.FileSettings.InputFile
	}

	// Resolve output path
	settings.OutputPath = flags.OutputFile
	if settings.OutputPath == "" {
		settings.OutputPath = flags.Output
	}
	if settings.OutputPath == "" && config != nil {
		settings.OutputPath = config.FileSettings.OutputFile
	}

	// Resolve scrub level
	settings.ScrubLevel = flags.Level
	if settings.ScrubLevel == 0 {
		settings.ScrubLevel = flags.LevelLong
	}
	if settings.ScrubLevel == 0 && config != nil {
		settings.ScrubLevel = config.ScrubSettings.ScrubLevel
	}

	// Resolve verbose setting
	settings.Verbose = flags.Verbose || flags.VerboseLong
	if !settings.Verbose && config != nil {
		settings.Verbose = config.OutputSettings.Verbose
	}

	// Resolve audit path
	settings.AuditPath = flags.AuditFile
	if settings.AuditPath == "" {
		settings.AuditPath = flags.AuditLong
	}
	if settings.AuditPath == "" && config != nil {
		settings.AuditPath = config.FileSettings.AuditFile
	}

	// Resolve audit file type
	settings.AuditFileType = flags.AuditType
	if settings.AuditFileType == "" && config != nil {
		settings.AuditFileType = config.FileSettings.AuditFileType
	}
	if settings.AuditFileType == "" {
		settings.AuditFileType = constants.AuditTypeCSV
	}

	// Set dry run (CLI only)
	settings.DryRun = flags.DryRun

	return settings
}

// ValidateSettings validates the resolved configuration settings
func ValidateSettings(settings ResolvedSettings) error {
	if settings.InputPath == "" {
		return fmt.Errorf("input file path is required")
	}

	if settings.ScrubLevel < constants.ScrubLevelLow || settings.ScrubLevel > constants.ScrubLevelHigh {
		return fmt.Errorf("scrubbing level must be %d, %d, or %d", 
			constants.ScrubLevelLow, constants.ScrubLevelMedium, constants.ScrubLevelHigh)
	}

	// Check if input file exists
	if _, err := os.Stat(settings.InputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file '%s' does not exist", settings.InputPath)
	}

	return nil
}