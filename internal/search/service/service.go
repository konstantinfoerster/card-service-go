package service

import (
	"github.com/konstantinfoerster/card-service/internal/common/errors"
	"github.com/konstantinfoerster/card-service/internal/search/domain"
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
		return domain.PagedResult{}, errors.NewError(err, "unable-to-execute-simple-search")
	}
	return r, nil
}
