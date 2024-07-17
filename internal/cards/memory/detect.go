package memory

import (
	"cmp"
	"context"
	"image"
	_ "image/jpeg"
	"os"
	"path"
	"slices"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/common/detect"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

type InMemDetectRepository struct {
	hasher    detect.Hasher
	collected map[string][]cards.Item
	cards     []cards.Card
	cfg       config.Images
}

func NewDetectRepository(
	data []cards.Card, collected map[string][]cards.Item, cfg config.Images, hasher detect.Hasher,
) (cards.DetectRepository, error) {
	return &InMemDetectRepository{
		cards:     data,
		hasher:    hasher,
		collected: collected,
		cfg:       cfg,
	}, nil
}

func (r InMemDetectRepository) hash(c cards.Card) (detect.Hash, error) {
	fImg, err := os.Open(path.Join(r.cfg.Host, c.Image))
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

func (r InMemDetectRepository) Top5MatchesByHash(ctx context.Context, hashes ...detect.Hash) (cards.Matches, error) {
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

func (r InMemDetectRepository) Top5MatchesByCollectorAndHash(
	ctx context.Context, collector cards.Collector, hashes ...detect.Hash) (cards.Matches, error) {
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

func (r InMemDetectRepository) allMatchesByHash(_ context.Context, hashes ...detect.Hash) (cards.Matches, error) {
	var result cards.Matches
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
