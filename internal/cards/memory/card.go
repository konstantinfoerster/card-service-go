package memory

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type inMemCardRepository struct {
	collected map[string][]cards.Collectable
	cards     []cards.Card
}

func NewCardRepository(data []cards.Card, collected map[string][]cards.Collectable) (cards.CardRepository, error) {
	if collected == nil {
		collected = make(map[string][]cards.Collectable)
	}

	return &inMemCardRepository{
		cards:     data,
		collected: collected,
	}, nil
}

func NewCollectRepository(data []cards.Card) (cards.CardRepository, error) {
	return &inMemCardRepository{
		cards:     data,
		collected: make(map[string][]cards.Collectable),
	}, nil
}

func (r inMemCardRepository) Find(ctx context.Context, f cards.Filter, page cards.Page) (cards.Cards, error) {
	// FIXME: allow empty name
	if f.Name == "" {
		return cards.Empty(page), nil
	}

	matches := make([]cards.Card, 0)
	for _, c := range r.cards {
		if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(f.Name)) {
			continue
		}

		if f.Collector != nil {
			isCollected := false
			for _, collected := range r.collected[f.Collector.ID] {
				if collected.ID != c.ID {
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

	matches = cards.GetPage(matches, page)

	return cards.NewCards(matches, page), nil
}

func (r inMemCardRepository) Exist(_ context.Context, id int) (bool, error) {
	for _, c := range r.cards {
		if c.ID == id {
			return true, nil
		}
	}

	return false, nil
}

func (r inMemCardRepository) Collect(_ context.Context, item cards.Collectable, collector cards.Collector) error {
	cID := collector.ID

	if _, ok := r.collected[cID]; !ok {
		r.collected[cID] = make([]cards.Collectable, 0)
	}

	for i, c := range r.collected[cID] {
		if c.ID == item.ID {
			r.collected[cID][i] = item

			return nil
		}
	}

	// not in collection yet
	for _, c := range r.cards {
		if c.ID == item.ID {
			r.collected[cID] = append(r.collected[cID], item)

			return nil
		}
	}

	return nil
}

func (r inMemCardRepository) Remove(_ context.Context, item cards.Collectable, collector cards.Collector) error {
	cID := collector.ID

	toDelete := -1
	for i, c := range r.collected[cID] {
		if c.ID == item.ID {
			toDelete = i

			break
		}
	}

	if toDelete != -1 {
		r.collected[cID] = slices.Delete(r.collected[cID], toDelete, toDelete+1)
	}

	return nil
}
