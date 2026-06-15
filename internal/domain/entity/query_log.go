package entity

import "time"

type QueryLog struct {
	ID                string    `db:"id" json:"id"`
	ConversationID    *string   `db:"conversationId" json:"conversationId,omitempty"`
	UserID            string    `db:"userId" json:"userId"`
	Query             string    `db:"query" json:"query"`
	ReformulatedQuery *string   `db:"reformulatedQuery" json:"reformulatedQuery,omitempty"`
	SearchType        string    `db:"searchType" json:"searchType"`
	ChunksRetrieved   int       `db:"chunksRetrieved" json:"chunksRetrieved"`
	ResponseTimeMs    int       `db:"responseTimeMs" json:"responseTimeMs"`
	Metadata          []byte    `db:"metadata" json:"metadata,omitempty"`
	CreatedAt         time.Time `db:"createdAt" json:"createdAt"`
}
