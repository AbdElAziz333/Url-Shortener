package link

import "time"

type Link struct {
	ID uint64 `gorm:"primaryKey;"`
	Code string
	OriginalURL string
	CustomAlias string
	ExpiresAt time.Time
	IsActive bool
	CreatedAt time.Time
}

type OutboxEvent struct {
	ID uint64
	EventType string
	Payload map[string]any
	Status string
	CreatedAt time.Time
	PublishedAt time.Time
}