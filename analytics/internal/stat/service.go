package stat

import "context"

type Service interface {
	GetTotalClicks(ctx context.Context) ([]Dto, error)
	GetGeo(ctx context.Context) ([]Dto, error)
	GetReferrers(ctx context.Context) ([]Dto, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetTotalClicks(ctx context.Context) ([]Dto, error) {
	return nil, nil
}

func (s *service) GetGeo(ctx context.Context) ([]Dto, error) {
	return nil, nil
}

func (s *service) GetReferrers(ctx context.Context) ([]Dto, error) {
	return nil, nil
}