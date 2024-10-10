package cardsapi

import (
	"fmt"
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

func newResponse[T any](pr cards.PagedResult[T]) *PagedResponse {
	data := make([]Card, len(pr.Result))
	for i, r := range pr.Result {
		switch v := any(r).(type) {
		case cards.Card:
			data[i] = newCard(v)
		case cards.Match:
			data[i] = newCard(v.Card).WithConfidence(v.Confidence)
		}
	}

	return &PagedResponse{
		Data:     data,
		HasMore:  pr.HasMore,
		Page:     pr.Page,
		NextPage: pr.Page + 1,
	}
}

type PagedResponse struct {
	Data     []Card `json:"data"`
	HasMore  bool   `json:"hasMore"`
	Page     int    `json:"page"`
	NextPage int    `json:"nextPage"`
}

func newCard(c cards.Card) Card {
	return Card{
		ID:     c.ID.CardID,
		Amount: Amount(c.Amount),
		Name:   c.Name,
		Set: Set{
			Code: c.Set.Code,
			Name: c.Set.Name,
		},
		Image: c.Image.URL,
	}
}

type Card struct {
	// Confidence indicates the confidence level of the match, lower is better.
	Confidence *int `json:"score,omitempty"`
	// Set is the set the card belongs to.
	Set Set `json:"set"`
	// Name the name of the card face.
	Name string `json:"name"`
	// Image is the image URL.
	Image string `json:"image,omitempty"`
	// ID is the card ID.
	ID int `json:"id"`
	// Amount indicates how many copies the current user owns.
	Amount Amount `json:"amount,omitempty"`
}

func (c Card) WithConfidence(v int) Card {
	c.Confidence = &v

	return c
}

func (c Card) Title() string {
	return fmt.Sprintf("%s - %s (%s)", c.Name, c.Set.Name, c.Set.Code)
}

type Set struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func NewItem(id cards.ID, amount int) Item {
	return Item{
		ID:     id.CardID,
		Amount: Amount(amount),
	}
}

type Item struct {
	// ID is the card ID.
	ID int `json:"id"`
	// Amount the total amount of the given card.
	Amount Amount `json:"amount,omitempty"`
}

type Amount int

func (a Amount) Next() Amount {
	return a + 1
}

func (a Amount) Previous() Amount {
	prev := a - 1
	if prev < 0 {
		return 0
	}

	return prev
}

func (a Amount) Value() int {
	return int(a)
}

func asCollector(u web.User) cards.Collector {
	return cards.NewCollector(u.ID)
}
