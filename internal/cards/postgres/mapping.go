package postgres

import (
	"database/sql"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type dbCard struct {
	Row      sql.NullInt64
	Name     string
	SetName  string
	SetCode  string
	ImageURL sql.NullString
	CardID   int
	FaceID   int
	Amount   int
}

func toCard(dbCard dbCard) cards.Card {
	return cards.Card{
		ID:   cards.ID{CardID: dbCard.CardID, FaceID: dbCard.FaceID},
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
