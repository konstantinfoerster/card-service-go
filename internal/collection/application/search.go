package application

import (
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common"
)

type SearchService interface {
	Search(name string, page domain.Page, collector domain.Collector) (domain.PagedResult, error)
}

type searchService struct {
	repo domain.SearchRepository
}

func NewSearchService(repo domain.SearchRepository) SearchService {
	return &searchService{
		repo: repo,
	}
}

func (s *searchService) Search(name string, page domain.Page, collector domain.Collector) (domain.PagedResult, error) {
	if collector.ID == "" {
		r, err := s.repo.FindByName(name, page)
		if err != nil {
			return domain.PagedResult{}, common.NewUnknownError(err, "unable-to-execute-search")
		}

		return r, nil
	}

	r, err := s.repo.FindByNameAndCollector(name, page, collector)
	if err != nil {
		return domain.PagedResult{}, common.NewUnknownError(err, "unable-to-execute-search")
	}

	return r, nil
}
