# Mattermost Log Scrubber

A Golang application that scrubs identifying information from Mattermost log files with configurable levels of data masking.

## Features

- **Three scrubbing levels** with different levels of data masking
- **User mapping mode** - replaces usernames/emails with consistent `user1@domain.com` format
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
- `--use-mapping`: Use user mapping instead of masking (user1@domain.com vs ***@domain.com)
- `--dry-run`: Preview changes without writing output
- `-v, --verbose`: Enable verbose output

## Scrubbing Levels

### Level 1 (Low) - Partial Masking
- **Emails**: `claude@domain.com` → `***ude@domain.com` (keep last 3 chars of local part)
- **Usernames**: `claude` → `***ude` (keep last 3 chars)
- **IP Addresses**: `192.168.1.154` → `***.***.***.154` (keep last octet)
- **UIDs/Channel IDs/Team IDs**: Keep intact

### Level 2 (Medium) - Local Part Masking
- **Emails**: `claude@domain.com` → `*****@domain.com` (mask entire local part)
- **Usernames**: `claude` → `*****` (mask entire username)
- **IP Addresses**: `192.168.1.154` → `***.***.***.***` (mask entire IP)
- **UIDs/Channel IDs/Team IDs**: Keep intact

### Level 3 (High) - Full Masking
- **Emails**: `claude@domain.com` → `******@**********` (mask everything)
- **Usernames**: `claude` → `*****` (mask entirely)
- **IP Addresses**: `192.168.1.154` → `***.***.***.***` (mask entirely)
- **UIDs/Channel IDs/Team IDs**: `abcdef123456789012345678901234` → `**********************1234` (mask all but last 4, maintain 26 char length)

## User Mapping Mode

When using `--use-mapping`, the scrubber creates consistent user mappings instead of masking:

### How It Works
- **User Detection**: When username + email appear on the same log line, they're linked as the same user
- **Sequential Naming**: First user becomes `user1`/`user1@domain.com`, second becomes `user2`/`user2@domain.com`, etc.
- **Consistency**: Same original username/email always maps to the same userN across the entire file
- **IP/UID Masking**: Still follows the selected level (1-3) for IP addresses and UIDs

### Example
**Input:**
```json
{"user":"claude","email":"claude@mattermost.com","ip":"192.168.1.10"}
{"user":"alice","email":"alice@company.org","ip":"10.0.0.5"}  
{"user":"claude","email":"claude@mattermost.com","ip":"172.16.0.1"}
```

**Output (Level 1 + Mapping):**
```json
{"user":"user1","email":"user1@domain.com","ip":"***.***.***.10"}
{"user":"user2","email":"user2@domain.com","ip":"***.***.***.5"}
{"user":"user1","email":"user1@domain.com","ip":"***.***.***.1"}
```

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

# Use user mapping instead of masking
./mattermost-scrubber -i mattermost.log -l 2 --use-mapping -o mapped.log

# User mapping with verbose output
./mattermost-scrubber -i mattermost.log -l 1 --use-mapping -v
```

## Sample Input

```json
{"level":"info","msg":"User login successful","time":"2024-01-15T10:30:45.123Z","user":"claude","user_id":"abcdef123456789012345678901234","email":"claude@example.com","ip":"192.168.1.154","team":"engineering","team_id":"zyxwvu987654321098765432109876"}
```

## Sample Output (Level 1 Masking)

```json
{"channel":"general","email":"***ude@example.com","ip":"***.***.***.154","level":"info","msg":"User login successful","team":"engineering","team_id":"zyxwvu987654321098765432109876","time":"2024-01-15T10:30:45.123Z","user":"***ude","user_id":"abcdef123456789012345678901234"}
```

## Sample Output (Level 1 + User Mapping)

```json
{"channel":"general","email":"user1@domain.com","ip":"***.***.***.154","level":"info","msg":"User login successful","team":"engineering","team_id":"zyxwvu987654321098765432109876","time":"2024-01-15T10:30:45.123Z","user":"user1","user_id":"abcdef123456789012345678901234"}
```

## Supported Data Types

The scrubber automatically detects and masks:
- Email addresses (RFC 5322 compliant)
- IPv4 addresses
- Usernames in JSON fields (`user`, `username`, `name`)
- Long hexadecimal UIDs (20+ characters) - Level 3 only

## Notes

- The application maintains referential consistency - the same input value will always produce the same masked output
- **User mapping mode** preserves user relationships while providing better data utility for analysis
- **Masking mode** provides traditional asterisk-based obfuscation
- Invalid JSON lines are treated as plain text and processed accordingly
- The application preserves the original log structure and formatting
- All scrubbing is deterministic and repeatable