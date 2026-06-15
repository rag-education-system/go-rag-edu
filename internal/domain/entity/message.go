package entity

import "time"

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
)

type Message struct {
	ID             string      `db:"id" json:"id"`
	ConversationID string      `db:"conversationId" json:"conversationId"`
	Role           MessageRole `db:"role" json:"role"`
	Content        string      `db:"content" json:"content"`
	Sources        []byte      `db:"sources" json:"sources,omitempty"`
	Metadata       []byte      `db:"metadata" json:"metadata,omitempty"`
	CreatedAt      time.Time   `db:"createdAt" json:"createdAt"`
}
