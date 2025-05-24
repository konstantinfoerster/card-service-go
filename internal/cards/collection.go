package cards

import (
	"context"
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
)

// Collectable a collectable item.
type Collectable struct {
	ID     ID
	Amount int
}

func NewCollectable(id ID, amount int) (Collectable, error) {
	if id.CardID <= 0 {
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

type CollectionRepository interface {
	// Find returns the cards for the requested page matching the given criteria.
	Find(ctx context.Context, filter Filter, page Page) (Cards, error)
	// Exist returns true if a card with the given ID exist, false otherwise.
	Exist(ctx context.Context, id ID) (bool, error)
	// Collect adds or removes an card from a collection.
	Collect(ctx context.Context, item Collectable, c Collector) error
	// Remove removes the item from the collection.
	Remove(ctx context.Context, item Collectable, c Collector) error
}

type CollectionService interface {
	Search(ctx context.Context, name string, c Collector, p Page) (Cards, error)
	Collect(ctx context.Context, item Collectable, c Collector) (Collectable, error)
}

type collectionService struct {
	repo CollectionRepository
}

func NewCollectionService(cRepo CollectionRepository) CollectionService {
	return &collectionService{
		repo: cRepo,
	}
}

func (s *collectionService) Search(ctx context.Context, name string, c Collector, page Page) (Cards, error) {
	filter := NewFilter().
		WithName(name).
		WithCollector(c).
		WithOnlyCollected().
		WithLanguage(DefaultLang)
	r, err := s.repo.Find(ctx, filter, page)
	if err != nil {
		return EmptyCards(page), aerrors.NewUnknownError(err, "unable-to-execute-search-in-collected")
	}

	return r, nil
}

func (s *collectionService) Collect(ctx context.Context, item Collectable, c Collector) (Collectable, error) {
	exist, err := s.repo.Exist(ctx, item.ID)
	if err != nil {
		return Collectable{}, aerrors.NewUnknownError(err, "unable-to-find-item")
	}

	if !exist {
		msg := fmt.Sprintf("item with id %v not found", item.ID)

		return Collectable{}, aerrors.NewInvalidInputError(err, "unable-to-find-item", msg)
	}

	if item.Amount == 0 {
		if err := s.repo.Remove(ctx, item, c); err != nil {
			return Collectable{}, aerrors.NewUnknownError(err, "unable-to-remove-item")
		}

		return item, nil
	}

	if err := s.repo.Collect(ctx, item, c); err != nil {
		return Collectable{}, aerrors.NewUnknownError(err, "unable-to-collect-item")
	}

	return item, nil
}
