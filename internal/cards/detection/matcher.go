package detection

import (
	"context"
	"fmt"
	"io"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

// MaxScore is the score, the summed Hamming distance of all hashes, from which on an image
// no longer counts as a match.
// It only filters out noise. Telling similar cards apart is up to MinLead.
const MaxScore = 270

// MinLead is the minimum score gap between the best and the second best card.
// A smaller gap means the image, e.g. only part of a card, fits several cards.
const MinLead = 25

type Finder interface {
	// Top5MatchesByHash returns the cards closest to the given hashes,
	// one per card name, best first.
	Top5MatchesByHash(ctx context.Context, hashes ...Hash) (cards.Scores, error)
}

type Matcher struct {
	finder Finder
	hasher Hasher
}

func NewMatcher(f Finder, h Hasher) *Matcher {
	return &Matcher{
		hasher: h,
		finder: f,
	}
}

// Top5Matches tries to find the cards closest to the produced hashes based on the given input.
// Matching also considers images that are upside down. It returns no matches
// if the two best cards are less than MinLead apart.
func (m *Matcher) Top5Matches(ctx context.Context, in io.Reader) (cards.Scores, error) {
	hashes, err := m.hasher.Hashes(in, None, Degree180)
	if err != nil {
		return cards.Scores{}, fmt.Errorf("failed to hash image %w", err)
	}

	scores, err := m.finder.Top5MatchesByHash(ctx, hashes...)
	if err != nil {
		return cards.Scores{}, err
	}

	if len(scores) > 1 && scores[1].Score-scores[0].Score < MinLead {
		return cards.Scores{}, nil
	}

	return scores, nil
}
