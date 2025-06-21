package scrubber

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type UserMapping struct {
	Username string
	Email    string
	MappedID int
}

type Scrubber struct {
	level       int
	verbose     bool
	emailMap    map[string]string
	userMap     map[string]string
	ipMap       map[string]string
	uidMap      map[string]string
	userMappings map[string]*UserMapping // key: username or email -> UserMapping
	userCounter  int
}

func NewScrubber(level int, verbose bool) *Scrubber {
	return &Scrubber{
		level:        level,
		verbose:      verbose,
		emailMap:     make(map[string]string),
		userMap:      make(map[string]string),
		ipMap:        make(map[string]string),
		uidMap:       make(map[string]string),
		userMappings: make(map[string]*UserMapping),
		userCounter:  0,
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
	emptyCount := 0
	failedCount := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		
		if strings.TrimSpace(line) == "" {
			emptyCount++
			continue
		}

		scrubbedLine, err := s.processLogLine(line)
		if err != nil {
			failedCount++
			fmt.Printf("Warning: Failed to process line %d: %v\n", lineCount, err)
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

	// Always show processed lines count with breakdown
	fmt.Printf("Processed %d lines out of %d total lines", processedCount, lineCount)
	if emptyCount > 0 {
		fmt.Printf(" (%d empty lines skipped)", emptyCount)
	}
	if failedCount > 0 {
		fmt.Printf(" (%d lines failed processing but were included)", failedCount)
	}
	fmt.Println()

	return nil
}

// processLogLine processes a single log line and returns the scrubbed version
func (s *Scrubber) processLogLine(line string) (string, error) {
	// Try to parse as JSON to validate and extract user mapping data
	var rawData map[string]interface{}
	if err := json.Unmarshal([]byte(line), &rawData); err != nil {
		// If not valid JSON, treat as plain text and scrub
		return s.scrubPlainText(line), nil
	}

	// If using mapping mode, detect and create user mappings first
	// Always detect and create user mappings
	s.detectAndMapUser(rawData)

	// Work directly with the JSON string to preserve field order
	scrubbedJSON := s.scrubJSONString(line)
	
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

	// Scrub emails (all levels)
	result = s.scrubEmails(result)

	// Scrub usernames (all levels)
	result = s.scrubUsernames(result)

	// Scrub IP addresses (levels 2 and 3 only)
	if s.level >= 2 {
		result = s.scrubIPAddresses(result)
	}

	// Scrub UIDs (level 3 only)
	if s.level == 3 {
		result = s.scrubUIDs(result)
	}

	return result
}

// scrubPlainText scrubs sensitive data from plain text
func (s *Scrubber) scrubPlainText(text string) string {
	result := text

	// Scrub emails (all levels)
	result = s.scrubEmails(result)

	// Scrub usernames (all levels)
	result = s.scrubUsernames(result)

	// Scrub IP addresses (levels 2 and 3 only)
	if s.level >= 2 {
		result = s.scrubIPAddresses(result)
	}

	// Scrub UIDs (level 3 only)
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

		// Always use user mapping for emails
		scrubbed := s.getUserMappedEmail(email)
		
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
var usernameRegex = regexp.MustCompile(`"(?:user|username)"\s*:\s*"([^"]+)"`)

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

		// Always use user mapping for usernames
		scrubbed := s.getUserMappedName(username)
		
		s.userMap[username] = scrubbed
		return key + scrubbed + `"`
	})

	return result
}

// UID patterns - look for long alphanumeric strings that look like IDs
var uidRegex = regexp.MustCompile(`\b[a-z0-9]{20,}\b`)

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

// detectAndMapUser detects username and email pairs in JSON data and creates user mappings
func (s *Scrubber) detectAndMapUser(data map[string]interface{}) {
	s.findUserMappingsRecursive(data)
}

// findUserMappingsRecursive recursively searches through JSON data to find username/email pairs
func (s *Scrubber) findUserMappingsRecursive(data interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		// Check if this object has both username and email fields
		var username, email string
		
		// Look for username fields in this object
		if userVal, exists := v["user"]; exists {
			if userStr, ok := userVal.(string); ok {
				username = userStr
			}
		} else if userVal, exists := v["username"]; exists {
			if userStr, ok := userVal.(string); ok {
				username = userStr
			}
		}
		
		// Look for email field in this object
		if emailVal, exists := v["email"]; exists {
			if emailStr, ok := emailVal.(string); ok {
				email = emailStr
			}
		}
		
		// If we found both username and email in this object, create mapping
		if username != "" && email != "" {
			s.createUserMapping(username, email)
		}
		
		// Recursively search all nested objects
		for _, value := range v {
			s.findUserMappingsRecursive(value)
		}
		
	case []interface{}:
		// Recursively search all array elements
		for _, item := range v {
			s.findUserMappingsRecursive(item)
		}
	}
}

// createUserMapping creates a mapping for a username/email pair
func (s *Scrubber) createUserMapping(username, email string) {
	// Check if we already have a mapping for either username or email
	if mapping, exists := s.userMappings[username]; exists {
		// Link the email to existing mapping if not already linked
		if mapping.Email == "" {
			mapping.Email = email
			s.userMappings[email] = mapping
		}
		return
	}
	
	if mapping, exists := s.userMappings[email]; exists {
		// Link the username to existing mapping if not already linked
		if mapping.Username == "" {
			mapping.Username = username
			s.userMappings[username] = mapping
		}
		return
	}
	
	// Create new user mapping
	s.userCounter++
	mapping := &UserMapping{
		Username: username,
		Email:    email,
		MappedID: s.userCounter,
	}
	
	s.userMappings[username] = mapping
	s.userMappings[email] = mapping
	
	if s.verbose {
		fmt.Printf("Created user mapping: %s / %s -> user%d\n", username, email, s.userCounter)
	}
}

// getUserMappedName returns the mapped username for a given original username
func (s *Scrubber) getUserMappedName(username string) string {
	if mapping, exists := s.userMappings[username]; exists {
		return fmt.Sprintf("user%d", mapping.MappedID)
	}
	// If no mapping exists, create one for standalone username
	s.userCounter++
	mapping := &UserMapping{
		Username: username,
		MappedID: s.userCounter,
	}
	s.userMappings[username] = mapping
	
	if s.verbose {
		fmt.Printf("Created standalone user mapping: %s -> user%d\n", username, s.userCounter)
	}
	
	return fmt.Sprintf("user%d", mapping.MappedID)
}

// getUserMappedEmail returns the mapped email for a given original email
func (s *Scrubber) getUserMappedEmail(email string) string {
	if mapping, exists := s.userMappings[email]; exists {
		return fmt.Sprintf("user%d@domain.com", mapping.MappedID)
	}
	// If no mapping exists, create one for standalone email
	s.userCounter++
	mapping := &UserMapping{
		Email: email,
		MappedID: s.userCounter,
	}
	s.userMappings[email] = mapping
	
	if s.verbose {
		fmt.Printf("Created standalone email mapping: %s -> user%d@domain.com\n", email, s.userCounter)
	}
	
	return fmt.Sprintf("user%d@domain.com", mapping.MappedID)
}