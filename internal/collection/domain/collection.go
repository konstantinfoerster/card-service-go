package domain

import (
	"github.com/konstantinfoerster/card-service-go/internal/common"
)

type Collector struct {
	ID string
}

type Item struct {
	ID int
}

func NewItem(id int) (Item, error) {
	if id <= 0 {
		return Item{}, common.NewInvalidInputMsg("invalid-item-id", "invalid id")
	}

	return Item{
		ID: id,
	}, nil
}

type CollectableResult struct {
	Item
	Amount int
}

func NewCollectableResult(item Item, amount int) CollectableResult {
	if amount < 0 {
		amount = 0
	}

	return CollectableResult{
		Item:   item,
		Amount: amount,
	}
}

type CollectionRepository interface {
	FindByName(name string, page Page, collector Collector) (PagedResult, error)
	Add(item Item, collector Collector) error
	Remove(itemID int, collector Collector) error
	Count(itemID int, collector Collector) (int, error)
}
