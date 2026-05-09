package stat

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LinkStatsDaily struct {
	ID         uuid.UUID `gorm:"primaryKey;"`
	LinkCode   string    `gorm:"index"`
	Date       time.Time `gorm:"index"`
	ClickCount int       `gorm:"index"`
	UniqueIPs  uuid.UUID `gorm:"index"`
}

func (LinkStatsDaily) TableName() string {
	return "link_stats_daily"
}

func (l *LinkStatsDaily) BeforeCreate(tx *gorm.DB) (err error) {
	l.ID, err = uuid.NewV7()
	if err != nil {
		return err
	}

	return nil
}
