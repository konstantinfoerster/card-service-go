package postgres

import (
	"database/sql"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type dbCard struct {
	Name     string
	SetName  string
	SetCode  string
	ImageURL sql.NullString
	CardID   int
	Amount   int
}

func toCard(dbCard dbCard) cards.Card {
	return cards.Card{
		ID:   dbCard.CardID,
		Name: dbCard.Name,
		Image: cards.Image{
			URL: dbCard.ImageURL.String,
		},
		Amount: dbCard.Amount,
		Set: cards.Set{
			Name: dbCard.SetName,
			Code: dbCard.SetCode,
		},
	}
}
