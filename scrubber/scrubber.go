package scrubber

import (
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	Source          string // source filename
}

type JSONFailure struct {
	LineNumber int
	Error      string
	SampleContent string // First 100 chars of the problematic line
}

type Scrubber struct {
	level            int
	verbose          bool
	emailMap         map[string]string
	userMap          map[string]string
	ipMap            map[string]string
	uidMap           map[string]string
	userMappings     map[string]*UserMapping // key: username or email -> UserMapping
	userCounter      int
	auditEntries     map[string]*AuditEntry // key: original value -> AuditEntry
	domainMap        map[string]string      // key: original domain -> mapped domain
	domainCounter    int
	jsonSuccessCount int
	jsonFailureCount int
	jsonFailures     []JSONFailure // Store sample of failed lines
}

func NewScrubber(level int, verbose bool) *Scrubber {
	return &Scrubber{
		level:            level,
		verbose:          verbose,
		emailMap:         make(map[string]string),
		userMap:          make(map[string]string),
		ipMap:            make(map[string]string),
		uidMap:           make(map[string]string),
		userMappings:     make(map[string]*UserMapping),
		userCounter:      0,
		auditEntries:     make(map[string]*AuditEntry),
		domainMap:        make(map[string]string),
		domainCounter:    0,
		jsonSuccessCount: 0,
		jsonFailureCount: 0,
		jsonFailures:     make([]JSONFailure, 0),
	}
}

// ProcessFile processes the input file and writes scrubbed output
// Returns the actual output path used (which may differ from inputPath if renamed)
func (s *Scrubber) ProcessFile(inputPath, outputPath string, dryRun bool, compress bool, overwriteAction string) (string, error) {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	var outputWriter io.Writer
	var outputFile *os.File
	var gzipWriter *gzip.Writer
	
	// Track the final output path (may change if renamed)
	finalOutputPath := outputPath
	
	if !dryRun {
		// Check if output file already exists
		if checkFileExists(outputPath) {
			choice, err := handleFileConflict(outputPath, overwriteAction)
			if err != nil {
				return "", fmt.Errorf("failed to handle file conflict: %w", err)
			}
			
			switch choice {
			case "cancel":
				return "", createCancelError(outputPath, overwriteAction)
			case "rename":
				finalOutputPath = generateTimestampSuffix(outputPath)
				fmt.Printf("Output will be written to: %s\n", finalOutputPath)
			case "overwrite":
				// Continue with original path
			}
		}
		
		outputFile, err = os.Create(finalOutputPath)
		if err != nil {
			return "", fmt.Errorf("failed to create output file: %w", err)
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

		scrubbedLine, err := s.processLogLine(line, filepath.Base(inputPath), lineCount)
		if err != nil {
			failedCount++
			fmt.Printf("\nWarning: Failed to process line %d: %v\n", lineCount, err)
			// Write original line if processing fails
			scrubbedLine = line
		}

		processedCount++

		if !dryRun {
			if _, err := outputWriter.Write([]byte(scrubbedLine + "\n")); err != nil {
				return "", fmt.Errorf("failed to write to output file: %w", err)
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
		return "", fmt.Errorf("error reading input file: %w", err)
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
	
	// Show JSON processing statistics
	if s.jsonSuccessCount > 0 || s.jsonFailureCount > 0 {
		totalProcessed := s.jsonSuccessCount + s.jsonFailureCount
		if totalProcessed > 0 {
			jsonPercent := float64(s.jsonSuccessCount) / float64(totalProcessed) * 100
			plainPercent := float64(s.jsonFailureCount) / float64(totalProcessed) * 100
			fmt.Printf("JSON processed: %d lines (%.1f%%)\n", s.jsonSuccessCount, jsonPercent)
			fmt.Printf("Plain text processed: %d lines (%.1f%%)\n", s.jsonFailureCount, plainPercent)
		}
	}
	
	// Show JSON issues summary if any occurred
	if s.jsonFailureCount > 0 {
		fmt.Printf("\nJSON Processing Issues:\n")
		fmt.Printf("  %d lines had JSON parsing issues and were processed as plain text\n", s.jsonFailureCount)
		
		// Show line numbers of first few failures
		if len(s.jsonFailures) > 0 {
			fmt.Print("  Lines with issues: ")
			for i, failure := range s.jsonFailures {
				if i >= 5 { // Show first 5 line numbers
					fmt.Printf("... and %d more", s.jsonFailureCount-5)
					break
				}
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%d", failure.LineNumber)
			}
			fmt.Println()
		}
		
		// In verbose mode, show detailed sample of failed lines
		if s.verbose && len(s.jsonFailures) > 0 {
			fmt.Println("  Sample failure details:")
			for i, failure := range s.jsonFailures {
				if i >= 3 { // Limit to first 3 in verbose output
					fmt.Printf("    ... and %d more failures\n", len(s.jsonFailures)-3)
					break
				}
				fmt.Printf("    Line %d: %s\n", failure.LineNumber, failure.SampleContent)
				fmt.Printf("      Error: %s\n", failure.Error)
			}
		}
	}

	// Return the actual path used (for dry run, return original path)
	if dryRun {
		return outputPath, nil
	}
	return finalOutputPath, nil
}

// processLogLine processes a single log line and returns the scrubbed version
func (s *Scrubber) processLogLine(line, source string, lineNumber int) (string, error) {
	// Try to parse as JSON to validate and extract user mapping data
	var rawData map[string]interface{}
	if err := json.Unmarshal([]byte(line), &rawData); err != nil {
		// Track JSON failure and show warning
		s.trackJSONFailure(lineNumber, line, err)
		return s.scrubPlainText(line, source), nil
	}

	// Successfully parsed as JSON
	s.jsonSuccessCount++
	
	// If using mapping mode, detect and create user mappings first
	// Always detect and create user mappings
	s.detectAndMapUser(rawData)

	// Work directly with the JSON string to preserve field order
	scrubbedJSON := s.scrubJSONString(line, source)
	
	// Validate that the result is still valid JSON
	var temp interface{}
	if err := json.Unmarshal([]byte(scrubbedJSON), &temp); err != nil {
		// If scrubbing broke JSON, return original
		return line, nil
	}

	return scrubbedJSON, nil
}

// scrubJSONString scrubs sensitive data from a JSON string
func (s *Scrubber) scrubJSONString(jsonStr, source string) string {
	result := jsonStr

	// Scrub emails (all levels)
	result = s.scrubEmails(result, source)

	// Scrub usernames (all levels)
	result = s.scrubUsernames(result, source)

	// Scrub IP addresses (levels 2 and 3 only)
	if s.level >= 2 {
		result = s.scrubIPAddresses(result, source)
	}

	// Scrub UIDs (level 3 only)
	if s.level == 3 {
		result = s.scrubUIDs(result, source)
	}

	return result
}

// scrubPlainText scrubs sensitive data from plain text
func (s *Scrubber) scrubPlainText(text, source string) string {
	result := text

	// Scrub emails (all levels)
	result = s.scrubEmails(result, source)

	// Scrub usernames (all levels)
	result = s.scrubUsernames(result, source)

	// Scrub IP addresses (levels 2 and 3 only)
	if s.level >= 2 {
		result = s.scrubIPAddresses(result, source)
	}

	// Scrub UIDs (level 3 only)
	if s.level == 3 {
		result = s.scrubUIDs(result, source)
	}

	return result
}

// Email regex pattern
var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

func (s *Scrubber) scrubEmails(text, source string) string {
	return emailRegex.ReplaceAllStringFunc(text, func(email string) string {
		emailLower := strings.ToLower(email)
		if scrubbed, exists := s.emailMap[emailLower]; exists {
			s.trackReplacement(email, scrubbed, constants.TypeEmail, source)
			return scrubbed
		}

		// Always use user mapping for emails
		scrubbed := s.getUserMappedEmail(email)
		
		s.emailMap[emailLower] = scrubbed
		s.trackReplacement(email, scrubbed, constants.TypeEmail, source)
		return scrubbed
	})
}

// IP address regex pattern
var ipRegex = regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)

func (s *Scrubber) scrubIPAddresses(text, source string) string {
	return ipRegex.ReplaceAllStringFunc(text, func(ip string) string {
		if scrubbed, exists := s.ipMap[ip]; exists {
			s.trackReplacement(ip, scrubbed, constants.TypeIP, source)
			return scrubbed
		}

		scrubbed := s.scrubIPByLevel(ip)
		s.ipMap[ip] = scrubbed
		s.trackReplacement(ip, scrubbed, constants.TypeIP, source)
		return scrubbed
	})
}

// Username patterns - look for quoted usernames in JSON and word boundaries in plain text
var usernameRegex = regexp.MustCompile(`"(?:user|username)"\s*:\s*"([^"]+)"`)

func (s *Scrubber) scrubUsernames(text, source string) string {
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
			s.trackReplacement(username, scrubbed, constants.TypeUsername, source)
			return key + scrubbed + `"`
		}

		// Always use user mapping for usernames
		scrubbed := s.getUserMappedName(username)
		
		s.userMap[usernameLower] = scrubbed
		s.trackReplacement(username, scrubbed, constants.TypeUsername, source)
		return key + scrubbed + `"`
	})

	return result
}

// UID patterns - look for long alphanumeric strings that look like IDs
var uidRegex = regexp.MustCompile(`\b[a-z0-9]{` + fmt.Sprintf("%d", constants.MinUIDLength) + `,}\b`)

func (s *Scrubber) scrubUIDs(text, source string) string {
	return uidRegex.ReplaceAllStringFunc(text, func(uid string) string {
		if len(uid) < constants.MinUIDLength {
			return uid
		}

		if scrubbed, exists := s.uidMap[uid]; exists {
			s.trackReplacement(uid, scrubbed, constants.TypeUID, source)
			return scrubbed
		}

		scrubbed := s.scrubUIDByLevel(uid)
		s.uidMap[uid] = scrubbed
		s.trackReplacement(uid, scrubbed, constants.TypeUID, source)
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
func (s *Scrubber) trackReplacement(original, newValue, valueType, source string) {
	if entry, exists := s.auditEntries[original]; exists {
		entry.TimesReplaced++
	} else {
		s.auditEntries[original] = &AuditEntry{
			OriginalValue: original,
			NewValue:      newValue,
			TimesReplaced: 1,
			Type:          valueType,
			Source:        source,
		}
	}
}

// WriteAuditFile writes the audit log to a CSV file
func (s *Scrubber) WriteAuditFile(filePath string, overwriteAction string) (string, error) {
	// Check if audit file already exists
	finalAuditPath := filePath
	if checkFileExists(filePath) {
		choice, err := handleFileConflict(filePath, overwriteAction)
		if err != nil {
			return "", fmt.Errorf("failed to handle file conflict: %w", err)
		}
		
		switch choice {
		case "cancel":
			return "", createCancelError(filePath, overwriteAction)
		case "rename":
			finalAuditPath = generateTimestampSuffix(filePath)
			fmt.Printf("Audit file will be written to: %s\n", finalAuditPath)
		case "overwrite":
			// Continue with original path
		}
	}
	
	file, err := os.Create(finalAuditPath)
	if err != nil {
		return "", fmt.Errorf("failed to create audit file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"Original Value", "New Value", "Times Replaced", "Type", "Source"}); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write audit entries
	for _, entry := range s.auditEntries {
		record := []string{
			entry.OriginalValue,
			entry.NewValue,
			fmt.Sprintf("%d", entry.TimesReplaced),
			entry.Type,
			entry.Source,
		}
		if err := writer.Write(record); err != nil {
			return "", fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return finalAuditPath, nil
}

// trackJSONFailure records a JSON parsing failure for reporting
func (s *Scrubber) trackJSONFailure(lineNumber int, line string, err error) {
	s.jsonFailureCount++
	
	// Store sample of failed lines (limit to first 10 to avoid memory issues)
	if len(s.jsonFailures) < 10 {
		sampleContent := line
		if len(sampleContent) > 100 {
			sampleContent = sampleContent[:100] + "..."
		}
		
		s.jsonFailures = append(s.jsonFailures, JSONFailure{
			LineNumber:    lineNumber,
			Error:         err.Error(),
			SampleContent: sampleContent,
		})
	}
	
	// Don't show warning immediately to avoid interrupting progress
	// Warnings will be shown at the end during statistics
}

// checkFileExists returns true if the file exists
func checkFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// createCancelError creates an appropriate error message based on the overwrite action
func createCancelError(filePath string, overwriteAction string) error {
	switch overwriteAction {
	case constants.OverwriteCancel:
		return fmt.Errorf("file '%s' already exists and OverwriteAction is set to 'cancel'", filePath)
	default:
		return fmt.Errorf("operation cancelled by user")
	}
}

// handleFileConflict determines how to handle an existing file based on the overwrite action
// Returns: "overwrite", "cancel", or "rename"
func handleFileConflict(filePath string, overwriteAction string) (string, error) {
	switch overwriteAction {
	case constants.OverwriteOverwrite:
		return "overwrite", nil
	case constants.OverwriteTimestamp:
		return "rename", nil
	case constants.OverwriteCancel:
		return "cancel", nil
	case constants.OverwritePrompt:
		return promptUserChoice(filePath)
	default:
		// Fallback to prompting if invalid action
		return promptUserChoice(filePath)
	}
}

// promptUserChoice prompts the user to choose how to handle an existing file
// Returns: "overwrite", "cancel", or "rename"
func promptUserChoice(filePath string) (string, error) {
	fmt.Printf("File '%s' already exists.\n", filePath)
	fmt.Print("Choose an option: (o)verwrite, (c)ancel, or (r)ename with timestamp? ")
	
	var choice string
	_, err := fmt.Scanln(&choice)
	if err != nil {
		return "", fmt.Errorf("failed to read user input: %w", err)
	}
	
	choice = strings.ToLower(strings.TrimSpace(choice))
	switch choice {
	case "o", "overwrite":
		return "overwrite", nil
	case "c", "cancel":
		return "cancel", nil
	case "r", "rename":
		return "rename", nil
	default:
		fmt.Println("Invalid choice. Please enter 'o', 'c', or 'r'.")
		return promptUserChoice(filePath) // Recursive call for invalid input
	}
}

// generateTimestampSuffix creates a timestamp suffix for filenames
func generateTimestampSuffix(originalPath string) string {
	timestamp := time.Now().Format("20060102_150405")
	
	// Split the path into directory, name, and extension
	dir := filepath.Dir(originalPath)
	base := filepath.Base(originalPath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)
	
	newName := fmt.Sprintf("%s_%s%s", nameWithoutExt, timestamp, ext)
	return filepath.Join(dir, newName)
}

// WriteAuditFileJSON writes the audit log to a JSON file
// Returns the actual file path used (which may differ if renamed)
func (s *Scrubber) WriteAuditFileJSON(filePath string, overwriteAction string) (string, error) {
	// Check if audit file already exists
	finalAuditPath := filePath
	if checkFileExists(filePath) {
		choice, err := handleFileConflict(filePath, overwriteAction)
		if err != nil {
			return "", fmt.Errorf("failed to handle file conflict: %w", err)
		}
		
		switch choice {
		case "cancel":
			return "", createCancelError(filePath, overwriteAction)
		case "rename":
			finalAuditPath = generateTimestampSuffix(filePath)
			fmt.Printf("Audit file will be written to: %s\n", finalAuditPath)
		case "overwrite":
			// Continue with original path
		}
	}
	
	file, err := os.Create(finalAuditPath)
	if err != nil {
		return "", fmt.Errorf("failed to create audit file: %w", err)
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
		return "", fmt.Errorf("failed to write JSON audit file: %w", err)
	}

	return finalAuditPath, nil
}