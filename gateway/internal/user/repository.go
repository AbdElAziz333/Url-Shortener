package user

import (
	"context"
	"errors"
	"time"

	sqlcgen "aziz.dev/gateway/internal/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	FindByEmail(ctx context.Context, email string) (*sqlcgen.User, error)
	Create(ctx context.Context, user sqlcgen.User) error
}

type repository struct {
	queries *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		queries: sqlcgen.New(db),
	}
}

func (r *repository) FindByEmail(ctx context.Context, email string) (*sqlcgen.User, error) {
	u, err := r.queries.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return toDomainUser(u), nil
}

func (r *repository) Create(ctx context.Context, user sqlcgen.User) error {
	if user.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		
		user.ID = id
	}

	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	return r.queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		IsActive:     user.IsActive,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	})
}

func toDomainUser(u sqlcgen.User) *sqlcgen.User {
	return &sqlcgen.User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}
