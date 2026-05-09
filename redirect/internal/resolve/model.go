package resolve

import (
	"time"

	"github.com/google/uuid"
)

type Link struct {
	ID          uuid.UUID `gorm:"primaryKey;"`
	UserID      uuid.UUID `gorm:"index"`
	Code        string `gorm:"uniqueIndex"`
	OriginalURL string
	CustomAlias string
	ExpiresAt   *time.Time
	IsActive    bool
	CreatedAt   time.Time
}