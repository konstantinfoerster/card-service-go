package mocks

import (
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
)

type MockCardRepository struct {
	MockByID                   func(id int) (*domain.Card, error)
	MockFindByName             func(name string, page domain.Page) (domain.PagedResult, error)
	MockFindByNameAndCollector func(name string, page domain.Page, collector domain.Collector) (domain.PagedResult, error)
}

var _ domain.SearchRepository = (*MockCardRepository)(nil)

func (r *MockCardRepository) ByID(id int) (*domain.Card, error) {
	if r.MockByID == nil {
		return nil, fmt.Errorf("unexpected function call, no mock implementation provided")
	}

	return r.MockByID(id)
}
func (r *MockCardRepository) FindByName(name string, page domain.Page) (domain.PagedResult, error) {
	if r.MockFindByName == nil {
		return domain.PagedResult{}, fmt.Errorf("unexpected function call, no mock implementation provided")
	}

	return r.MockFindByName(name, page)
}

func (r *MockCardRepository) FindByNameAndCollector(name string, page domain.Page,
	collector domain.Collector) (domain.PagedResult, error) {
	if r.MockFindByNameAndCollector == nil {
		return domain.PagedResult{}, fmt.Errorf("unexpected function call, no mock implementation provided")
	}

	return r.MockFindByNameAndCollector(name, page, collector)
}
