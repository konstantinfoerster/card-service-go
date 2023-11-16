package mocks

import (
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
)

type MockCollectionRepository struct {
	MockFindByName func(name string, page domain.Page, collector domain.Collector) (domain.PagedResult, error)
	MockAdd        func(item domain.Item, collector domain.Collector) error
	MockRemove     func(itemID int, collector domain.Collector) error
	MockCount      func(itemID int, collector domain.Collector) (int, error)
}

var _ domain.CollectionRepository = (*MockCollectionRepository)(nil)

func (r *MockCollectionRepository) FindByName(name string, page domain.Page,
	collector domain.Collector) (domain.PagedResult, error) {
	if r.MockFindByName == nil {
		return domain.PagedResult{}, fmt.Errorf("unexpected function call, no mock implementation provided")
	}

	return r.MockFindByName(name, page, collector)
}

func (r *MockCollectionRepository) Add(item domain.Item, collector domain.Collector) error {
	if r.MockAdd == nil {
		return fmt.Errorf("unexpected function call, no mock implementation provided")
	}

	return r.MockAdd(item, collector)
}

func (r *MockCollectionRepository) Remove(itemID int, collector domain.Collector) error {
	if r.MockRemove == nil {
		return fmt.Errorf("unexpected function call, no mock implementation provided")
	}

	return r.MockRemove(itemID, collector)
}

func (r *MockCollectionRepository) Count(itemID int, collector domain.Collector) (int, error) {
	if r.MockCount == nil {
		return 0, fmt.Errorf("unexpected function call, no mock implementation provided")
	}

	return r.MockCount(itemID, collector)
}
