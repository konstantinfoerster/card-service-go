package memory

import (
	"cmp"
	"context"
	"os"
	"path"
	"slices"

	"github.com/konstantinfoerster/card-service-go/internal/aio"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/imaging"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
)

type InMemDetectRepository struct {
	cards []cards.Card
	cfg   postgres.Images
}

func NewDetectRepository(data []cards.Card, cfg postgres.Images) (*InMemDetectRepository, error) {
	return &InMemDetectRepository{
		cards: data,
		cfg:   cfg,
	}, nil
}

func (r *InMemDetectRepository) Top5MatchesByHash(ctx context.Context, hashes ...cards.Hash) (cards.Scores, error) {
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
			d, err := imaging.Distance(cHash, h)
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

func (r *InMemDetectRepository) hash(c cards.Card) (cards.Hash, error) {
	fImg, err := os.Open(path.Join(r.cfg.Host, c.Image.URL))
	if err != nil {
		return cards.Hash{}, err
	}
	defer aio.Close(fImg)

	img, err := imaging.NewImage(fImg)
	if err != nil {
		return cards.Hash{}, err
	}

	return img.Hash()
}
