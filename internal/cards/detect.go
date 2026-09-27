package cards

import (
	"cmp"
	"context"
	"errors"
	"io"
	"slices"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
)

var ErrInvalidInput = errors.New("invalid input")

type Score struct {
	ID    ID
	Score int
}

type Scores []Score

func (s Scores) Find(oID ID) *Score {
	for _, score := range s {
		if score.ID.Eq(oID) {
			return &score
		}
	}

	return nil
}

type Match struct {
	Card
	Confidence int
}

type Matches struct {
	PagedResult[Match]
}

func NewMatches(cards Cards, scores Scores, page Page) Matches {
	matches := make([]Match, 0)
	for _, c := range cards.Result {
		if s := scores.Find(c.ID); s != nil {
			matches = append(matches, Match{Card: c, Confidence: s.Score})
		}
	}

	// sort by lowest-distance, lowest = best match
	slices.SortStableFunc(matches, func(a, b Match) int {
		return cmp.Compare(a.Confidence, b.Confidence)
	})

	return Matches{
		NewPagedResult(matches, page),
	}
}

func EmptyMatches(p Page) Matches {
	return Matches{
		NewEmptyResult[Match](p),
	}
}

type Matcher interface {
	Top5Matches(ctx context.Context, in io.Reader) (Scores, error)
}

type DetectService struct {
	cRepo   CardRepository
	matcher Matcher
}

func NewDetectService(cRepo CardRepository, matcher Matcher) *DetectService {
	return &DetectService{
		cRepo:   cRepo,
		matcher: matcher,
	}
}

func (s *DetectService) Detect(ctx context.Context, c Collector, in io.Reader) (Matches, error) {
	if in == nil {
		return EmptyMatches(DefaultPage()), nil
	}

	scores, err := s.matcher.Top5Matches(ctx, in)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return Matches{}, aerrors.NewInvalidInputError(err, "invalid-detect-input", "invalid detection input")
		}

		return Matches{}, aerrors.NewUnknownError(err, "unable-to-execute-matching")
	}

	if len(scores) == 0 {
		return EmptyMatches(DefaultPage()), nil
	}

	ids := make([]ID, 0, len(scores))
	for _, s := range scores {
		ids = append(ids, s.ID)
	}
	filter := NewFilter().WithCollector(c).WithID(ids...)

	limit := 5
	page := NewPage(1, limit)
	cards, err := s.cRepo.Find(ctx, filter, page)
	if err != nil {
		return Matches{}, aerrors.NewUnknownError(err, "unable-to-execute-card-search")
	}

	matches := NewMatches(cards, scores, page)
	matches.HasMore = false

	return matches, nil
}
