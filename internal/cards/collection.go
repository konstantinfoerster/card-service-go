package cards

import (
	"context"
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

type CollectionService interface {
	Search(ctx context.Context, name string, c Collector, p Page) (Cards, error)
	Collect(ctx context.Context, item Collectable, c Collector) (Collectable, error)
}

type collectionService struct {
	cardRepo CardRepository
}

func NewCollectionService(cRepo CardRepository) CollectionService {
	return &collectionService{
		cardRepo: cRepo,
	}
}

func (s *collectionService) Search(ctx context.Context, name string, c Collector, page Page) (Cards, error) {
	filter := NewFilter().WithName(name).WithCollector(c).WithOnlyCollected()
	r, err := s.cardRepo.Find(ctx, filter, page)
	if err != nil {
		return Empty(page), aerrors.NewUnknownError(err, "unable-to-execute-search-in-collected")
	}

	return r, nil
}

func (s *collectionService) Collect(ctx context.Context, item Collectable, c Collector) (Collectable, error) {
	exist, err := s.cardRepo.Exist(ctx, item.ID)
	if err != nil {
		return Collectable{}, aerrors.NewUnknownError(err, "unable-to-find-item")
	}

	if !exist {
		msg := fmt.Sprintf("item with id %d not found", item.ID)

		return Collectable{}, aerrors.NewInvalidInputError(err, "unable-to-find-item", msg)
	}

	if item.Amount == 0 {
		if err := s.cardRepo.Remove(ctx, item, c); err != nil {
			return Collectable{}, aerrors.NewUnknownError(err, "unable-to-remove-item")
		}

		return item, nil
	}

	if err := s.cardRepo.Collect(ctx, item, c); err != nil {
		return Collectable{}, aerrors.NewUnknownError(err, "unable-to-collect-item")
	}

	return item, nil
}
