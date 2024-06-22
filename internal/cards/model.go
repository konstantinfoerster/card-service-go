package cards

import "github.com/konstantinfoerster/card-service-go/internal/common"

const DefaultLang = "eng"

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

func NewCollector(id string) Collector {
	return Collector{ID: id}
}

// Collector user who interacts with his collection.
type Collector struct {
	ID string
}
