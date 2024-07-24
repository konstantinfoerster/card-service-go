package cardsapi

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

func newPage(c *fiber.Ctx) cards.Page {
	size, _ := strconv.Atoi(c.Query("size", ""))
	page, _ := strconv.Atoi(c.Query("page", "0"))

	return cards.NewPage(page, size)
}

type PagedResponse[T any] struct {
	Data     []T  `json:"data"`
	HasMore  bool `json:"hasMore"`
	Page     int  `json:"page"`
	NextPage int  `json:"nextPage"`
}

func newPagedResponse(pr cards.Cards) *PagedResponse[Card] {
	data := make([]Card, len(pr.Result))
	for i, c := range pr.Result {
		data[i] = Card{
			Item:  newItem(c.ID, c.Amount),
			Name:  c.Name,
			Image: c.Image,
		}
	}

	return &PagedResponse[Card]{
		Data:     data,
		HasMore:  pr.HasMore,
		Page:     pr.Page,
		NextPage: pr.Page + 1,
	}
}

type Card struct {
	Score *int   `json:"score,omitempty"`
	Name  string `json:"name"`
	Image string `json:"image,omitempty"`
	Item
}

func (c Card) WithScore(v int) Card {
	c.Score = &v

	return c
}

func newItem(id int, amount int) Item {
	return Item{
		ID:     id,
		Amount: amount,
	}
}

type Item struct {
	ID     int `json:"id"`
	Amount int `json:"amount,omitempty"`
}

func (i Item) NextAmount() int {
	return i.Amount + 1
}

func (i Item) PreviousAmount() int {
	prev := i.Amount - 1
	if prev < 0 {
		return 0
	}

	return prev
}

func asCollector(u *web.User) cards.Collector {
	if u == nil {
		return cards.NewCollector("")
	}

	return cards.NewCollector(u.ID)
}
