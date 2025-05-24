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
	hasher image.Hasher
	cards  []cards.Card
	cfg    config.Images
}

func NewDetectRepository(data []cards.Card, cfg config.Images, hasher image.Hasher) (cards.DetectRepository, error) {
	return &InMemDetectRepository{
		cards:  data,
		hasher: hasher,
		cfg:    cfg,
	}, nil
}

func (r InMemDetectRepository) Top5MatchesByHash(ctx context.Context, hashes ...image.Hash) (cards.Scores, error) {
	var result cards.Scores
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

		// only scores < 60 are allowed
		minConfidence := 60
		if lowest >= minConfidence {
			continue
		}

		result = append(result, cards.Score{
			ID:    card.ID,
			Score: lowest,
		})
	}

	slices.SortStableFunc(result, func(a cards.Score, b cards.Score) int {
		return cmp.Compare(a.Score, b.Score)
	})

	limit := 5
	if len(result) > limit {
		return result[:limit], nil
	}

	return result, nil
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
