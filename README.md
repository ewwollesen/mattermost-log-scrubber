# Mattermost Log Scrubber

A Golang application that scrubs identifying information from Mattermost log files with configurable levels of data masking.

## Features

- **Three scrubbing levels** with different levels of data masking
- **User mapping** - replaces usernames/emails with consistent `user1@domain.com` format
- **JSON and JSONL format support** for Mattermost log files
- **Consistent replacement mapping** - same inputs always produce same outputs
- **Audit tracking** - CSV file with original values, replacements, and usage counts
- **Dry-run capability** to preview changes before applying
- **Verbose mode** for detailed processing information

## Versioning

This project follows [Semantic Versioning](https://semver.org/). Releases are automated via GitHub Actions and include cross-platform binaries for Linux, macOS, and Windows.

**Current Version**: 0.3.1

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
- `-a, --audit`: Audit file path for tracking mappings (default: `<input>_audit.csv`)
- `--dry-run`: Preview changes without writing output
- `-v, --verbose`: Enable verbose output
- `--version`: Show version and exit

## Scrubbing Levels

### Level 1 (Low) - User Mapping Only
- **Emails**: `claude@domain.com` → `user1@domain.com` (user mapping)
- **Usernames**: `claude` → `user1` (user mapping)
- **IP Addresses**: `192.168.1.154` → `192.168.1.154` (no masking)
- **UIDs/Channel IDs/Team IDs**: Keep intact

### Level 2 (Medium) - Partial IP Masking
- **Emails**: `claude@domain.com` → `user1@domain.com` (user mapping)
- **Usernames**: `claude` → `user1` (user mapping)
- **IP Addresses**: `192.168.1.154` → `***.***.***.154` (keep last octet)
- **UIDs/Channel IDs/Team IDs**: Keep intact

### Level 3 (High) - Full Masking
- **Emails**: `claude@domain.com` → `user1@domain.com` (user mapping)
- **Usernames**: `claude` → `user1` (user mapping)
- **IP Addresses**: `192.168.1.154` → `***.***.***.***` (mask entire IP)
- **UIDs/Channel IDs/Team IDs**: `abcdef123456789012345678901234` → `******************12345678` (mask all but last 8, maintain 26 char length)

## User Mapping

The scrubber automatically creates consistent user mappings for usernames and emails:

### How It Works
- **User Detection**: When username + email appear on the same log line, they're linked as the same user
- **Sequential Naming**: First user becomes `user1`/`user1@domain.com`, second becomes `user2`/`user2@domain.com`, etc.
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
{"user":"user1","email":"user1@domain.com","ip":"192.168.1.10"}
{"user":"user2","email":"user2@domain.com","ip":"10.0.0.5"}
{"user":"user1","email":"user1@domain.com","ip":"172.16.0.1"}
```

## Audit Tracking

The scrubber automatically generates a CSV audit file that tracks all replacements made during processing. This file helps both customers and support teams understand what was changed and how often.

### Audit File Format

The audit file contains four columns:
- **Original Value**: The original text that was replaced
- **New Value**: What it was replaced with
- **Times Replaced**: How many times this replacement occurred
- **Type**: The type of data (email, username, ip, uid)

### Example Audit File

```csv
Original Value,New Value,Times Replaced,Type
claude@mattermost.com,user1@domain.com,1164,email
claude,user1,582,username
192.168.1.10,***.***.***.10,3,ip
alice@company.org,user2@domain.com,856,email
alice,user2,291,username
```

### Using the Audit File

This audit file enables:
- **Customer troubleshooting**: "The issue is with user34, which maps to your original user 'alice'"
- **Support analysis**: "User1 appears 1164 times in logs, indicating high activity"
- **Data verification**: Confirm all sensitive data was properly replaced
- **Reverse lookup**: Map scrubbed identifiers back to original context when needed

## Examples

```bash
# Check version
./mattermost-scrubber --version

# Basic usage with level 1 scrubbing
./mattermost-scrubber -i mattermost.log -l 1

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

# Process without creating audit file (dry-run)
./mattermost-scrubber -i mattermost.log -l 3 --dry-run
```

## Sample Input

```json
{"level":"info","msg":"User login successful","time":"2024-01-15T10:30:45.123Z","user":"claude","user_id":"abcdef123456789012345678901234","email":"claude@example.com","ip":"192.168.1.154","team":"engineering","team_id":"zyxwvu987654321098765432109876"}
```

## Sample Output (Level 1)

```json
{"level":"info","msg":"User login successful","time":"2024-01-15T10:30:45.123Z","user":"user1","user_id":"abcdef123456789012345678901234","email":"user1@domain.com","ip":"192.168.1.154","team":"engineering","team_id":"zyxwvu987654321098765432109876"}
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