package constants

// Application constants
const (
	Version     = "0.10.0"
	AppName     = "mattermost-log-scrubber"
	Description = "A Golang application that scrubs identifying information from Mattermost log files."
)

// File-related constants
const (
	DefaultConfigFile = "scrubber_config.json"
	ScrubSuffix       = "_scrubbed"
	AuditSuffix       = "_audit"
)

// Audit file types
const (
	AuditTypeCSV  = "csv"
	AuditTypeJSON = "json"
)

// File extensions
const (
	ExtCSV  = ".csv"
	ExtJSON = ".json"
	ExtGZ   = ".gz"
)

// Scrubbing levels
const (
	ScrubLevelLow    = 1
	ScrubLevelMedium = 2
	ScrubLevelHigh   = 3
)

// Domain constants - removed DefaultDomain for simplified domain1, domain2 format

// Processing constants
const (
	ProgressInterval = 1000 // Show progress every N lines
	MinUIDLength     = 20   // Minimum UID length for scrubbing
	UIDTargetLength  = 26   // Target UID length after scrubbing
	UIDKeepChars     = 8    // Characters to keep at end of UID
)

// Scrubbing type constants
const (
	TypeEmail    = "email"
	TypeUsername = "username"
	TypeIP       = "ip"
	TypeUID      = "uid"
	TypeFQDN     = "fqdn"
)

// Overwrite action constants
const (
	OverwritePrompt    = "prompt"    // Prompt user for each conflict
	OverwriteOverwrite = "overwrite" // Automatically overwrite existing files
	OverwriteTimestamp = "timestamp" // Automatically add timestamp suffix
	OverwriteCancel    = "cancel"    // Cancel operation on any conflict
)

// File size constants
const (
	DefaultMaxFileSize = 150 * 1024 * 1024 // 150MB default limit
)