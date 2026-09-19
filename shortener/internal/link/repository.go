package link

import (
	"context"

	sqlcgen "aziz.dev/shortener/internal/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	FindAllByUserID(ctx context.Context, userID uuid.UUID, pagination Pagination) ([]sqlcgen.Link, error)
	FindByCodeAndUserID(ctx context.Context, code string, userID uuid.UUID) (sqlcgen.Link, error)
	Create(ctx context.Context, arg sqlcgen.CreateLinkParams) (sqlcgen.Link, error)
	Update(ctx context.Context, arg sqlcgen.UpdateLinkParams) (int64, error)
}

type repository struct {
	queries *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		queries: sqlcgen.New(db),
	}
}

type Pagination struct {
	Page int // e.g., 1 for the first page
	PageSize int // e.g., 10 items per page
}

func (r *repository) FindAllByUserID(ctx context.Context, userID uuid.UUID, pagination Pagination) ([]sqlcgen.Link, error) {
	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	// Calculate offset: (page - 1) * pageSize
    // Page 1: (1 - 1) * 10 = 0 (skip 0 items)
    // Page 2: (2 - 1) * 10 = 10 (skip first 10 items)
	offset := (pagination.Page - 1) * pagination.PageSize

	return r.queries.FindAllByUserID(ctx, sqlcgen.FindAllByUserIDParams{
		UserID: userID,
		Limit: int32(pagination.PageSize),
		Offset: int32(offset),
	})
}

func (r *repository) FindByCodeAndUserID(ctx context.Context, code string, userID uuid.UUID) (sqlcgen.Link, error) {
	return r.queries.FindByCodeAndUserID(ctx, sqlcgen.FindByCodeAndUserIDParams{
		Code: code,
		UserID: userID,
	})
}

func (r *repository) Create(ctx context.Context, arg sqlcgen.CreateLinkParams) (sqlcgen.Link, error) {
	return r.queries.CreateLink(ctx, arg)
}

func (r *repository) Update(ctx context.Context, arg sqlcgen.UpdateLinkParams) (int64, error) {
	return r.queries.UpdateLink(ctx, arg)
}