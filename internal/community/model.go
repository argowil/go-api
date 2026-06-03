package community

import "time"

type Message struct {
	ID           uint      `db:"id"           json:"id"`
	UserID       uint      `db:"user_id"       json:"user_id"`
	UserName     string    `db:"user_name"     json:"user_name"`
	Content      string    `db:"content"       json:"content"`
	Edited       bool      `db:"edited"        json:"edited"`
	ReplyToID    *uint     `db:"reply_to_id"   json:"reply_to_id,omitempty"`
	ReplyPreview *string   `db:"reply_preview" json:"reply_preview,omitempty"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
}

// WSEvent wraps outgoing WebSocket messages with a type discriminator.
type WSEvent struct {
	Type    string   `json:"type"`              // "new" | "edit" | "delete" | "presence"
	Message *Message `json:"message,omitempty"`
	ID      uint     `json:"id,omitempty"`
	UserID  uint     `json:"user_id,omitempty"` // presence events
	Online  bool     `json:"online,omitempty"`  // presence events
}
