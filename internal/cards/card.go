package cards

import (
	"context"
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
)

const DefaultLang = "eng"

var ErrCardNotFound = fmt.Errorf("card not found")

type Card struct {
	Name   string
	Image  string
	ID     int
	Amount int
}

type Cards struct {
	PagedResult[Card]
}

func Empty(p Page) Cards {
	return Cards{
		NewEmptyResult[Card](p),
	}
}

func NewCards(cards []Card, p Page) Cards {
	return Cards{
		NewPagedResult(cards, p),
	}
}

func NewCollector(id string) Collector {
	return Collector{ID: id}
}

// Collector user who interacts with his collection.
type Collector struct {
	ID string
}

type CardRepository interface {
	FindByName(ctx context.Context, name string, page Page) (Cards, error)
	FindByNameWithAmount(ctx context.Context, name string, collector Collector, page Page) (Cards, error)
}

type CardService interface {
	Search(ctx context.Context, name string, collector Collector, page Page) (Cards, error)
}

type searchService struct {
	repo CardRepository
}

func NewCardService(repo CardRepository) CardService {
	return &searchService{
		repo: repo,
	}
}

func (s *searchService) Search(ctx context.Context, name string, collector Collector, page Page) (Cards, error) {
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
