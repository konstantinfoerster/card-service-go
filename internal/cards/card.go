package cards

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
)

const DefaultLang = "eng"

var ErrCardNotFound = errors.New("card not found")
var ErrResultMismatch = errors.New("result mismatch")
var ErrInvalidID = errors.New("invalid id")

type ID struct {
	// CardID the card ID
	CardID int
	// FaceID the face ID
	FaceID int
}

func NewID(id int) ID {
	return ID{CardID: id, FaceID: 0}
}

func (id ID) WithFace(faceID int) ID {
	return ID{CardID: id.CardID, FaceID: faceID}
}

func (id ID) String() string {
	return fmt.Sprintf("{ID - card %d, face %d}", id.CardID, id.FaceID)
}

func (id ID) Eq(o ID) bool {
	if id.FaceID != 0 && o.FaceID != 0 {
		return id.FaceID == o.FaceID
	}
	if id.CardID != 0 && o.CardID != 0 {
		return id.CardID == o.CardID
	}

	return false
}

type IDs []ID

func (ids IDs) NotEmpty() bool {
	return len(ids) > 0
}

func (ids IDs) Find(oID ID) *ID {
	for _, id := range ids {
		if id.Eq(oID) {
			return &id
		}
	}

	return nil
}

type Card struct {
	// Set the set that the card belongs to
	Set Set
	// Name is the name of the card.
	Name string
	// Number the card number.
	Number string
	// Image the card image metadata
	Image Image
	// ID is the identifier of the card.
	ID ID
	// Amount show how often the card is in the users collection.
	Amount int
}

type Set struct {
	// Name is the set name.
	Name string
	// Code is the set identifier.
	Code string
}

type Image struct {
	// URL the image URL
	URL string
}

type Cards struct {
	PagedResult[Card]
}

func NewCards(cards []Card, p Page) Cards {
	return Cards{
		NewPagedResult(cards, p),
	}
}

func EmptyCards(p Page) Cards {
	return Cards{
		NewEmptyResult[Card](p),
	}
}

type CardPrint struct {
	ID     ID
	Name   string
	Number string
	Code   string
	Amount int
}

func EmptyCardPrints(p Page) CardPrints {
	return CardPrints{
		NewEmptyResult[CardPrint](p),
	}
}

type CardPrints struct {
	PagedResult[CardPrint]
}

func NewCardPrints(prints []CardPrint, p Page) CardPrints {
	return CardPrints{
		NewPagedResult(prints, p),
	}
}

type CardDetail struct {
	Card   Card
	Prints CardPrints
}

// Collector user who interacts with his collection.
type Collector struct {
	ID string
}

func NewCollector(id string) Collector {
	return Collector{ID: id}
}

type Filter struct {
	Collector     *Collector
	Name          string
	Lang          string
	IDs           IDs
	OnlyCollected bool
}

func NewFilter() Filter {
	return Filter{}
}

func (f Filter) WithName(name string) Filter {
	f.Name = strings.TrimSpace(name)

	return f
}

func (f Filter) WithCollector(c Collector) Filter {
	if c.ID == "" {
		return f
	}

	f.Collector = &c

	return f
}

func (f Filter) WithOnlyCollected() Filter {
	f.OnlyCollected = true

	return f
}

func (f Filter) WithID(id ...ID) Filter {
	f.IDs = id

	return f
}

func (f Filter) WithLanguage(lang string) Filter {
	f.Lang = strings.TrimSpace(lang)

	return f
}

type CardRepository interface {
	// Find returns the cards for the requested page matching the given criteria.
	Find(ctx context.Context, filter Filter, page Page) (Cards, error)
	Prints(ctx context.Context, name string, collector Collector, page Page) (CardPrints, error)
}

type CardService interface {
	Search(ctx context.Context, name string, collector Collector, page Page) (Cards, error)
	Detail(ctx context.Context, id ID, collector Collector, page Page) (CardDetail, error)
}

type searchService struct {
	repo CardRepository
}

func NewCardService(repo CardRepository) CardService {
	return &searchService{
		repo: repo,
	}
}

func (s *searchService) Search(ctx context.Context, name string, c Collector, page Page) (Cards, error) {
	filter := NewFilter().
		WithName(name).
		WithLanguage(DefaultLang).
		WithCollector(c)
	r, err := s.repo.Find(ctx, filter, page)
	if err != nil {
		return EmptyCards(page), aerrors.NewUnknownError(err, "unable-to-execute-search")
	}

	return r, nil
}

func (s searchService) Detail(ctx context.Context, id ID, collector Collector, page Page) (CardDetail, error) {
	filter := NewFilter().
		WithLanguage(DefaultLang).
		WithID(id).
		WithCollector(collector)

	r, err := s.repo.Find(ctx, filter, DefaultPage())
	if err != nil {
		return CardDetail{}, aerrors.NewUnknownError(err, "unable-to-execute-detail-search")
	}

	if r.Size == 0 {
		err := fmt.Errorf("card with id %v not found, %w", id, ErrCardNotFound)

		return CardDetail{}, aerrors.NewNotFoundError(err, "card-detail-not-found")
	}

	if r.Size > 1 {
		err := fmt.Errorf("expected one card with id %v but found %d, %w", id, r.Size, ErrResultMismatch)

		return CardDetail{}, aerrors.NewUnknownError(err, "unexpected-detail-result")
	}

	match := r.Result[0]
	prints, err := s.repo.Prints(ctx, match.Name, collector, page)
	if err != nil {
		return CardDetail{}, aerrors.NewUnknownError(err, "unable-to-execute-prints-search")
	}

	return CardDetail{
		Card:   match,
		Prints: prints,
	}, nil
}
