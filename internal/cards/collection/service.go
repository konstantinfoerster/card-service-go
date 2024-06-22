package collection

import (
	"context"
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
	"github.com/pkg/errors"
)

type Service interface {
	cards.Searcher
	Collect(ctx context.Context, item Item) (Item, error)
}

type collectionService struct {
	collectionRepo Repository
	cardRepo       cards.Repository
}

func NewService(collectionRepo Repository, cardRepo cards.Repository) Service {
	return &collectionService{
		collectionRepo: collectionRepo,
		cardRepo:       cardRepo,
	}
}

func (s *collectionService) Search(ctx context.Context, name string, collector cards.Collector, page common.Page) (cards.Cards, error) {
	r, err := s.collectionRepo.FindCollectedByName(ctx, name, collector, page)
	if err != nil {
		return cards.Empty(page), aerrors.NewUnknownError(err, "unable-to-execute-search-in-collected")
	}

	return r, nil
}

func (s *collectionService) Collect(ctx context.Context, item Item) (Item, error) {
	_, err := s.collectionRepo.ByID(ctx, item.ID)
	if err != nil {
		if errors.Is(err, cards.ErrCardNotFound) {
			msg := fmt.Sprintf("item with id %d not found", item.ID)

			return Item{}, aerrors.NewInvalidInputError(err, "unable-to-find-item", msg)
		}

		return Item{}, aerrors.NewUnknownError(err, "unable-to-find-item")
	}

	if item.Amount == 0 {
		if err := s.collectionRepo.Remove(ctx, item); err != nil {
			return Item{}, aerrors.NewUnknownError(err, "unable-to-remove-item")
		}

		return item, nil
	}

	if err := s.collectionRepo.Upsert(ctx, item); err != nil {
		return Item{}, aerrors.NewUnknownError(err, "unable-to-upsert-item")
	}

	return item, nil
}
