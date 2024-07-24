package memory

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type inMemCollectionRepository struct {
	collected map[string][]cards.Collectable
	cards     []cards.Card
}

func NewCollectionRepository(data []cards.Card) (cards.CollectionRepository, error) {
	return &inMemCollectionRepository{
		cards:     data,
		collected: make(map[string][]cards.Collectable),
	}, nil
}

func (r inMemCollectionRepository) ByID(_ context.Context, id int) (cards.Card, error) {
	for _, c := range r.cards {
		if c.ID == id {
			return c, nil
		}
	}

	return cards.Card{}, fmt.Errorf("card with id %v not found %w", id, cards.ErrCardNotFound)
}

func (r inMemCollectionRepository) FindCollectedByName(
	_ context.Context, name string, collector cards.Collector, page cards.Page) (cards.Cards, error) {
	var matches []cards.Card
	for _, c := range r.cards {
		if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(name)) {
			continue
		}

		for _, collected := range r.collected[collector.ID] {
			if collected.ID != c.ID {
				continue
			}

			c.Amount = collected.Amount

			matches = append(matches, c)
		}
	}

	slices.SortStableFunc(matches, func(a cards.Card, b cards.Card) int {
		return cmp.Compare(a.Name, b.Name)
	})

	matches = cards.GetPage(matches, page)

	return cards.NewCards(matches, page), nil
}

func (r inMemCollectionRepository) Upsert(_ context.Context, item cards.Collectable, collector cards.Collector) error {
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

func (r inMemCollectionRepository) Remove(_ context.Context, item cards.Collectable, collector cards.Collector) error {
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
