package detection

import (
	"context"
	"fmt"
	"io"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type Finder interface {
	// Top5MatchesByHash tries to find the cards closest to the given hash.
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
// Matching also considers images that are upside down.
func (m *Matcher) Top5Matches(ctx context.Context, in io.Reader) (cards.Scores, error) {
	hashes, err := m.hasher.Hashes(in, None, Degree180)
	if err != nil {
		return cards.Scores{}, fmt.Errorf("failed to hash image %w", err)
	}

	return m.finder.Top5MatchesByHash(ctx, hashes...)
}
