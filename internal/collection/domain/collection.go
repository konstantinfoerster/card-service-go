package domain

import (
	"github.com/konstantinfoerster/card-service-go/internal/common"
)

// Collector user who interacts with his collection.
type Collector struct {
	ID string
}

// Item a collectable item.
type Item struct {
	ID     int
	Amount int
}

func NewItem(id int, amount int) (Item, error) {
	if id <= 0 {
		return Item{}, common.NewInvalidInputMsg("invalid-item-id", "invalid id")
	}

	if amount < 0 {
		return Item{}, common.NewInvalidInputMsg("invalid-item-amount", "amount cannot be negative")
	}

	return Item{
		ID:     id,
		Amount: amount,
	}, nil
}

type CollectionRepository interface {
	FindCollectedByName(name string, page common.Page, collector Collector) (Cards, error)
	Upsert(item Item, collector Collector) error
	Remove(itemID int, collector Collector) error
}
