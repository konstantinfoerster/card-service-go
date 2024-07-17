package memory

import (
	"cmp"
	"context"
	_ "image/jpeg"
	"slices"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/common"
)

type inMemCardRepository struct {
	collected map[string][]cards.Item
	cards     []cards.Card
}

func NewCardRepository(data []cards.Card, collected map[string][]cards.Item) (cards.CardRepository, error) {
	return &inMemCardRepository{
		cards:     data,
		collected: collected,
	}, nil
}

func (r inMemCardRepository) FindByName(ctx context.Context, name string, page common.Page) (cards.Cards, error) {
	return r.FindByNameWithAmount(ctx, name, cards.Collector{}, page)
}

func (r inMemCardRepository) FindByNameWithAmount(
	_ context.Context, name string, collector cards.Collector, page common.Page) (cards.Cards, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return cards.Empty(page), nil
	}

	matches := make([]cards.Card, 0)
	for _, c := range r.cards {
		if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(name)) {
			continue
		}

		if collector.ID != "" {
			for _, collected := range r.collected[collector.ID] {
				if collected.ID == c.ID {
					c.Amount = collected.Amount
				}
			}
		}

		matches = append(matches, c)
	}

	slices.SortStableFunc(matches, func(a cards.Card, b cards.Card) int {
		return cmp.Compare(a.Name, b.Name)
	})

	matches = common.GetPage(matches, page)

	return cards.NewCards(matches, page), nil
}

func values(m map[int]cards.Card) []cards.Card {
	result := make([]cards.Card, 0, len(m))

	for _, v := range m {
		result = append(result, v)
	}

	return result
}
