package cards

import (
	"context"
	"errors"
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
)

// Collectable a collectable item.
type Collectable struct {
	ID     int
	Amount int
}

func NewCollectable(id int, amount int) (Collectable, error) {
	if id <= 0 {
		return Collectable{}, aerrors.NewInvalidInputMsg("invalid-id", "invalid id")
	}

	if amount < 0 {
		return Collectable{}, aerrors.NewInvalidInputMsg("invalid-amount", "amount cannot be negative")
	}

	return Collectable{
		ID:     id,
		Amount: amount,
	}, nil
}

func RemoveItem(id int) (Collectable, error) {
	return NewCollectable(id, 0)
}

type CollectionRepository interface {
	ByID(ctx context.Context, id int) (Card, error)
	FindCollectedByName(ctx context.Context, name string, c Collector, p Page) (Cards, error)
	Upsert(ctx context.Context, item Collectable, c Collector) error
	Remove(ctx context.Context, item Collectable, c Collector) error
}

type CollectionService interface {
	Search(ctx context.Context, name string, c Collector, p Page) (Cards, error)
	Collect(ctx context.Context, item Collectable, c Collector) (Collectable, error)
}

type collectionService struct {
	collectionRepo CollectionRepository
}

func NewCollectionService(collectionRepo CollectionRepository) CollectionService {
	return &collectionService{
		collectionRepo: collectionRepo,
	}
}

func (s *collectionService) Search(ctx context.Context, name string, c Collector, page Page) (Cards, error) {
	r, err := s.collectionRepo.FindCollectedByName(ctx, name, c, page)
	if err != nil {
		return Empty(page), aerrors.NewUnknownError(err, "unable-to-execute-search-in-collected")
	}

	return r, nil
}

func (s *collectionService) Collect(ctx context.Context, item Collectable, c Collector) (Collectable, error) {
	_, err := s.collectionRepo.ByID(ctx, item.ID)
	if err != nil {
		if errors.Is(err, ErrCardNotFound) {
			msg := fmt.Sprintf("item with id %d not found", item.ID)

			return Collectable{}, aerrors.NewInvalidInputError(err, "unable-to-find-item", msg)
		}

		return Collectable{}, aerrors.NewUnknownError(err, "unable-to-find-item")
	}

	if item.Amount == 0 {
		if err := s.collectionRepo.Remove(ctx, item, c); err != nil {
			return Collectable{}, aerrors.NewUnknownError(err, "unable-to-remove-item")
		}

		return item, nil
	}

	if err := s.collectionRepo.Upsert(ctx, item, c); err != nil {
		return Collectable{}, aerrors.NewUnknownError(err, "unable-to-upsert-item")
	}

	return item, nil
}
