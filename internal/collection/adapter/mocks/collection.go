package mocks

import (
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
)

type MockCollectionRepository struct {
	MockFindCollectedByName func(name string, page domain.Page, collector domain.Collector) (domain.PagedResult, error)
	MockUpsert              func(item domain.Item, collector domain.Collector) error
	MockRemove              func(itemID int, collector domain.Collector) error
}

var _ domain.CollectionRepository = (*MockCollectionRepository)(nil)

func (r *MockCollectionRepository) FindCollectedByName(name string, page domain.Page,
	collector domain.Collector) (domain.PagedResult, error) {
	if r.MockFindCollectedByName == nil {
		return domain.PagedResult{}, fmt.Errorf("unexpected function call, no mock implementation provided")
	}

	return r.MockFindCollectedByName(name, page, collector)
}

func (r *MockCollectionRepository) Upsert(item domain.Item, collector domain.Collector) error {
	if r.MockUpsert == nil {
		return fmt.Errorf("unexpected function call, no mock implementation provided")
	}

	return r.MockUpsert(item, collector)
}

func (r *MockCollectionRepository) Remove(itemID int, collector domain.Collector) error {
	if r.MockRemove == nil {
		return fmt.Errorf("unexpected function call, no mock implementation provided")
	}

	return r.MockRemove(itemID, collector)
}
