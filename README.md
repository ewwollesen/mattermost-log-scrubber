# Mattermost Log Scrubber

A Golang application that scrubs identifying information from Mattermost log files with configurable levels of data masking.

## Features

- **Three scrubbing levels** with different levels of data masking
- **JSON and JSONL format support** for Mattermost log files
- **Consistent replacement mapping** - same inputs always produce same outputs
- **Dry-run capability** to preview changes before applying
- **Verbose mode** for detailed processing information

## Installation

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
- `--dry-run`: Preview changes without writing output
- `-v, --verbose`: Enable verbose output

## Scrubbing Levels

### Level 1 (Low) - Partial Masking
- **Emails**: `claude@domain.com` → `xxxude@domain.com` (keep last 3 chars of local part)
- **Usernames**: `claude` → `xxxude` (keep last 3 chars)
- **IP Addresses**: `192.168.1.154` → `xxx.xxx.xxx.154` (keep last octet)
- **UIDs/Channel IDs/Team IDs**: Keep intact

### Level 2 (Medium) - Local Part Masking
- **Emails**: `claude@domain.com` → `xxxxx@domain.com` (mask entire local part)
- **Usernames**: `claude` → `xxxxx` (mask entire username)
- **IP Addresses**: `192.168.1.154` → `xxx.xxx.xxx.xxx` (mask entire IP)
- **UIDs/Channel IDs/Team IDs**: Keep intact

### Level 3 (High) - Full Masking
- **Emails**: `claude@domain.com` → `xxxxxx@xxxxxxxxxx` (mask everything)
- **Usernames**: `claude` → `xxxxx` (mask entirely)
- **IP Addresses**: `192.168.1.154` → `xxx.xxx.xxx.xxx` (mask entirely)
- **UIDs/Channel IDs/Team IDs**: `abcdef123456789012345678901234` → `xxxxxxxxxxxxxxxxxxxxxx1234` (mask all but last 4, maintain 26 char length)

## Examples

```bash
# Basic usage with level 1 scrubbing
./mattermost-scrubber -i mattermost.log -l 1

# Specify custom output file with level 2 scrubbing
./mattermost-scrubber -i mattermost.log -o clean.log -l 2

# Preview changes with dry-run and verbose output
./mattermost-scrubber -i mattermost.log -l 3 --dry-run -v

# Process with maximum scrubbing level
./mattermost-scrubber -i mattermost.log -l 3 -o fully_scrubbed.log
```

## Sample Input

```json
{"level":"info","msg":"User login successful","time":"2024-01-15T10:30:45.123Z","user":"claude","user_id":"abcdef123456789012345678901234","email":"claude@example.com","ip":"192.168.1.154","team":"engineering","team_id":"zyxwvu987654321098765432109876"}
```

## Sample Output (Level 1)

```json
{"channel":"general","email":"xxxude@example.com","ip":"xxx.xxx.xxx.154","level":"info","msg":"User login successful","team":"engineering","team_id":"zyxwvu987654321098765432109876","time":"2024-01-15T10:30:45.123Z","user":"xxxude","user_id":"abcdef123456789012345678901234"}
```

## Supported Data Types

The scrubber automatically detects and masks:
- Email addresses (RFC 5322 compliant)
- IPv4 addresses
- Usernames in JSON fields (`user`, `username`, `name`)
- Long hexadecimal UIDs (20+ characters) - Level 3 only

## Notes

- The application maintains referential consistency - the same input value will always produce the same masked output
- Invalid JSON lines are treated as plain text and processed accordingly
- The application preserves the original log structure and formatting
- All scrubbing is deterministic and repeatable