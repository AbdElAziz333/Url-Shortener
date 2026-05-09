package stat

import "context"

type Service interface {
	GetTotalClicks(ctx context.Context, code string) ([]Dto, error)
	GetGeo(ctx context.Context, code string) ([]Dto, error)
	GetReferrers(ctx context.Context, code string) ([]Dto, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetTotalClicks(ctx context.Context, code string) ([]Dto, error) {
	return s.repo.GetTotalClicks(ctx, code)
}

func (s *service) GetGeo(ctx context.Context, code string) ([]Dto, error) {
	return s.repo.GetGeo(ctx, code)
}

func (s *service) GetReferrers(ctx context.Context, code string) ([]Dto, error) {
	return s.repo.GetReferrers(ctx, code)
}
