package memory

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type InMemCardRepository struct {
	collected map[string][]cards.Collectable
	cards     []cards.Card
}

func NewCardRepository(data []cards.Card, collected map[string][]cards.Collectable) (*InMemCardRepository, error) {
	if collected == nil {
		collected = make(map[string][]cards.Collectable)
	}

	return &InMemCardRepository{
		cards:     data,
		collected: collected,
	}, nil
}

func NewCollectRepository(data []cards.Card) (*InMemCardRepository, error) {
	return &InMemCardRepository{
		cards:     data,
		collected: make(map[string][]cards.Collectable),
	}, nil
}

func (r *InMemCardRepository) Find(ctx context.Context, f cards.Filter, page cards.Page) (cards.Cards, error) {
	matches := make([]cards.Card, 0)
	for _, c := range r.cards {
		if f.Name != "" && !strings.Contains(strings.ToLower(c.Name), strings.ToLower(f.Name)) {
			continue
		}

		if f.IDs.NotEmpty() {
			if id := f.IDs.Find(c.ID); id == nil {
				continue
			}
		}

		if f.Collector != nil {
			isCollected := false
			for _, collected := range r.collected[f.Collector.ID] {
				if !collected.ID.Eq(c.ID) {
					continue
				}

				c.Amount = collected.Amount
				isCollected = true
			}

			if f.OnlyCollected && !isCollected {
				continue
			}
		}

		matches = append(matches, c)
	}

	slices.SortStableFunc(matches, func(a cards.Card, b cards.Card) int {
		return cmp.Compare(a.Name, b.Name)
	})

	matches = getPage(matches, page)

	return cards.NewCards(matches, page), nil
}

func (r *InMemCardRepository) Exist(_ context.Context, id cards.ID) (bool, error) {
	for _, c := range r.cards {
		if c.ID.Eq(id) {
			return true, nil
		}
	}

	return false, nil
}

func (r *InMemCardRepository) Collect(_ context.Context, item cards.Collectable, collector cards.Collector) error {
	cID := collector.ID

	if _, ok := r.collected[cID]; !ok {
		r.collected[cID] = make([]cards.Collectable, 0)
	}

	for i, c := range r.collected[cID] {
		if c.ID.Eq(item.ID) {
			r.collected[cID][i] = item

			return nil
		}
	}

	// not in collection yet
	for _, c := range r.cards {
		if c.ID.Eq(item.ID) {
			r.collected[cID] = append(r.collected[cID], item)

			return nil
		}
	}

	return nil
}

func (r *InMemCardRepository) Remove(_ context.Context, item cards.Collectable, collector cards.Collector) error {
	cID := collector.ID

	toDelete := -1
	for i, c := range r.collected[cID] {
		if c.ID.Eq(item.ID) {
			toDelete = i

			break
		}
	}

	if toDelete != -1 {
		r.collected[cID] = slices.Delete(r.collected[cID], toDelete, toDelete+1)
	}

	return nil
}

func (r *InMemCardRepository) Prints(
	ctx context.Context, name string, collector cards.Collector, page cards.Page) (cards.CardPrints, error) {
	matches := make([]cards.CardPrint, 0)
	for _, c := range r.cards {
		if !strings.EqualFold(name, c.Name) {
			continue
		}

		amount := 0
		if collector.ID != "" {
			for _, collected := range r.collected[collector.ID] {
				if collected.ID.Eq(c.ID) {
					amount = collected.Amount

					break
				}
			}
		}

		matches = append(matches, cards.CardPrint{
			ID:     c.ID,
			Name:   c.Name,
			Number: c.Number,
			Code:   c.Set.Code,
			Amount: amount,
		})
	}

	slices.SortStableFunc(matches, func(a cards.CardPrint, b cards.CardPrint) int {
		return cmp.Compare(a.Name, b.Name)
	})

	matches = getPage(matches, page)

	return cards.NewCardPrints(matches, page), nil
}

func getPage[T any](data []T, page cards.Page) []T {
	offset := page.Offset()
	if len(data) < offset {
		return []T{}
	}

	limit := page.Size()
	maxIdx := offset + limit
	if len(data) >= maxIdx {
		return data[offset:maxIdx]
	}

	return data[offset:]
}
