package link

import "time"

type Dto struct {
	Code        string     `json:"code"`
	OriginalURL string     `json:"original_url"`
	CustomAlias *string    `json:"custom_alias,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateRequest struct {
	OriginalURL string     `json:"original_url" binding:"required,url"`
	CustomAlias string     `json:"custom_alias,omitempty" binding:"omitempty,alphanum"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type UpdateExpiryDto struct {
	ExpiresAt *time.Time `json:"expires_at" binding:"required"`
}

type UpdateAliasDto struct {
	CustomAlias string `json:"custom_alias" binding:"required"`
}

func mapToDto(l *Link) Dto {
	return Dto{
		Code:        l.Code,
		OriginalURL: l.OriginalURL,
		CustomAlias: l.CustomAlias,
		ExpiresAt:   l.ExpiresAt,
		IsActive:    l.IsActive,
		CreatedAt:   l.CreatedAt,
	}
}