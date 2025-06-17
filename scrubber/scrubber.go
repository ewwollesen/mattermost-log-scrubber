package scrubber

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Scrubber struct {
	level    int
	verbose  bool
	emailMap map[string]string
	userMap  map[string]string
	ipMap    map[string]string
	uidMap   map[string]string
}

func NewScrubber(level int, verbose bool) *Scrubber {
	return &Scrubber{
		level:    level,
		verbose:  verbose,
		emailMap: make(map[string]string),
		userMap:  make(map[string]string),
		ipMap:    make(map[string]string),
		uidMap:   make(map[string]string),
	}
}

// ProcessFile processes the input file and writes scrubbed output
func (s *Scrubber) ProcessFile(inputPath, outputPath string, dryRun bool) error {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	var outputFile *os.File
	if !dryRun {
		outputFile, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outputFile.Close()
	}

	scanner := bufio.NewScanner(inputFile)
	lineCount := 0
	processedCount := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		
		if strings.TrimSpace(line) == "" {
			continue
		}

		scrubbedLine, err := s.processLogLine(line)
		if err != nil {
			if s.verbose {
				fmt.Printf("Warning: Failed to process line %d: %v\n", lineCount, err)
			}
			// Write original line if processing fails
			scrubbedLine = line
		}

		processedCount++

		if !dryRun {
			if _, err := outputFile.WriteString(scrubbedLine + "\n"); err != nil {
				return fmt.Errorf("failed to write to output file: %w", err)
			}
		} else if s.verbose {
			fmt.Printf("Line %d would be scrubbed\n", lineCount)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input file: %w", err)
	}

	if s.verbose {
		fmt.Printf("Processed %d lines out of %d total lines\n", processedCount, lineCount)
	}

	return nil
}

// processLogLine processes a single log line and returns the scrubbed version
func (s *Scrubber) processLogLine(line string) (string, error) {
	// Try to parse as JSON
	var rawData map[string]interface{}
	if err := json.Unmarshal([]byte(line), &rawData); err != nil {
		// If not valid JSON, treat as plain text and scrub
		return s.scrubPlainText(line), nil
	}

	// Convert to JSON, scrub, and convert back
	jsonBytes, err := json.Marshal(rawData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	scrubbedJSON := s.scrubJSONString(string(jsonBytes))
	
	// Validate that the result is still valid JSON
	var temp interface{}
	if err := json.Unmarshal([]byte(scrubbedJSON), &temp); err != nil {
		// If scrubbing broke JSON, return original
		return line, nil
	}

	return scrubbedJSON, nil
}

// scrubJSONString scrubs sensitive data from a JSON string
func (s *Scrubber) scrubJSONString(jsonStr string) string {
	result := jsonStr

	// Scrub emails
	result = s.scrubEmails(result)

	// Scrub usernames
	result = s.scrubUsernames(result)

	// Scrub IP addresses
	result = s.scrubIPAddresses(result)

	// Scrub UIDs (for level 3 only)
	if s.level == 3 {
		result = s.scrubUIDs(result)
	}

	return result
}

// scrubPlainText scrubs sensitive data from plain text
func (s *Scrubber) scrubPlainText(text string) string {
	result := text

	// Scrub emails
	result = s.scrubEmails(result)

	// Scrub usernames (simple word boundary approach)
	result = s.scrubUsernames(result)

	// Scrub IP addresses
	result = s.scrubIPAddresses(result)

	// Scrub UIDs (for level 3 only)
	if s.level == 3 {
		result = s.scrubUIDs(result)
	}

	return result
}

// Email regex pattern
var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

func (s *Scrubber) scrubEmails(text string) string {
	return emailRegex.ReplaceAllStringFunc(text, func(email string) string {
		if scrubbed, exists := s.emailMap[email]; exists {
			return scrubbed
		}

		scrubbed := s.scrubEmailByLevel(email)
		s.emailMap[email] = scrubbed
		return scrubbed
	})
}

// IP address regex pattern
var ipRegex = regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)

func (s *Scrubber) scrubIPAddresses(text string) string {
	return ipRegex.ReplaceAllStringFunc(text, func(ip string) string {
		if scrubbed, exists := s.ipMap[ip]; exists {
			return scrubbed
		}

		scrubbed := s.scrubIPByLevel(ip)
		s.ipMap[ip] = scrubbed
		return scrubbed
	})
}

// Username patterns - look for quoted usernames in JSON and word boundaries in plain text
var usernameRegex = regexp.MustCompile(`"(?:user|username|name)"\s*:\s*"([^"]+)"`)

func (s *Scrubber) scrubUsernames(text string) string {
	// Scrub usernames in JSON format
	result := usernameRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract just the username value
		parts := strings.Split(match, `":"`)
		if len(parts) != 2 {
			return match
		}
		
		key := parts[0] + `":"`
		username := strings.TrimSuffix(parts[1], `"`)
		
		if scrubbed, exists := s.userMap[username]; exists {
			return key + scrubbed + `"`
		}

		scrubbed := s.scrubUsernameByLevel(username)
		s.userMap[username] = scrubbed
		return key + scrubbed + `"`
	})

	return result
}

// UID patterns - look for long alphanumeric strings that look like IDs
var uidRegex = regexp.MustCompile(`\b[a-f0-9]{20,}\b`)

func (s *Scrubber) scrubUIDs(text string) string {
	return uidRegex.ReplaceAllStringFunc(text, func(uid string) string {
		if len(uid) < 20 {
			return uid
		}

		if scrubbed, exists := s.uidMap[uid]; exists {
			return scrubbed
		}

		scrubbed := s.scrubUIDByLevel(uid)
		s.uidMap[uid] = scrubbed
		return scrubbed
	})
}