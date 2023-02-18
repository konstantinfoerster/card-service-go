package application

import (
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/search/domain"
)

type Service interface {
	SimpleSearch(name string, page domain.Page) (domain.PagedResult, error)
}

type service struct {
	repo domain.Repository
}

func New(repo domain.Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) SimpleSearch(name string, page domain.Page) (domain.PagedResult, error) {
	r, err := s.repo.FindByName(name, page)
	if err != nil {
		return domain.PagedResult{}, common.NewUnknownError(err, "unable-to-execute-simple-search")
	}

	return r, nil
}
