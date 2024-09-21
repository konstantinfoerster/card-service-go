package memory

import (
	"cmp"
	"context"
	"os"
	"path"
	"slices"

	"github.com/konstantinfoerster/card-service-go/internal/aio"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/image"
)

type InMemDetectRepository struct {
	hasher    image.Hasher
	collected map[string][]cards.Collectable
	cards     []cards.Card
	cfg       config.Images
}

func NewDetectRepository(
	data []cards.Card, collected map[string][]cards.Collectable, cfg config.Images, hasher image.Hasher,
) (cards.DetectRepository, error) {
	return &InMemDetectRepository{
		cards:     data,
		hasher:    hasher,
		collected: collected,
		cfg:       cfg,
	}, nil
}

func (r InMemDetectRepository) hash(c cards.Card) (image.Hash, error) {
	fImg, err := os.Open(path.Join(r.cfg.Host, c.Image.URL))
	if err != nil {
		return image.Hash{}, err
	}
	defer aio.Close(fImg)

	img, err := image.NewImage(fImg)
	if err != nil {
		return image.Hash{}, err
	}

	return r.hasher.Hash(img)
}

func (r InMemDetectRepository) Top5MatchesByHash(
	ctx context.Context, c cards.Collector, hashes ...image.Hash) (cards.Matches, error) {
	result, err := r.allMatchesByHash(ctx, hashes...)
	if err != nil {
		return nil, err
	}

	if c.ID != "" {
		for _, c := range r.collected[c.ID] {
			for i := range result {
				result[i].Amount = c.Amount
			}
		}
	}

	limit := 5
	if len(result) > limit {
		return result[:limit], nil
	}

	return result, nil
}

func (r InMemDetectRepository) allMatchesByHash(_ context.Context, hashes ...image.Hash) (cards.Matches, error) {
	var result cards.Matches
	for _, card := range r.cards {
		if card.Image.URL == "" {
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
		result = append(result, cards.Match{
			Card:  card,
			Score: lowest,
		})
	}

	slices.SortStableFunc(result, func(a cards.Match, b cards.Match) int {
		return cmp.Compare(a.Score, b.Score)
	})

	return result, nil
}
