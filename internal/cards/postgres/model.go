package postgres

import (
	"database/sql"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type dbCard struct {
	Name   string
	Image  sql.NullString
	CardID int
	Amount int
}

func toCard(dbCard *dbCard) cards.Card {
	return cards.Card{
		ID:     dbCard.CardID,
		Name:   dbCard.Name,
		Image:  dbCard.Image.String,
		Amount: dbCard.Amount,
	}
}
