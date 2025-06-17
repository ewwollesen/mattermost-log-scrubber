package models

import "encoding/json"

// MattermostLogEntry represents a generic Mattermost log entry
type MattermostLogEntry struct {
	Type    string          `json:"type,omitempty"`
	Level   string          `json:"level,omitempty"`
	Msg     string          `json:"msg,omitempty"`
	Time    string          `json:"time,omitempty"`
	User    string          `json:"user,omitempty"`
	UserID  string          `json:"user_id,omitempty"`
	Email   string          `json:"email,omitempty"`
	IP      string          `json:"ip,omitempty"`
	Team    string          `json:"team,omitempty"`
	TeamID  string          `json:"team_id,omitempty"`
	Channel string          `json:"channel,omitempty"`
	ChannelID string        `json:"channel_id,omitempty"`
	Post    *PostData       `json:"post,omitempty"`
	Raw     json.RawMessage `json:"-"`
}

// PostData represents post-specific data in log entries
type PostData struct {
	Team     string `json:"team,omitempty"`
	Channel  string `json:"channel,omitempty"`
	User     string `json:"user,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Message  string `json:"message,omitempty"`
	CreateAt int64  `json:"create_at,omitempty"`
}

// UnmarshalJSON custom unmarshaling to capture raw JSON
func (m *MattermostLogEntry) UnmarshalJSON(data []byte) error {
	// Store raw JSON for processing
	m.Raw = data
	
	// Create an alias type to avoid infinite recursion
	type Alias MattermostLogEntry
	alias := (*Alias)(m)
	
	return json.Unmarshal(data, alias)
}