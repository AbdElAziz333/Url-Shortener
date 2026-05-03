package link

import "context"

type Service interface {
	GetAll(ctx context.Context) ([]Dto, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetAll(ctx context.Context) ([]Dto, error) {
	return nil, nil
}