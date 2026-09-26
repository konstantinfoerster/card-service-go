package cards_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubMatcher struct {
	scores cards.Scores
	err    error
}

func (s stubMatcher) Top5Matches(_ context.Context, _ io.Reader) (cards.Scores, error) {
	return s.scores, s.err
}

type stubCardRepository struct {
	cards.CardRepository

	cards  []cards.Card
	err    error
	called bool
	filter cards.Filter
	page   cards.Page
}

func (s *stubCardRepository) Find(_ context.Context, filter cards.Filter, page cards.Page) (cards.Cards, error) {
	s.called = true
	s.filter = filter
	s.page = page
	if s.err != nil {
		return cards.Cards{}, s.err
	}

	return cards.NewCards(s.cards, page), nil
}

func TestDetectErrors(t *testing.T) {
	cases := []struct {
		name           string
		matcherErr     error
		repoErr        error
		expectedErrTyp aerrors.ErrorType
	}{
		{
			name:           "invalid input",
			matcherErr:     errors.Join(assert.AnError, cards.ErrInvalidInput),
			expectedErrTyp: aerrors.ErrInvalidInput,
		},
		{
			name:           "matcher failure",
			matcherErr:     assert.AnError,
			expectedErrTyp: aerrors.ErrUnknown,
		},
		{
			name:           "card search failure",
			repoErr:        assert.AnError,
			expectedErrTyp: aerrors.ErrUnknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			matcher := stubMatcher{scores: cards.Scores{{ID: cards.NewID(1), Score: 1}}, err: tc.matcherErr}
			repo := &stubCardRepository{err: tc.repoErr}
			svc := cards.NewDetectService(repo, matcher)

			_, err := svc.Detect(t.Context(), cards.Collector{}, strings.NewReader(""))

			var appErr aerrors.AppError
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, tc.expectedErrTyp, appErr.ErrorType)
			assert.ErrorIs(t, err, assert.AnError)
		})
	}
}

func TestDetectNoScoresSkipsCardSearch(t *testing.T) {
	repo := &stubCardRepository{}
	svc := cards.NewDetectService(repo, stubMatcher{})

	matches, err := svc.Detect(t.Context(), cards.Collector{}, strings.NewReader(""))

	require.NoError(t, err)
	assert.Empty(t, matches.Result)
	assert.False(t, repo.called)
}

func TestDetectSearchesScoredCards(t *testing.T) {
	matcher := stubMatcher{scores: cards.Scores{
		{ID: cards.NewID(1), Score: 30},
		{ID: cards.NewID(2), Score: 10},
	}}
	repo := &stubCardRepository{}
	svc := cards.NewDetectService(repo, matcher)
	collector := cards.NewCollector("myUser")

	_, err := svc.Detect(t.Context(), collector, strings.NewReader(""))

	require.NoError(t, err)
	expected := cards.NewFilter().WithCollector(collector).WithID(cards.NewID(1), cards.NewID(2))
	assert.Equal(t, expected, repo.filter)
	assert.Equal(t, cards.NewPage(1, 5), repo.page)
}

func TestDetect(t *testing.T) {
	card1 := cards.Card{ID: cards.NewID(1), Name: "Card 1"}
	card2 := cards.Card{ID: cards.NewID(2), Name: "Card 2"}
	cardWithoutScore := cards.Card{ID: cards.NewID(3), Name: "Card 3"}
	matcher := stubMatcher{scores: cards.Scores{
		{ID: card1.ID, Score: 30},
		{ID: card2.ID, Score: 10},
	}}
	repo := &stubCardRepository{cards: []cards.Card{card1, card2, cardWithoutScore}}
	svc := cards.NewDetectService(repo, matcher)

	matches, err := svc.Detect(t.Context(), cards.Collector{}, strings.NewReader(""))

	require.NoError(t, err)
	expected := []cards.Match{
		{Card: card2, Confidence: 10},
		{Card: card1, Confidence: 30},
	}
	assert.Equal(t, expected, matches.Result)
	assert.False(t, matches.HasMore)
}

func TestDetectAllowsEmptyReader(t *testing.T) {
	svc := cards.NewDetectService(nil, nil)

	matches, err := svc.Detect(t.Context(), cards.Collector{}, nil)

	require.NoError(t, err)
	assert.Empty(t, matches.Result)
}

func TestNewMatchesSortsByConfidenceAscending(t *testing.T) {
	cardList := []cards.Card{
		{ID: cards.NewID(1), Name: "Card A"},
		{ID: cards.NewID(2), Name: "Card B"},
		{ID: cards.NewID(3), Name: "Card C"},
	}
	scores := cards.Scores{
		{ID: cards.NewID(1), Score: 80},
		{ID: cards.NewID(2), Score: 20},
		{ID: cards.NewID(3), Score: 50},
	}

	matches := cards.NewMatches(cards.NewCards(cardList, cards.DefaultPage()), scores, cards.DefaultPage())

	assert.Len(t, matches.Result, 3)
	assert.Equal(t, cards.NewID(2), matches.Result[0].ID)
	assert.Equal(t, 20, matches.Result[0].Confidence)
	assert.Equal(t, cards.NewID(3), matches.Result[1].ID)
	assert.Equal(t, 50, matches.Result[1].Confidence)
	assert.Equal(t, cards.NewID(1), matches.Result[2].ID)
	assert.Equal(t, 80, matches.Result[2].Confidence)
}
