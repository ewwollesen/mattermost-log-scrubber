package scrubber

import (
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"mattermost-log-scrubber/constants"
)

type UserMapping struct {
	Username string
	Email    string
	MappedID int
}

type AuditEntry struct {
	OriginalValue   string
	NewValue        string
	TimesReplaced   int
	Type            string // "email", "username", "ip", "uid"
}

type Scrubber struct {
	level        int
	verbose      bool
	emailMap     map[string]string
	userMap      map[string]string
	ipMap        map[string]string
	uidMap       map[string]string
	userMappings map[string]*UserMapping // key: username or email -> UserMapping
	userCounter  int
	auditEntries map[string]*AuditEntry // key: original value -> AuditEntry
	domainMap    map[string]string      // key: original domain -> mapped domain
	domainCounter int
}

func NewScrubber(level int, verbose bool) *Scrubber {
	return &Scrubber{
		level:         level,
		verbose:       verbose,
		emailMap:      make(map[string]string),
		userMap:       make(map[string]string),
		ipMap:         make(map[string]string),
		uidMap:        make(map[string]string),
		userMappings:  make(map[string]*UserMapping),
		userCounter:   0,
		auditEntries:  make(map[string]*AuditEntry),
		domainMap:     make(map[string]string),
		domainCounter: 0,
	}
}

// ProcessFile processes the input file and writes scrubbed output
func (s *Scrubber) ProcessFile(inputPath, outputPath string, dryRun bool, compress bool) error {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	var outputWriter io.Writer
	var outputFile *os.File
	var gzipWriter *gzip.Writer
	
	if !dryRun {
		outputFile, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outputFile.Close()
		
		if compress {
			gzipWriter = gzip.NewWriter(outputFile)
			defer gzipWriter.Close()
			outputWriter = gzipWriter
		} else {
			outputWriter = outputFile
		}
	}

	scanner := bufio.NewScanner(inputFile)
	lineCount := 0
	processedCount := 0
	emptyCount := 0
	failedCount := 0
	
	// Progress tracking (only if not verbose)
	var startTime, lastProgressTime time.Time
	progressInterval := constants.ProgressInterval // Show progress every N lines
	
	if !s.verbose {
		startTime = time.Now()
		lastProgressTime = startTime
		fmt.Print("Processing... ")
	}

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
			fmt.Printf("\nWarning: Failed to process line %d: %v\n", lineCount, err)
			// Write original line if processing fails
			scrubbedLine = line
		}

		processedCount++

		if !dryRun {
			if _, err := outputWriter.Write([]byte(scrubbedLine + "\n")); err != nil {
				return fmt.Errorf("failed to write to output file: %w", err)
			}
		} else if s.verbose {
			fmt.Printf("Line %d would be scrubbed\n", lineCount)
		}
		
		// Show progress every 1000 lines or every second (only if not verbose)
		if !s.verbose {
			now := time.Now()
			if lineCount%progressInterval == 0 || now.Sub(lastProgressTime) >= time.Second {
				fmt.Printf("\rProcessing... %d lines", lineCount)
				lastProgressTime = now
			}
		}
	}
	
	// Clear progress line (only if not verbose)
	if !s.verbose {
		fmt.Print("\r" + strings.Repeat(" ", 50) + "\r")
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
		emailLower := strings.ToLower(email)
		if scrubbed, exists := s.emailMap[emailLower]; exists {
			s.trackReplacement(email, scrubbed, constants.TypeEmail)
			return scrubbed
		}

		// Always use user mapping for emails
		scrubbed := s.getUserMappedEmail(email)
		
		s.emailMap[emailLower] = scrubbed
		s.trackReplacement(email, scrubbed, constants.TypeEmail)
		return scrubbed
	})
}

// IP address regex pattern
var ipRegex = regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)

func (s *Scrubber) scrubIPAddresses(text string) string {
	return ipRegex.ReplaceAllStringFunc(text, func(ip string) string {
		if scrubbed, exists := s.ipMap[ip]; exists {
			s.trackReplacement(ip, scrubbed, constants.TypeIP)
			return scrubbed
		}

		scrubbed := s.scrubIPByLevel(ip)
		s.ipMap[ip] = scrubbed
		s.trackReplacement(ip, scrubbed, constants.TypeIP)
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
		
		usernameLower := strings.ToLower(username)
		if scrubbed, exists := s.userMap[usernameLower]; exists {
			s.trackReplacement(username, scrubbed, constants.TypeUsername)
			return key + scrubbed + `"`
		}

		// Always use user mapping for usernames
		scrubbed := s.getUserMappedName(username)
		
		s.userMap[usernameLower] = scrubbed
		s.trackReplacement(username, scrubbed, constants.TypeUsername)
		return key + scrubbed + `"`
	})

	return result
}

// UID patterns - look for long alphanumeric strings that look like IDs
var uidRegex = regexp.MustCompile(`\b[a-z0-9]{` + fmt.Sprintf("%d", constants.MinUIDLength) + `,}\b`)

func (s *Scrubber) scrubUIDs(text string) string {
	return uidRegex.ReplaceAllStringFunc(text, func(uid string) string {
		if len(uid) < constants.MinUIDLength {
			return uid
		}

		if scrubbed, exists := s.uidMap[uid]; exists {
			s.trackReplacement(uid, scrubbed, constants.TypeUID)
			return scrubbed
		}

		scrubbed := s.scrubUIDByLevel(uid)
		s.uidMap[uid] = scrubbed
		s.trackReplacement(uid, scrubbed, constants.TypeUID)
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
	// Normalize case for consistent lookups
	usernameLower := strings.ToLower(username)
	emailLower := strings.ToLower(email)
	
	// Check if we already have a mapping for either username or email (case insensitive)
	if mapping, exists := s.userMappings[usernameLower]; exists {
		// Link the email to existing mapping if not already linked
		if mapping.Email == "" {
			mapping.Email = email
			s.userMappings[emailLower] = mapping
		}
		return
	}
	
	if mapping, exists := s.userMappings[emailLower]; exists {
		// Link the username to existing mapping if not already linked
		if mapping.Username == "" {
			mapping.Username = username
			s.userMappings[usernameLower] = mapping
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
	
	s.userMappings[usernameLower] = mapping
	s.userMappings[emailLower] = mapping
	
	if s.verbose {
		fmt.Printf("Created user mapping: %s / %s -> user%d\n", username, email, s.userCounter)
	}
}

// getUserMappedName returns the mapped username for a given original username
func (s *Scrubber) getUserMappedName(username string) string {
	usernameLower := strings.ToLower(username)
	if mapping, exists := s.userMappings[usernameLower]; exists {
		return fmt.Sprintf("user%d", mapping.MappedID)
	}
	// If no mapping exists, create one for standalone username
	s.userCounter++
	mapping := &UserMapping{
		Username: username,
		MappedID: s.userCounter,
	}
	s.userMappings[usernameLower] = mapping
	
	if s.verbose {
		fmt.Printf("Created standalone user mapping: %s -> user%d\n", username, s.userCounter)
	}
	
	return fmt.Sprintf("user%d", mapping.MappedID)
}

// getUserMappedEmail returns the mapped email for a given original email
func (s *Scrubber) getUserMappedEmail(email string) string {
	emailLower := strings.ToLower(email)
	if mapping, exists := s.userMappings[emailLower]; exists {
		return fmt.Sprintf("user%d@%s", mapping.MappedID, s.getMappedDomain(email))
	}
	// If no mapping exists, create one for standalone email
	s.userCounter++
	mapping := &UserMapping{
		Email: email,
		MappedID: s.userCounter,
	}
	s.userMappings[emailLower] = mapping
	
	if s.verbose {
		fmt.Printf("Created standalone email mapping: %s -> user%d@%s\n", email, s.userCounter, s.getMappedDomain(email))
	}
	
	return fmt.Sprintf("user%d@%s", mapping.MappedID, s.getMappedDomain(email))
}

// getMappedDomain returns the mapped domain for a given email address
func (s *Scrubber) getMappedDomain(email string) string {
	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return constants.DefaultDomain // fallback for invalid emails
	}
	
	originalDomain := strings.ToLower(parts[1])
	
	// Check if we already have a mapping for this domain
	if mappedDomain, exists := s.domainMap[originalDomain]; exists {
		return mappedDomain
	}
	
	// Create new domain mapping
	s.domainCounter++
	mappedDomain := fmt.Sprintf("domain%d.%s", s.domainCounter, constants.DefaultDomain)
	s.domainMap[originalDomain] = mappedDomain
	
	if s.verbose {
		fmt.Printf("Created domain mapping: %s -> %s\n", originalDomain, mappedDomain)
	}
	
	return mappedDomain
}

// trackReplacement tracks a replacement for audit purposes
func (s *Scrubber) trackReplacement(original, newValue, valueType string) {
	if entry, exists := s.auditEntries[original]; exists {
		entry.TimesReplaced++
	} else {
		s.auditEntries[original] = &AuditEntry{
			OriginalValue: original,
			NewValue:      newValue,
			TimesReplaced: 1,
			Type:          valueType,
		}
	}
}

// WriteAuditFile writes the audit log to a CSV file
func (s *Scrubber) WriteAuditFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create audit file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"Original Value", "New Value", "Times Replaced", "Type"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write audit entries
	for _, entry := range s.auditEntries {
		record := []string{
			entry.OriginalValue,
			entry.NewValue,
			fmt.Sprintf("%d", entry.TimesReplaced),
			entry.Type,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}

// WriteAuditFileJSON writes the audit log to a JSON file
func (s *Scrubber) WriteAuditFileJSON(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create audit file: %w", err)
	}
	defer file.Close()

	// Convert audit entries to a slice for JSON serialization
	auditData := make([]AuditEntry, 0, len(s.auditEntries))
	for _, entry := range s.auditEntries {
		auditData = append(auditData, *entry)
	}

	// Write JSON with proper formatting
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(auditData); err != nil {
		return fmt.Errorf("failed to write JSON audit file: %w", err)
	}

	return nil
}