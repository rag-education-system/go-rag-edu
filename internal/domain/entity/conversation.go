package entity

import "time"

type Conversation struct {
	ID         string     `db:"id" json:"id"`
	UserID     string     `db:"userId" json:"userId"`
	Title      string     `db:"title" json:"title"`
	DocumentID *string    `db:"documentId" json:"documentId,omitempty"`
	Pinned     bool       `db:"pinned" json:"pinned"`
	PinnedAt   *time.Time `db:"pinnedAt" json:"pinnedAt,omitempty"`
	CreatedAt  time.Time  `db:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time  `db:"updatedAt" json:"updatedAt"`
}
