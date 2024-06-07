package domain

import (
	"context"
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/common"
)

type Card struct {
	Name   string
	Image  string
	ID     int
	Amount int
}

type Cards struct {
	common.PagedResult[Card]
}

func Empty(p common.Page) Cards {
	return Cards{
		common.NewEmptyResult[Card](p),
	}
}

func NewCards(cards []Card, p common.Page) Cards {
	return Cards{
		common.NewPagedResult(cards, p),
	}
}

type Match struct {
	Card
	Score int
}

type Matches []Match

type Hash struct {
	Value []uint64
	Bits  int
}

func (h Hash) AsBase2() []string {
	base2 := make([]string, 0, len(h.Value))
	for _, v := range h.Value {
		base2 = append(base2, fmt.Sprintf("%064b", v))
	}

	return base2
}

var ErrCardNotFound = fmt.Errorf("card not found")

type SearchRepository interface {
	ByID(id int) (Card, error)
	FindByName(name string, page common.Page) (Cards, error)
	FindByCollectorAndName(collector Collector, name string, page common.Page) (Cards, error)
	Top5MatchesByHash(ctx context.Context, hashes ...Hash) (Matches, error)
	Top5MatchesByCollectorAndHash(ctx context.Context, collector Collector, hashes ...Hash) (Matches, error)
}
