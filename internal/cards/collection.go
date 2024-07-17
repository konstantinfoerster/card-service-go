package cards

import (
	"context"
	"errors"
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
)

// Item a collectable item.
// FIXME: rename to Collectable
type Item struct {
	ID     int
	Amount int
}

func NewItem(id int, amount int) (Item, error) {
	if id <= 0 {
		return Item{}, aerrors.NewInvalidInputMsg("invalid-item-id", "invalid id")
	}

	if amount < 0 {
		return Item{}, aerrors.NewInvalidInputMsg("invalid-item-amount", "amount cannot be negative")
	}

	return Item{
		ID:     id,
		Amount: amount,
	}, nil
}

func RemoveItem(id int) (Item, error) {
	return NewItem(id, 0)
}

type CollectionRepository interface {
	ByID(ctx context.Context, id int) (Card, error)
	FindCollectedByName(ctx context.Context, name string, collector Collector, page common.Page) (Cards, error)
	Upsert(ctx context.Context, item Item, collector Collector) error
	Remove(ctx context.Context, item Item, collector Collector) error
}

type CollectionService interface {
	Search(ctx context.Context, name string, collector Collector, page common.Page) (Cards, error)
	Collect(ctx context.Context, item Item, collector Collector) (Item, error)
}

type collectionService struct {
	collectionRepo CollectionRepository
}

func NewCollectionService(collectionRepo CollectionRepository) CollectionService {
	return &collectionService{
		collectionRepo: collectionRepo,
	}
}

func (s *collectionService) Search(ctx context.Context, name string, collector Collector, page common.Page) (Cards, error) {
	r, err := s.collectionRepo.FindCollectedByName(ctx, name, collector, page)
	if err != nil {
		return Empty(page), aerrors.NewUnknownError(err, "unable-to-execute-search-in-collected")
	}

	return r, nil
}

func (s *collectionService) Collect(ctx context.Context, item Item, collector Collector) (Item, error) {
	_, err := s.collectionRepo.ByID(ctx, item.ID)
	if err != nil {
		if errors.Is(err, ErrCardNotFound) {
			msg := fmt.Sprintf("item with id %d not found", item.ID)

			return Item{}, aerrors.NewInvalidInputError(err, "unable-to-find-item", msg)
		}

		return Item{}, aerrors.NewUnknownError(err, "unable-to-find-item")
	}

	if item.Amount == 0 {
		if err := s.collectionRepo.Remove(ctx, item, collector); err != nil {
			return Item{}, aerrors.NewUnknownError(err, "unable-to-remove-item")
		}

		return item, nil
	}

	if err := s.collectionRepo.Upsert(ctx, item, collector); err != nil {
		return Item{}, aerrors.NewUnknownError(err, "unable-to-upsert-item")
	}

	return item, nil
}
