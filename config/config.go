package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"mattermost-log-scrubber/constants"
)

// FileSettings contains file-related configuration
type FileSettings struct {
	InputFile          string `json:"InputFile"`
	OutputFile         string `json:"OutputFile"`
	AuditFile          string `json:"AuditFile"`
	AuditFileType      string `json:"AuditFileType"`
	CompressOutputFile bool   `json:"CompressOutputFile"`
	OverwriteAction    string `json:"OverwriteAction"`
}

// ScrubSettings contains scrubbing-related configuration
type ScrubSettings struct {
	ScrubLevel int `json:"ScrubLevel"`
}

// OutputSettings contains output-related configuration
type OutputSettings struct {
	Verbose bool `json:"Verbose"`
}

// ProcessingSettings contains processing-related configuration
type ProcessingSettings struct {
	MaxInputFileSize string `json:"MaxInputFileSize"`
}

// Config represents the complete configuration structure
type Config struct {
	FileSettings        FileSettings        `json:"FileSettings"`
	ScrubSettings       ScrubSettings       `json:"ScrubSettings"`
	OutputSettings      OutputSettings      `json:"OutputSettings"`
	ProcessingSettings  ProcessingSettings  `json:"ProcessingSettings"`
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

// parseFileSize parses human-readable file sizes (e.g., "150MB", "1GB", "500KB")
func parseFileSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return constants.DefaultMaxFileSize, nil
	}
	
	// Regex to match number and optional unit
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(B|KB|MB|GB|TB)?$`)
	matches := re.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(sizeStr)))
	
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid file size format: %s (expected format like '150MB', '1GB', etc.)", sizeStr)
	}
	
	// Parse the numeric part
	size, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value in file size: %s", matches[1])
	}
	
	// Convert based on unit (default to bytes if no unit)
	unit := matches[2]
	if unit == "" {
		unit = "B"
	}
	
	var multiplier int64
	switch unit {
	case "B":
		multiplier = 1
	case "KB":
		multiplier = 1024
	case "MB":
		multiplier = 1024 * 1024
	case "GB":
		multiplier = 1024 * 1024 * 1024
	case "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unsupported file size unit: %s (supported: B, KB, MB, GB, TB)", unit)
	}
	
	return int64(size * float64(multiplier)), nil
}

// formatFileSize formats a file size in bytes to human-readable format
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ResolvedSettings contains all resolved configuration values
type ResolvedSettings struct {
	InputPath          string
	OutputPath         string
	AuditPath          string
	AuditFileType      string
	ScrubLevel         int
	Verbose            bool
	DryRun             bool
	CompressOutputFile bool
	OverwriteAction    string
	MaxInputFileSize   int64
}

// CLIFlags represents command line flag values
type CLIFlags struct {
	InputFile       string
	Input           string
	OutputFile      string
	Output          string
	Level           int
	LevelLong       int
	ConfigFile      string
	ConfigLong      string
	AuditFile       string
	AuditLong       string
	AuditType       string
	OverwriteAction string
	MaxFileSize     string
	Verbose         bool
	VerboseLong     bool
	DryRun          bool
	Compress        bool
	CompressLong    bool
}

// ResolveSettings resolves final configuration values from CLI flags and config file
// CLI flags take precedence over config file values when both are provided
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

	// Resolve compression setting
	settings.CompressOutputFile = flags.Compress || flags.CompressLong
	if !settings.CompressOutputFile && config != nil {
		settings.CompressOutputFile = config.FileSettings.CompressOutputFile
	}

	// Resolve overwrite action
	settings.OverwriteAction = flags.OverwriteAction
	if settings.OverwriteAction == "" && config != nil {
		settings.OverwriteAction = config.FileSettings.OverwriteAction
	}
	if settings.OverwriteAction == "" {
		settings.OverwriteAction = constants.OverwritePrompt
	}

	// Resolve max input file size - CLI flags take precedence over config file
	maxFileSizeStr := flags.MaxFileSize
	if maxFileSizeStr == "" && config != nil {
		maxFileSizeStr = config.ProcessingSettings.MaxInputFileSize
	}
	
	var err error
	settings.MaxInputFileSize, err = parseFileSize(maxFileSizeStr)
	if err != nil {
		// If there's an error parsing, use the default
		settings.MaxInputFileSize = constants.DefaultMaxFileSize
	}

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

	// Validate overwrite action
	validActions := []string{
		constants.OverwritePrompt,
		constants.OverwriteOverwrite,
		constants.OverwriteTimestamp,
		constants.OverwriteCancel,
	}
	validAction := false
	for _, action := range validActions {
		if settings.OverwriteAction == action {
			validAction = true
			break
		}
	}
	if !validAction {
		return fmt.Errorf("overwrite action must be one of: %s, %s, %s, %s",
			constants.OverwritePrompt, constants.OverwriteOverwrite, constants.OverwriteTimestamp, constants.OverwriteCancel)
	}

	// Check if input file exists and get its size
	fileInfo, err := os.Stat(settings.InputPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("input file '%s' does not exist", settings.InputPath)
	}
	if err != nil {
		return fmt.Errorf("failed to get file info for '%s': %w", settings.InputPath, err)
	}

	// Check file size against limit
	fileSize := fileInfo.Size()
	if fileSize > settings.MaxInputFileSize {
		return fmt.Errorf("input file '%s' size (%s) exceeds maximum allowed size (%s). Use --max-file-size or config setting to override",
			settings.InputPath,
			formatFileSize(fileSize),
			formatFileSize(settings.MaxInputFileSize))
	}

	return nil
}