package fakes

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/img"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
)

type FakeRepository struct {
	hasher    img.Hasher
	collected map[string][]domain.Card
	cards     []domain.Card
}

func NewRepository(hasher img.Hasher) (*FakeRepository, error) {
	return newRepository(hasher)
}

func newRepository(hasher img.Hasher) (*FakeRepository, error) {
	dir := currentDir()

	cards := make([]domain.Card, 0)

	cards10E, err := readFromJSON(filepath.Join(dir, "testdata/cards10E.json"))
	if err != nil {
		return nil, err
	}
	cards = append(cards, cards10E...)

	cards2ED, err := readFromJSON(filepath.Join(dir, "testdata/cards2ED.json"))
	if err != nil {
		return nil, err
	}
	cards = append(cards, cards2ED...)

	cards2X2, err := readFromJSON(filepath.Join(dir, "testdata/cards2X2.json"))
	if err != nil {
		return nil, err
	}
	cards = append(cards, cards2X2...)

	return &FakeRepository{
		cards:     cards,
		hasher:    hasher,
		collected: make(map[string][]domain.Card),
	}, nil
}

func currentDir() string {
	_, cf, _, _ := runtime.Caller(0)

	return path.Join(path.Dir(cf))
}

func (r FakeRepository) ByID(id int) (domain.Card, error) {
	for _, c := range r.cards {
		if c.ID == id {
			return c, nil
		}
	}

	return domain.Card{}, fmt.Errorf("card with id %v not found %w", id, domain.ErrCardNotFound)
}

func (r FakeRepository) FindByName(name string, page common.Page) (domain.Cards, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Empty(page), nil
	}

	var result []domain.Card
	for _, c := range r.cards {
		if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(name)) {
			continue
		}

		result = append(result, c)
	}

	slices.SortStableFunc(result, func(a domain.Card, b domain.Card) int {
		return cmp.Compare(a.Name, b.Name)
	})

	result = truncateResult(result, page)

	return domain.NewCards(result, page), nil
}

func (r FakeRepository) FindByCollectorAndName(
	collector domain.Collector, name string, page common.Page) (domain.Cards, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Empty(page), nil
	}

	cards := make(map[int]domain.Card, 0)
	for _, c := range r.cards {
		if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(name)) {
			continue
		}

		cards[c.ID] = c
	}

	for _, c := range r.collected[collector.ID] {
		if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(name)) {
			continue
		}

		// replace with collected card
		cards[c.ID] = c
	}

	result := values(cards)

	slices.SortStableFunc(result, func(a domain.Card, b domain.Card) int {
		return cmp.Compare(a.Name, b.Name)
	})

	result = truncateResult(result, page)

	return domain.NewCards(result, page), nil
}

func (r FakeRepository) hash(c domain.Card) (domain.Hash, error) {
	fImg, err := os.Open(filepath.Join(currentDir(), "testdata", c.Image))
	if err != nil {
		return domain.Hash{}, err
	}
	defer commonio.Close(fImg)

	dImg, _, err := image.Decode(fImg)
	if err != nil {
		return domain.Hash{}, err
	}

	return r.hasher.Hash(img.Image{Image: dImg})
}

func (r FakeRepository) Top5MatchesByHash(ctx context.Context, hashes ...domain.Hash) (domain.Matches, error) {
	var result domain.Matches
	for _, card := range r.cards {
		if card.Image == "" {
			continue
		}

		cHash, err := r.hash(card)
		if err != nil {
			return nil, err
		}

		lowest := 1000
		for _, h := range hashes {
			d, err := r.hasher.Distance(cHash, h)
			if err != nil {
				return nil, err
			}
			if d < lowest {
				lowest = d
			}
		}
		result = append(result, domain.Match{
			Card:  card,
			Score: lowest,
		})
	}

	return result, nil
}

func (r FakeRepository) Top5MatchesByCollectorAndHash(
	ctx context.Context, collector domain.Collector, hashes ...domain.Hash) (domain.Matches, error) {
	return nil, nil
}

func (r FakeRepository) FindCollectedByName(
	name string, page common.Page, collector domain.Collector) (domain.Cards, error) {
	var result []domain.Card
	for _, c := range r.collected[collector.ID] {
		if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(name)) {
			continue
		}

		result = append(result, c)
	}

	slices.SortStableFunc(result, func(a domain.Card, b domain.Card) int {
		return cmp.Compare(a.Name, b.Name)
	})

	result = truncateResult(result, page)

	return domain.NewCards(result, page), nil
}

func (r FakeRepository) Upsert(item domain.Item, collector domain.Collector) error {
	if _, ok := r.collected[collector.ID]; !ok {
		r.collected[collector.ID] = make([]domain.Card, 0)
	}

	for i, c := range r.collected[collector.ID] {
		if c.ID == item.ID {
			c.Amount = item.Amount
			r.collected[collector.ID][i] = c

			return nil
		}
	}

	// not in collected yet
	for _, c := range r.cards {
		if c.ID == item.ID {
			c.Amount = item.Amount
			r.collected[collector.ID] = append(r.collected[collector.ID], c)

			return nil
		}
	}

	return nil
}

func (r FakeRepository) Remove(itemID int, collector domain.Collector) error {
	toDelete := -1
	for i, c := range r.collected[collector.ID] {
		if c.ID == itemID {
			toDelete = i

			break
		}
	}

	if toDelete != -1 {
		r.collected[collector.ID] = slices.Delete(r.collected[collector.ID], toDelete, toDelete+1)
	}

	return nil
}

func readFromJSON(path string) ([]domain.Card, error) {
	// #nosec G304 only used in tests
	cardsRaw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s, %w", path, err)
	}

	var cards []domain.Card
	if err := json.Unmarshal(cardsRaw, &cards); err != nil {
		return nil, err
	}

	return cards, nil
}

func values(m map[int]domain.Card) []domain.Card {
	result := make([]domain.Card, 0, len(m))

	for _, v := range m {
		result = append(result, v)
	}

	return result
}

func truncateResult(data []domain.Card, page common.Page) []domain.Card {
	offset := page.Offset()
	if len(data) < offset {
		return []domain.Card{}
	}

	limit := page.Size()
	maxIdx := offset + limit
	if len(data) >= maxIdx {
		return data[offset:maxIdx]
	}

	return data[offset:]
}
