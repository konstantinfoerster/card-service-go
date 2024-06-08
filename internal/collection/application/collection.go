package application

import (
	"errors"
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common"
)

type CollectionService interface {
	Search(name string, page common.Page, collector domain.Collector) (domain.Cards, error)
	Collect(item domain.Item, collector domain.Collector) (domain.Item, error)
}

type collectionService struct {
	collectionRepo domain.CollectionRepository
	cardRepo       domain.SearchRepository
}

func NewCollectionService(collectionRepo domain.CollectionRepository,
	cardRepo domain.SearchRepository) CollectionService {
	return &collectionService{
		collectionRepo: collectionRepo,
		cardRepo:       cardRepo,
	}
}

func (s *collectionService) Search(name string, page common.Page, collector domain.Collector) (domain.Cards, error) {
	r, err := s.collectionRepo.FindCollectedByName(name, page, collector)
	if err != nil {
		return domain.Empty(page), common.NewUnknownError(err, "unable-to-execute-search-in-collected")
	}

	return r, nil
}

func (s *collectionService) Collect(item domain.Item, collector domain.Collector) (domain.Item, error) {
	_, err := s.cardRepo.ByID(item.ID)
	if err != nil {
		if errors.Is(err, domain.ErrCardNotFound) {
			msg := fmt.Sprintf("item with id %d not found", item.ID)

			return domain.Item{}, common.NewInvalidInputError(err, "unable-to-find-item", msg)
		}

		return domain.Item{}, common.NewUnknownError(err, "unable-to-find-item")
	}

	if item.Amount == 0 {
		if err := s.collectionRepo.Remove(item.ID, collector); err != nil {
			return domain.Item{}, common.NewUnknownError(err, "unable-to-remove-item")
		}

		return item, nil
	}

	if err := s.collectionRepo.Upsert(item, collector); err != nil {
		return domain.Item{}, common.NewUnknownError(err, "unable-to-upsert-item")
	}

	return item, nil
}
