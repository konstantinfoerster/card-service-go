package application

import (
	"errors"
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common"
)

type CollectService interface {
	Search(name string, page domain.Page, collector domain.Collector) (domain.PagedResult, error)
	Collect(item domain.Item, collector domain.Collector) (domain.CollectableResult, error)
	Remove(item domain.Item, collector domain.Collector) (domain.CollectableResult, error)
}

type collectService struct {
	collectionRepo domain.CollectionRepository
	cardRepo       domain.CardRepository
}

func NewCollectService(collectionRepo domain.CollectionRepository, cardRepo domain.CardRepository) CollectService {
	return &collectService{
		collectionRepo: collectionRepo,
		cardRepo:       cardRepo,
	}
}

func (s *collectService) Search(name string, page domain.Page,
	collector domain.Collector) (domain.PagedResult, error) {
	r, err := s.collectionRepo.FindByName(name, page, collector)
	if err != nil {
		return domain.PagedResult{}, common.NewUnknownError(err, "unable-to-execute-search-in-collected")
	}

	return r, nil
}

func (s *collectService) Collect(item domain.Item, collector domain.Collector) (domain.CollectableResult, error) {
	_, err := s.cardRepo.ByID(item.ID)
	if err != nil {
		if errors.Is(err, domain.ErrCardNotFound) {
			msg := fmt.Sprintf("item with id %d not found", item.ID)

			return domain.CollectableResult{}, common.NewInvalidInputError(err, "unable-to-find-item", msg)
		}

		return domain.CollectableResult{}, common.NewUnknownError(err, "unable-to-find-item")
	}

	if err := s.collectionRepo.Add(item, collector); err != nil {
		return domain.CollectableResult{}, common.NewUnknownError(err, "unable-to-add-item")
	}

	amount, err := s.collectionRepo.Count(item.ID, collector)
	if err != nil {
		return domain.CollectableResult{}, common.NewUnknownError(err, "unable-to-count-items")
	}

	return domain.NewCollectableResult(item, amount), nil
}

func (s *collectService) Remove(item domain.Item, collector domain.Collector) (domain.CollectableResult, error) {
	if err := s.collectionRepo.Remove(item.ID, collector); err != nil {
		return domain.CollectableResult{}, common.NewUnknownError(err, "unable-to-remove-item")
	}

	amount, err := s.collectionRepo.Count(item.ID, collector)
	if err != nil {
		return domain.CollectableResult{}, common.NewUnknownError(err, "unable-to-count-items")
	}

	return domain.NewCollectableResult(item, amount), nil
}
