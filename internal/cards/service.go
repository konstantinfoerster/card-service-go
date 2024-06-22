package cards

import (
	"context"

	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
)

type Searcher interface {
	Search(ctx context.Context, name string, collector Collector, page common.Page) (Cards, error)
}

type searchService struct {
	repo Repository
}

func NewService(repo Repository) Searcher {
	return &searchService{
		repo: repo,
	}
}

func (s *searchService) Search(ctx context.Context, name string, collector Collector, page common.Page) (Cards, error) {
	if collector.ID == "" {
		r, err := s.repo.FindByName(ctx, name, page)
		if err != nil {
			return Empty(page), aerrors.NewUnknownError(err, "unable-to-execute-search")
		}

		return r, nil
	}

	r, err := s.repo.FindByNameWithAmount(ctx, name, collector, page)
	if err != nil {
		return Empty(page), aerrors.NewUnknownError(err, "unable-to-execute-search")
	}

	return r, nil
}
