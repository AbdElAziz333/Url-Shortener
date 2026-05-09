package link

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

func (l *Link) BeforeCreate(tx *gorm.DB) (err error) {
	l.ID, err = uuid.NewV7()
	if err != nil {
		return err
	}

	l.CreatedAt = time.Now()
	return nil
}
