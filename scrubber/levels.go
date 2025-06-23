package scrubber

import (
	"strings"

	"mattermost-log-scrubber/constants"
)

// scrubEmailByLevel scrubs email addresses based on the scrubbing level
func (s *Scrubber) scrubEmailByLevel(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email // Invalid email format
	}

	localPart := parts[0]
	domain := parts[1]

	switch s.level {
	case constants.ScrubLevelLow:
		// Keep last 3 characters of local part
		if len(localPart) <= 3 {
			return strings.Repeat("*", len(localPart)) + "@" + domain
		}
		masked := strings.Repeat("*", len(localPart)-3) + localPart[len(localPart)-3:]
		return masked + "@" + domain

	case constants.ScrubLevelMedium:
		// Mask entire local part
		masked := strings.Repeat("*", len(localPart))
		return masked + "@" + domain

	case constants.ScrubLevelHigh:
		// Mask everything including domain
		localMasked := strings.Repeat("*", len(localPart))
		domainMasked := strings.Repeat("*", len(domain))
		return localMasked + "@" + domainMasked

	default:
		return email
	}
}

// scrubUsernameByLevel scrubs usernames based on the scrubbing level
func (s *Scrubber) scrubUsernameByLevel(username string) string {
	switch s.level {
	case constants.ScrubLevelLow:
		// Keep last 3 characters
		if len(username) <= 3 {
			return strings.Repeat("*", len(username))
		}
		return strings.Repeat("*", len(username)-3) + username[len(username)-3:]

	case constants.ScrubLevelMedium, constants.ScrubLevelHigh:
		// Mask entire username
		return strings.Repeat("*", len(username))

	default:
		return username
	}
}

// scrubIPByLevel scrubs IP addresses based on the scrubbing level
func (s *Scrubber) scrubIPByLevel(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip // Invalid IP format
	}

	switch s.level {
	case constants.ScrubLevelMedium:
		// Keep last octet only
		return "***.***.***." + parts[3]

	case constants.ScrubLevelHigh:
		// Mask entire IP
		return "***.***.***.***"

	default:
		return ip
	}
}

// scrubUIDByLevel scrubs UIDs/Channel IDs/Team IDs based on the scrubbing level (level 3 only)
func (s *Scrubber) scrubUIDByLevel(uid string) string {
	if s.level != constants.ScrubLevelHigh {
		return uid // Don't scrub UIDs for levels 1 and 2
	}

	// For level 3: mask all but last 8 characters, keep total length at 26
	if len(uid) < constants.UIDKeepChars {
		return strings.Repeat("*", len(uid))
	}

	lastChars := uid[len(uid)-constants.UIDKeepChars:]
	
	// Ensure total length is 26
	maskedLength := constants.UIDTargetLength - constants.UIDKeepChars
	if maskedLength < 0 {
		maskedLength = len(uid) - constants.UIDKeepChars
	}
	
	masked := strings.Repeat("*", maskedLength)
	return masked + lastChars
}

