# Mattermost Log Scrubber

**Safely removes sensitive information from Mattermost log files while preserving their analytical value.**

Used for sharing logs with the Mattermost support teams.

## Requirements

Requires Mattermost version 10+

## Quick Start

1. **[Download the latest release](https://github.com/anthropics/mattermost-log-scrubber/releases)** for your platform
2. **Run the scrubber**: `./mattermost-scrubber -i mattermost.log -l 1`
3. **Share the cleaned file**: `mattermost_scrubbed.log` is now safe to share

That's it! Your log file is scrubbed and ready to go.

## What It Does

**Before (sensitive data exposed):**

```json
{
  "user": "alice.smith",
  "email": "alice.smith@acme.com",
  "ip": "192.168.1.100",
  "url": "https://chat.acme.com/api"
}
```

**After (scrubbed but still useful):**

```json
{
  "user": "user1",
  "email": "user1@domain1",
  "ip": "192.168.1.100",
  "url": "https://domain1/api"
}
```

### Key Features

- 🛡️ **Removes sensitive data**: Emails, usernames, URLs, IP addresses (optional), and internal IDs
- 🔄 **Maintains consistency**: Same original value always maps to the same replacement
- 📊 **Preserves structure**: JSON format and log structure remain intact for analysis
- 📋 **Creates audit trail**: Track exactly what was changed for reverse lookup
- ⚡ **Three security levels**: Choose how much to mask based on your needs
- 🔒 **Safe by default**: Won't overwrite existing files without permission

## Installation

### Download Pre-built Binary (Recommended)

**[→ Download Latest Release](https://github.com/anthropics/mattermost-log-scrubber/releases)**

Choose your platform:

- **Linux**: `mattermost-log-scrubber_Linux_x86_64.tar.gz`
- **macOS**: `mattermost-log-scrubber_Darwin_x86_64.tar.gz`
- **Windows**: `mattermost-log-scrubber_Windows_x86_64.zip`

### Build from Source

```bash
git clone https://github.com/anthropics/mattermost-log-scrubber
cd mattermost-log-scrubber
go build -o mattermost-scrubber
```

## Usage

### Basic Usage

```bash
./mattermost-scrubber -i <log-file> -l <security-level>
```

**Examples:**

```bash
# Light scrubbing (usernames/emails only)
./mattermost-scrubber -i mattermost.log -l 1

# Medium scrubbing (+ partial IP masking)
./mattermost-scrubber -i mattermost.log -l 2

# Full scrubbing (+ full IP and ID masking)
./mattermost-scrubber -i mattermost.log -l 3
```

### Common Options

| Flag            | Description                       | Example             |
| --------------- | --------------------------------- | ------------------- |
| `-i, --input`   | Input log file **(required)**     | `-i mattermost.log` |
| `-l, --level`   | Security level 1-3 **(required)** | `-l 2`              |
| `-o, --output`  | Custom output file                | `-o clean.log`      |
| `--dry-run`     | Preview changes only              | `--dry-run`         |
| `-v, --verbose` | Show detailed info                | `-v`                |

**[→ See all options](#all-command-options)**

## Security Levels

Choose the right level of protection for your use case:

### Level 1 - Basic (Share with internal teams)

**What's masked:** Usernames, emails, URLs  
**What's kept:** IP addresses, internal IDs, timestamps, error messages

```
alice@company.com → user1@domain1
https://chat.company.com → https://domain1
IP: 192.168.1.100 → 192.168.1.100 (unchanged)
```

### Level 2 - Moderate (Share with vendors)

**What's masked:** Everything from Level 1 + partial IP addresses  
**What's kept:** Last IP octet, internal IDs, timestamps, error messages

```
alice@company.com → user1@domain1
https://chat.company.com → https://subdomain1.domain1
IP: 192.168.1.100 → ***.***.***.100
```

### Level 3 - Maximum (Public sharing/compliance)

**What's masked:** Everything from Level 2 + full IPs and internal IDs  
**What's kept:** Timestamps, error messages, log structure

```
alice@company.com → user1@domain1
https://chat.company.com → https://subdomain1.domain1
IP: 192.168.1.100 → ***.***.***.**
ID: abc123...xyz789 → ******************xyz789
```

## Understanding the Output

After scrubbing, you'll get two files:

### 1. Scrubbed Log File

- **Default name**: `<original>_scrubbed.log`
- **Safe to share** - all sensitive data removed
- **Same format** as original for easy analysis
- **Consistent mapping** - same inputs always produce same outputs

### 2. Audit File

- **Default name**: `<original>_audit.csv`
- **Maps scrubbed values back to originals** (keep this private!)
- **Shows replacement statistics**
- **Enables reverse lookup** for troubleshooting

**Audit file example:**

```csv
Original Value,New Value,Times Replaced,Type,Source
alice@company.com,user1@domain1,245,email,mattermost.log
alice,user1,128,username,mattermost.log
https://chat.company.com,https://domain1,12,fqdn,mattermost.log
```

## Important Notes

- ⚠️ **Keep audit files private** - they contain the original sensitive data
- ✅ **Share scrubbed files freely** - they're safe for external use
- 🔄 **Consistent results** - running the tool multiple times on the same file produces identical output
- 📁 **File protection** - Tool won't overwrite existing files without confirmation

## Advanced Usage

<details>
<summary><strong>Configuration Files</strong></summary>

Create a `scrubber_config.json` file for repeated use:

```json
{
  "FileSettings": {
    "InputFile": "/path/to/mattermost.log",
    "ScrubLevel": 2,
    "OverwriteAction": "timestamp"
  }
}
```

Run with: `./mattermost-scrubber --config scrubber_config.json`

</details>

<details>
<summary><strong>File Size Limits</strong></summary>

**Default limit:** 150MB (covers 99.6% of Mattermost logs)

For larger files:

```bash
./mattermost-scrubber -i large.log -l 1 --max-file-size 500MB
```

**Memory usage:** Plan for ~1GB RAM per 1GB log file

</details>

<details>
<summary><strong>Batch Processing</strong></summary>

```bash
# Process multiple files
for file in *.log; do
  ./mattermost-scrubber -i "$file" -l 2 --overwrite timestamp
done
```

</details>

## All Command Options

### Required

- `-i, --input` - Input log file path
- `-l, --level` - Scrubbing level (1, 2, or 3)

### Output Control

- `-o, --output` - Output file path (default: `<input>_scrubbed.<ext>`)
- `-a, --audit` - Audit file path (default: `<input>_audit.csv`)
- `--audit-type` - Audit format: `csv` or `json` (default: csv)
- `-z, --compress` - Compress output with gzip

### File Handling

- `--overwrite` - When files exist: `prompt`|`overwrite`|`timestamp`|`cancel` (default: prompt)
- `--max-file-size` - Max input size: `150MB`, `1GB`, etc. (default: 150MB)

### Processing

- `--dry-run` - Preview changes without writing files
- `-v, --verbose` - Show detailed processing information
- `--config` - Use configuration file
- `--version` - Show version and exit

## What Data Gets Scrubbed

| Data Type          | Level 1   | Level 2    | Level 3   | Example                                        |
| ------------------ | --------- | ---------- | --------- | ---------------------------------------------- |
| **Emails**         | ✅ Masked | ✅ Masked  | ✅ Masked | `alice@company.com` → `user1@domain1`          |
| **Usernames**      | ✅ Masked | ✅ Masked  | ✅ Masked | `alice.smith` → `user1`                        |
| **URLs**           | ✅ Masked | ✅ Masked  | ✅ Masked | `https://chat.company.com` → `https://domain1` |
| **IP Addresses**   | ❌ Kept   | ⚠️ Partial | ✅ Masked | `192.168.1.100` → `***.***.***.***`            |
| **Internal IDs**   | ❌ Kept   | ❌ Kept    | ✅ Masked | `abc123...xyz` → `******...xyz`                |
| **Timestamps**     | ❌ Kept   | ❌ Kept    | ❌ Kept   | Always preserved                               |
| **Error Messages** | ❌ Kept   | ❌ Kept    | ❌ Kept   | Always preserved                               |

## Support & Contributing

- **Issues & Questions**: [GitHub Issues](https://github.com/anthropics/mattermost-log-scrubber/issues)
- **Current Version**: v0.10.0
- **License**: MIT License

---

**Need help?** Check the [troubleshooting section](https://github.com/anthropics/mattermost-log-scrubber/issues) or create an issue.
