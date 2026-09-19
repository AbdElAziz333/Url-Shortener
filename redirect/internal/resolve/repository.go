package resolve

import (
	"context"
	"errors"
	"fmt"

	sqlcgen "aziz.dev/redirect/internal/postgres/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("link not found")

type Repository interface {
	Find(ctx context.Context, code string) (*sqlcgen.Link, error)
}
 
type repository struct {
	queries *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		queries: sqlcgen.New(db),
	}
}

func (r *repository) Find(ctx context.Context, code string) (*sqlcgen.Link, error) {
	link, err := r.queries.FindByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, code)
		}
		
		return nil, err
	}
 
	return &link, nil
}