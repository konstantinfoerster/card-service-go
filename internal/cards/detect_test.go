package cards_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/stretchr/testify/assert"
)

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
