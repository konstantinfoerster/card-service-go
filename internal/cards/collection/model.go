package collection

import "github.com/konstantinfoerster/card-service-go/internal/common/aerrors"

// Item a collectable item.
type Item struct {
	ID     int
	Amount int
}

func NewItem(id int, amount int) (Item, error) {
	if id <= 0 {
		return Item{}, aerrors.NewInvalidInputMsg("invalid-item-id", "invalid id")
	}

	if amount < 0 {
		return Item{}, aerrors.NewInvalidInputMsg("invalid-item-amount", "amount cannot be negative")
	}

	return Item{
		ID:     id,
		Amount: amount,
	}, nil
}

func RemoveItem(id int) (Item, error) {
	return NewItem(id, 0)
}
