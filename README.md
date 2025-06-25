# Mattermost Log Scrubber

A Golang application that scrubs identifying information from Mattermost log files with configurable levels of data masking.

## Features

- **Three scrubbing levels** with different levels of data masking
- **User mapping** - replaces usernames/emails with consistent `user1@domain1.example.com` format
- **JSON and JSONL format support** for Mattermost log files
- **Consistent replacement mapping** - same inputs always produce same outputs
- **Audit tracking** - CSV file with original values, replacements, and usage counts
- **Dry-run capability** to preview changes before applying
- **Verbose mode** for detailed processing information

## Versioning

This project follows [Semantic Versioning](https://semver.org/). Releases are automated via GitHub Actions and include cross-platform binaries for Linux, macOS, and Windows.

**Current Version**: 0.9.0

## Installation

### Option 1: Download Pre-built Binary (Recommended)

Download the latest release for your platform from the [Releases page](https://github.com/anthropics/mattermost-log-scrubber/releases):

- **Linux**: `mattermost-log-scrubber_Linux_x86_64.tar.gz` or `mattermost-log-scrubber_Linux_arm64.tar.gz`
- **macOS**: `mattermost-log-scrubber_Darwin_x86_64.tar.gz` or `mattermost-log-scrubber_Darwin_arm64.tar.gz`  
- **Windows**: `mattermost-log-scrubber_Windows_x86_64.zip` or `mattermost-log-scrubber_Windows_arm64.zip`

### Option 2: Build from Source

```bash
go build -o mattermost-scrubber
```

## Usage

```bash
./mattermost-scrubber -i <input_file> -l <level> [options]
```

### Required Flags

- `-i, --input`: Input log file path
- `-l, --level`: Scrubbing level (1, 2, or 3)

### Optional Flags

- `-o, --output`: Output file path (default: `<input>_scrubbed.<ext>`)
- `-a, --audit`: Audit file path for tracking mappings (default: `<input>_audit.csv` or `<input>_audit.json`)
- `--audit-type`: Audit file format: csv or json (default: csv)
- `--overwrite`: Action when files exist: prompt, overwrite, timestamp, cancel (default: prompt)
- `--max-file-size`: Maximum input file size: 150MB, 1GB, etc. (default: 150MB)
- `-z, --compress`: Compress output file with gzip
- `--dry-run`: Preview changes without writing output
- `-v, --verbose`: Enable verbose output
- `--version`: Show version and exit

## File Overwrite Protection

The scrubber provides configurable protection against accidentally overwriting existing files:

### Overwrite Modes

- **`prompt`** (default): Ask user for each file conflict - (o)verwrite, (c)ancel, or (r)ename with timestamp
- **`overwrite`**: Automatically overwrite existing files without prompting
- **`timestamp`**: Automatically rename files with timestamp suffix (e.g., `output_20250623_142301.log`)
- **`cancel`**: Cancel operation immediately if any target file already exists

### Usage Examples

```bash
# Interactive prompting (default behavior)
./mattermost-scrubber -i mattermost.log -l 1

# Automatically add timestamps to avoid conflicts
./mattermost-scrubber -i mattermost.log -l 1 --overwrite timestamp

# Automatically overwrite existing files
./mattermost-scrubber -i mattermost.log -l 1 --overwrite overwrite

# Cancel if any files exist (useful for automated scripts)
./mattermost-scrubber -i mattermost.log -l 1 --overwrite cancel
```

## File Size Limits

**Default Limit**: 150MB (covers 99.6% of Mattermost log files based on real-world data)

### Why File Size Limits Exist

Processing very large log files can consume significant memory because the scrubber tracks:
- All unique user mappings discovered in the file
- All replacement operations for the audit trail
- JSON parsing failure samples for error reporting

### Memory Usage Guidelines

| File Size | Expected Memory | Recommendation |
|-----------|----------------|----------------|
| 50MB | ~50MB | ✅ Safe for all systems |
| 150MB | ~150MB | ✅ Default limit, works well |
| 500MB | ~500MB | ⚠️ Requires adequate system memory |
| 1GB+ | ~1GB+ | ⚠️ Monitor system resources |

### Overriding File Size Limits

**CLI Override:**
```bash
# Process a 500MB file
./mattermost-scrubber -i large.log -l 1 --max-file-size 500MB

# Process a 2GB file (ensure adequate system memory!)
./mattermost-scrubber -i huge.log -l 1 --max-file-size 2GB
```

**Config File:**
```json
{
  "ProcessingSettings": {
    "MaxInputFileSize": "500MB"
  }
}
```

### Resource Recommendations

**Before increasing file size limits:**
- Ensure your system has adequate RAM (2-3x the file size)
- Consider processing large files on dedicated systems
- Monitor memory usage during processing
- For files >1GB, consider splitting them first

**Error Handling:**
If you hit the file size limit, you'll see a clear error message:
```
Error: input file 'huge.log' size (200.0 MB) exceeds maximum allowed size (150.0 MB). 
Use --max-file-size or config setting to override
```

## Scrubbing Levels

### Level 1 (Low) - User Mapping Only
- **Emails**: `claude@example.org` → `user1@domain1.example.com` (user mapping with domain tracking)
- **Usernames**: `claude` → `user1` (user mapping)
- **IP Addresses**: `192.168.1.154` → `192.168.1.154` (no masking)
- **UIDs/Channel IDs/Team IDs**: Keep intact

### Level 2 (Medium) - Partial IP Masking
- **Emails**: `claude@example.org` → `user1@domain1.example.com` (user mapping with domain tracking)
- **Usernames**: `claude` → `user1` (user mapping)
- **IP Addresses**: `192.168.1.154` → `***.***.***.154` (keep last octet)
- **UIDs/Channel IDs/Team IDs**: Keep intact

### Level 3 (High) - Full Masking
- **Emails**: `claude@example.org` → `user1@domain1.example.com` (user mapping with domain tracking)
- **Usernames**: `claude` → `user1` (user mapping)
- **IP Addresses**: `192.168.1.154` → `***.***.***.***` (mask entire IP)
- **UIDs/Channel IDs/Team IDs**: `abcdef123456789012345678901234` → `******************12345678` (mask all but last 8, maintain 26 char length)

## User Mapping

The scrubber automatically creates consistent user mappings for usernames and emails:

### How It Works
- **User Detection**: When username + email appear on the same log line, they're linked as the same user
- **Sequential Naming**: First user becomes `user1`/`user1@domain1.example.com`, second becomes `user2`/`user2@domain1.example.com`, etc.
- **Domain Mapping**: Each original domain gets mapped to a numbered subdomain (`domain1.example.com`, `domain2.example.com`, etc.)
- **Consistency**: Same original username/email always maps to the same userN across the entire file
- **Level-based IP/UID Masking**: IP addresses and UIDs are masked according to the selected level (1-3)

### Example
**Input:**
```json
{"user":"claude","email":"claude@mattermost.com","ip":"192.168.1.10"}
{"user":"alice","email":"alice@company.org","ip":"10.0.0.5"}  
{"user":"claude","email":"claude@mattermost.com","ip":"172.16.0.1"}
```

**Output (Level 1):**
```json
{"user":"user1","email":"user1@domain1.example.com","ip":"192.168.1.10"}
{"user":"user2","email":"user2@domain2.example.com","ip":"10.0.0.5"}
{"user":"user1","email":"user1@domain1.example.com","ip":"172.16.0.1"}
```

## Audit Tracking

The scrubber automatically generates an audit file that tracks all replacements made during processing. This file helps both customers and support teams understand what was changed and how often. The audit file can be generated in CSV or JSON format.

### Audit File Formats

#### CSV Format (Default)
The CSV audit file contains five columns:
- **Original Value**: The original text that was replaced
- **New Value**: What it was replaced with
- **Times Replaced**: How many times this replacement occurred
- **Type**: The type of data (email, username, ip, uid)
- **Source**: The source filename where the replacement was found

```csv
Original Value,New Value,Times Replaced,Type,Source
claude@mattermost.com,user1@domain1.example.com,1164,email,mattermost.log
claude,user1,582,username,mattermost.log
192.168.1.10,***.***.***.10,3,ip,mattermost.log
alice@company.org,user2@domain2.example.com,856,email,mattermost.log
alice,user2,291,username,mattermost.log
```

#### JSON Format
The JSON audit file contains an array of audit entries with the same information:

```json
[
  {
    "OriginalValue": "claude@mattermost.com",
    "NewValue": "user1@domain1.example.com",
    "TimesReplaced": 1164,
    "Type": "email",
    "Source": "mattermost.log"
  },
  {
    "OriginalValue": "claude",
    "NewValue": "user1",
    "TimesReplaced": 582,
    "Type": "username",
    "Source": "mattermost.log"
  },
  {
    "OriginalValue": "192.168.1.10",
    "NewValue": "***.***.***.10",
    "TimesReplaced": 3,
    "Type": "ip",
    "Source": "mattermost.log"
  }
]
```

### Using the Audit File

This audit file enables:
- **Customer troubleshooting**: "The issue is with user34, which maps to your original user 'alice'"
- **Support analysis**: "User1 appears 1164 times in logs, indicating high activity"
- **Data verification**: Confirm all sensitive data was properly replaced
- **Reverse lookup**: Map scrubbed identifiers back to original context when needed

## Configuration File

The scrubber supports JSON configuration files for easier management of settings. An example configuration file is provided:

### Example Configuration

```json
{
  "FileSettings": {
    "InputFile": "/opt/mattermost/logs/mattermost.log",
    "OutputFile": "/var/tmp/scrubbed_mattermost.log",
    "AuditFile": "/var/tmp/mattermost_scrubbed_audit.csv",
    "AuditFileType": "csv",
    "CompressOutputFile": false,
    "OverwriteAction": "prompt"
  },
  "ScrubSettings": {
    "ScrubLevel": 1
  },
  "OutputSettings": {
    "Verbose": false
  },
  "ProcessingSettings": {
    "MaxInputFileSize": "150MB"
  }
}
```

**Configuration Options:**

**FileSettings:**
- **InputFile**: Path to the log file to be scrubbed
- **OutputFile**: Path where the scrubbed log will be written
- **AuditFile**: Path where the audit file will be written
- **AuditFileType**: Format for audit output ("csv" or "json")
- **CompressOutputFile**: Compress output file with gzip (true/false)
- **OverwriteAction**: How to handle existing files ("prompt", "overwrite", "timestamp", "cancel")

**ScrubSettings:**
- **ScrubLevel**: Scrubbing intensity level (1, 2, or 3)

**OutputSettings:**
- **Verbose**: Enable verbose output showing user mappings and processing details

**ProcessingSettings:**
- **MaxInputFileSize**: Maximum input file size ("150MB", "1GB", etc.)

### Configuration Usage

1. **Copy the example**: `cp example_scrubber_config.json scrubber_config.json`
2. **Edit the paths** and settings to match your environment
3. **Run with config**: `./mattermost-scrubber --config scrubber_config.json`

### Configuration Precedence

Command line arguments override configuration file values:
- **Highest Priority**: Command line flags (`-i`, `-l`, `-o`, etc.)
- **Medium Priority**: Configuration file values
- **Lowest Priority**: Default values

## Examples

```bash
# Check version
./mattermost-scrubber --version

# Basic usage with level 1 scrubbing
./mattermost-scrubber -i mattermost.log -l 1

# Using configuration file (loads scrubber_config.json by default)
./mattermost-scrubber

# Using custom configuration file
./mattermost-scrubber --config my_config.json

# Config file with command line overrides
./mattermost-scrubber --config scrubber_config.json --level 3 --verbose

# Specify custom output file with level 2 scrubbing
./mattermost-scrubber -i mattermost.log -o clean.log -l 2

# Preview changes with dry-run and verbose output
./mattermost-scrubber -i mattermost.log -l 3 --dry-run -v

# Process with maximum scrubbing level
./mattermost-scrubber -i mattermost.log -l 3 -o fully_scrubbed.log

# Process with verbose output to see user mappings
./mattermost-scrubber -i mattermost.log -l 1 -v

# Specify custom audit file location
./mattermost-scrubber -i mattermost.log -l 2 -a custom_audit.csv

# Generate JSON audit file instead of CSV
./mattermost-scrubber -i mattermost.log -l 2 --audit-type json

# Compress output file with gzip
./mattermost-scrubber -i mattermost.log -l 1 --compress

# Process without creating audit file (dry-run)
./mattermost-scrubber -i mattermost.log -l 3 --dry-run
```

## Sample Input

```json
{"level":"info","msg":"User login successful","time":"2024-01-15T10:30:45.123Z","user":"claude","user_id":"abcdef123456789012345678901234","email":"claude@example.com","ip":"192.168.1.154","team":"engineering","team_id":"zyxwvu987654321098765432109876"}
```

## Sample Output (Level 1)

```json
{"level":"info","msg":"User login successful","time":"2024-01-15T10:30:45.123Z","user":"user1","user_id":"abcdef123456789012345678901234","email":"user1@domain1.example.com","ip":"192.168.1.154","team":"engineering","team_id":"zyxwvu987654321098765432109876"}
```

## Supported Data Types

The scrubber automatically detects and masks:
- Email addresses (RFC 5322 compliant)
- IPv4 addresses
- Usernames in JSON fields (`user`, `username`, `name`)
- Long hexadecimal UIDs (20+ characters) - Level 3 only

## Notes

- The application maintains referential consistency - the same input value will always produce the same masked output
- **User mapping** preserves user relationships while providing better data utility for analysis
- Invalid JSON lines are treated as plain text and processed accordingly
- The application preserves the original log structure and formatting
- All scrubbing is deterministic and repeatable