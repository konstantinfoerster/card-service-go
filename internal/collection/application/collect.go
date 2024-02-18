package application

import (
	"errors"
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common"
)

type CollectService interface {
	Search(name string, page domain.Page, collector domain.Collector) (domain.PagedResult, error)
	Collect(item domain.Item, collector domain.Collector) (domain.Item, error)
}

type collectService struct {
	collectionRepo domain.CollectionRepository
	cardRepo       domain.SearchRepository
}

func NewCollectService(collectionRepo domain.CollectionRepository, cardRepo domain.SearchRepository) CollectService {
	return &collectService{
		collectionRepo: collectionRepo,
		cardRepo:       cardRepo,
	}
}

func (s *collectService) Search(name string, page domain.Page,
	collector domain.Collector) (domain.PagedResult, error) {
	r, err := s.collectionRepo.FindCollectedByName(name, page, collector)
	if err != nil {
		return domain.PagedResult{}, common.NewUnknownError(err, "unable-to-execute-search-in-collected")
	}

	return r, nil
}

func (s *collectService) Collect(item domain.Item, collector domain.Collector) (domain.Item, error) {
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
