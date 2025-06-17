package scrubber

import (
	"strings"
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
	case 1:
		// Keep last 3 characters of local part
		if len(localPart) <= 3 {
			return strings.Repeat("x", len(localPart)) + "@" + domain
		}
		masked := strings.Repeat("x", len(localPart)-3) + localPart[len(localPart)-3:]
		return masked + "@" + domain

	case 2:
		// Mask entire local part
		masked := strings.Repeat("x", len(localPart))
		return masked + "@" + domain

	case 3:
		// Mask everything including domain
		localMasked := strings.Repeat("x", len(localPart))
		domainMasked := strings.Repeat("x", len(domain))
		return localMasked + "@" + domainMasked

	default:
		return email
	}
}

// scrubUsernameByLevel scrubs usernames based on the scrubbing level
func (s *Scrubber) scrubUsernameByLevel(username string) string {
	switch s.level {
	case 1:
		// Keep last 3 characters
		if len(username) <= 3 {
			return strings.Repeat("x", len(username))
		}
		return strings.Repeat("x", len(username)-3) + username[len(username)-3:]

	case 2, 3:
		// Mask entire username
		return strings.Repeat("x", len(username))

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
	case 1:
		// Keep last octet only
		return "xxx.xxx.xxx." + parts[3]

	case 2, 3:
		// Mask entire IP
		return "xxx.xxx.xxx.xxx"

	default:
		return ip
	}
}

// scrubUIDByLevel scrubs UIDs/Channel IDs/Team IDs based on the scrubbing level (level 3 only)
func (s *Scrubber) scrubUIDByLevel(uid string) string {
	if s.level != 3 {
		return uid // Don't scrub UIDs for levels 1 and 2
	}

	// For level 3: mask all but last 4 digits, keep total length at 26
	if len(uid) < 4 {
		return strings.Repeat("x", len(uid))
	}

	last4 := uid[len(uid)-4:]
	
	// Ensure total length is 26
	maskedLength := 26 - 4
	if maskedLength < 0 {
		maskedLength = len(uid) - 4
	}
	
	masked := strings.Repeat("x", maskedLength)
	return masked + last4
}