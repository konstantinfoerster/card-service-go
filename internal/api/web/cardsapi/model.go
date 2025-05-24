package cardsapi

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

const (
	cardIDKey = "card"
	faceIDKey = "face"
)

var ErrInvalidInput = errors.New("invalid unput")

func newPage(c *fiber.Ctx) cards.Page {
	size, _ := strconv.Atoi(c.Query("size", ""))
	page, _ := strconv.Atoi(c.Query("page", "0"))

	return cards.NewPage(page, size)
}

func newPagedResponse[T any](pr cards.PagedResult[T]) *PagedResponse[any] {
	data := make([]any, len(pr.Result))
	for i, r := range pr.Result {
		switch v := any(r).(type) {
		case cards.Card:
			data[i] = newCard(v)
		case cards.Match:
			data[i] = newCard(v.Card).WithConfidence(v.Confidence)
		case cards.CardPrint:
			data[i] = CardPrint{
				ID:     asClientID(v.ID),
				Name:   v.Name,
				Number: v.Number,
				Code:   v.Code,
				Amount: Amount(v.Amount),
			}
		}
	}

	return &PagedResponse[any]{
		Data:     data,
		HasMore:  pr.HasMore,
		Page:     pr.Page,
		NextPage: pr.Page + 1,
	}
}

type PagedResponse[T any] struct {
	Data     []T  `json:"data"`
	HasMore  bool `json:"hasMore"`
	Page     int  `json:"page"`
	NextPage int  `json:"nextPage"`
}

func asClientID(id cards.ID) string {
	v := url.Values{}
	if id.CardID != 0 {
		v.Set("card", strconv.Itoa(id.CardID))
	}
	if id.FaceID != 0 {
		v.Set("face", strconv.Itoa(id.FaceID))
	}

	return base64.URLEncoding.EncodeToString([]byte(v.Encode()))
}

func toID(rawID string) (cards.ID, error) {
	if strings.TrimSpace(rawID) == "" {
		return cards.ID{}, fmt.Errorf("id must not be empty, %w", ErrInvalidInput)
	}
	dID, err := base64.URLEncoding.DecodeString(rawID)
	if err != nil {
		return cards.ID{}, fmt.Errorf("failed to decoded base64 id %s, %w", rawID, errors.Join(err, ErrInvalidInput))
	}

	v, err := url.ParseQuery(string(dID))
	if err != nil {
		return cards.ID{}, fmt.Errorf("failed to parse id %s %w", dID, errors.Join(err, ErrInvalidInput))
	}

	var cardID cards.ID
	if v.Has(cardIDKey) {
		id, err := strconv.Atoi(v.Get(cardIDKey))
		if err != nil {
			return cards.ID{}, fmt.Errorf("card-id is not a number %s %w", dID, errors.Join(err, ErrInvalidInput))
		}
		cardID.CardID = id
	}
	if v.Has(faceIDKey) {
		id, err := strconv.Atoi(v.Get(faceIDKey))
		if err != nil {
			return cards.ID{}, fmt.Errorf("face-id is not a number %s %w", dID, errors.Join(err, ErrInvalidInput))
		}
		cardID.FaceID = id
	}

	return cardID, nil
}

func newCard(c cards.Card) Card {
	return Card{
		ID:     asClientID(c.ID),
		Amount: Amount(c.Amount),
		Name:   c.Name,
		Number: c.Number,
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
	// Number the card number.
	Number string `json:"number"`
	// Image is the image URL.
	Image string `json:"image,omitempty"`
	// ID is the  base64 encoded card and face ID
	ID string `json:"id"`
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

type CardPrint struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Number string `json:"number"`
	Code   string `json:"code"`
	// Amount the number of the card in the users collection.
	Amount Amount `json:"amount,omitempty"`
}

func (p CardPrint) IsSame(id string) bool {
	return p.ID == id
}

type Item struct {
	// ID is the base64 encoded card and face ID.
	ID string `json:"id"`
	// Amount the number of the card in the users collection.
	Amount Amount `json:"amount,omitempty"`
}

func NewItem(id cards.ID, amount int) Item {
	return Item{
		ID:     asClientID(id),
		Amount: Amount(amount),
	}
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

func (a Amount) Print() string {
	return fmt.Sprintf("%02d", a.Value())
}

func asCollector(u web.User) cards.Collector {
	return cards.NewCollector(u.ID)
}
