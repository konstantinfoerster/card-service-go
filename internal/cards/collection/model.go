package collection

import "github.com/konstantinfoerster/card-service-go/internal/common/aerrors"

// Item a collectable item.
type Item struct {
	Owner  string
	ID     int
	Amount int
}

func NewItem(id int, amount int, owner string) (Item, error) {
	if id <= 0 {
		return Item{}, aerrors.NewInvalidInputMsg("invalid-item-id", "invalid id")
	}

	if amount < 0 {
		return Item{}, aerrors.NewInvalidInputMsg("invalid-item-amount", "amount cannot be negative")
	}

	if owner == "" {
		return Item{}, aerrors.NewInvalidInputMsg("invalid-item-owner", "invalid owner")
	}
	return Item{
		ID:     id,
		Amount: amount,
		Owner:  owner,
	}, nil
}

func RemoveItem(id int, owner string) (Item, error) {
    return NewItem(id, 0, owner)
}
