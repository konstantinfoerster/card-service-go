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

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/collection"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detect"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
)

type FakeRepository struct {
	hasher    detect.Hasher
	collected map[string][]cards.Card
	cards     []cards.Card
}

var _ cards.Repository = FakeRepository{}
var _ collection.Repository = FakeRepository{}
var _ detect.Repository = FakeRepository{}

func NewRepository(hasher detect.Hasher) (*FakeRepository, error) {
	return newRepository(hasher)
}

func newRepository(hasher detect.Hasher) (*FakeRepository, error) {
	dir := currentDir()

	cc := make([]cards.Card, 0)

	cards10E, err := readFromJSON(filepath.Join(dir, "testdata/cards10E.json"))
	if err != nil {
		return nil, err
	}
	cc = append(cc, cards10E...)

	cards2ED, err := readFromJSON(filepath.Join(dir, "testdata/cards2ED.json"))
	if err != nil {
		return nil, err
	}
	cc = append(cc, cards2ED...)

	cards2X2, err := readFromJSON(filepath.Join(dir, "testdata/cards2X2.json"))
	if err != nil {
		return nil, err
	}
	cc = append(cc, cards2X2...)

	return &FakeRepository{
		cards:     cc,
		hasher:    hasher,
		collected: make(map[string][]cards.Card),
	}, nil
}

func currentDir() string {
	_, cf, _, _ := runtime.Caller(0)

	return path.Join(path.Dir(cf))
}

func (r FakeRepository) ByID(_ context.Context, id int) (cards.Card, error) {
	for _, c := range r.cards {
		if c.ID == id {
			return c, nil
		}
	}

	return cards.Card{}, fmt.Errorf("card with id %v not found %w", id, cards.ErrCardNotFound)
}

func (r FakeRepository) FindByName(_ context.Context, name string, page common.Page) (cards.Cards, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return cards.Empty(page), nil
	}

	var result []cards.Card
	for _, c := range r.cards {
		if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(name)) {
			continue
		}

		result = append(result, c)
	}

	slices.SortStableFunc(result, func(a cards.Card, b cards.Card) int {
		return cmp.Compare(a.Name, b.Name)
	})

	result = truncateResult(result, page)

	return cards.NewCards(result, page), nil
}

func (r FakeRepository) FindByNameWithAmount(
	_ context.Context, name string, collector cards.Collector, page common.Page) (cards.Cards, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return cards.Empty(page), nil
	}

	cc := make(map[int]cards.Card, 0)
	for _, c := range r.cards {
		if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(name)) {
			continue
		}

		cc[c.ID] = c
	}

	for _, c := range r.collected[collector.ID] {
		if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(name)) {
			continue
		}

		// replace with collected card
		cc[c.ID] = c
	}

	result := values(cc)

	slices.SortStableFunc(result, func(a cards.Card, b cards.Card) int {
		return cmp.Compare(a.Name, b.Name)
	})

	result = truncateResult(result, page)

	return cards.NewCards(result, page), nil
}

func (r FakeRepository) hash(c cards.Card) (detect.Hash, error) {
	fImg, err := os.Open(filepath.Join(currentDir(), "testdata", c.Image))
	if err != nil {
		return detect.Hash{}, err
	}
	defer commonio.Close(fImg)

	dImg, _, err := image.Decode(fImg)
	if err != nil {
		return detect.Hash{}, err
	}

	return r.hasher.Hash(detect.Image{Image: dImg})
}

func (r FakeRepository) allMatchesByHash(_ context.Context, hashes ...detect.Hash) (detect.Matches, error) {
	var result detect.Matches
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
		result = append(result, detect.Match{
			Card:  card,
			Score: lowest,
		})
	}

	slices.SortStableFunc(result, func(a detect.Match, b detect.Match) int {
		return cmp.Compare(a.Score, b.Score)
	})

	return result, nil
}

func (r FakeRepository) Top5MatchesByHash(ctx context.Context, hashes ...detect.Hash) (detect.Matches, error) {
	result, err := r.allMatchesByHash(ctx, hashes...)
	if err != nil {
		return nil, err
	}

	limit := 5
	if len(result) > limit {
		return result[:limit], nil
	}

	return result, nil
}

func (r FakeRepository) Top5MatchesByCollectorAndHash(
	ctx context.Context, collector cards.Collector, hashes ...detect.Hash) (detect.Matches, error) {
	result, err := r.allMatchesByHash(ctx, hashes...)
	if err != nil {
		return nil, err
	}

	for _, c := range r.collected[collector.ID] {
		for i := range result {
			result[i].Amount = c.Amount
		}
	}

	limit := 5
	if len(result) > limit {
		return result[:limit], nil
	}

	return result, nil
}

func (r FakeRepository) FindCollectedByName(
	_ context.Context, name string, collector cards.Collector, page common.Page) (cards.Cards, error) {
	var result []cards.Card
	for _, c := range r.collected[collector.ID] {
		if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(name)) {
			continue
		}

		result = append(result, c)
	}

	slices.SortStableFunc(result, func(a cards.Card, b cards.Card) int {
		return cmp.Compare(a.Name, b.Name)
	})

	result = truncateResult(result, page)

	return cards.NewCards(result, page), nil
}

func (r FakeRepository) Upsert(_ context.Context, item collection.Item, collector cards.Collector) error {
	cID := collector.ID

	if _, ok := r.collected[cID]; !ok {
		r.collected[cID] = make([]cards.Card, 0)
	}

	for i, c := range r.collected[cID] {
		if c.ID == item.ID {
			c.Amount = item.Amount
			r.collected[cID][i] = c

			return nil
		}
	}

	// not in collected yet
	for _, c := range r.cards {
		if c.ID == item.ID {
			c.Amount = item.Amount
			r.collected[cID] = append(r.collected[cID], c)

			return nil
		}
	}

	return nil
}

func (r FakeRepository) Remove(_ context.Context, item collection.Item, collector cards.Collector) error {
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

func readFromJSON(path string) ([]cards.Card, error) {
	// #nosec G304 only used in tests
	cardsRaw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s, %w", path, err)
	}

	var cards []cards.Card
	if err := json.Unmarshal(cardsRaw, &cards); err != nil {
		return nil, err
	}

	return cards, nil
}

func values(m map[int]cards.Card) []cards.Card {
	result := make([]cards.Card, 0, len(m))

	for _, v := range m {
		result = append(result, v)
	}

	return result
}

func truncateResult(data []cards.Card, page common.Page) []cards.Card {
	offset := page.Offset()
	if len(data) < offset {
		return []cards.Card{}
	}

	limit := page.Size()
	maxIdx := offset + limit
	if len(data) >= maxIdx {
		return data[offset:maxIdx]
	}

	return data[offset:]
}
