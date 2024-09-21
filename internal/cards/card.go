package cards

import (
	"context"
	"fmt"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
)

const DefaultLang = "eng"

var ErrCardNotFound = fmt.Errorf("card not found")

type Card struct {
	// Set the set that the card belongs to
	Set Set
	// Name is the name of the card.
	Name string
	// Image the card image metadata
	Image Image
	// ID is the identifier of the card.
	ID int
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

func Empty(p Page) Cards {
	return Cards{
		NewEmptyResult[Card](p),
	}
}

func NewCards(cards []Card, p Page) Cards {
	return Cards{
		NewPagedResult(cards, p),
	}
}

func NewCollector(id string) Collector {
	return Collector{ID: id}
}

// Collector user who interacts with his collection.
type Collector struct {
	ID string
}

func NewFilter() Filter {
	return Filter{}
}

type Filter struct {
	Collector     *Collector
	Name          string
	OnlyCollected bool
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

type CardRepository interface {
	// Find returns the cards for the requested page matching the given criteria.
	Find(ctx context.Context, filter Filter, page Page) (Cards, error)
	// Exist returns true if a card with the given ID exist, false otherwise.
	Exist(ctx context.Context, id int) (bool, error)
	// Collect adds or removes an card from a collection.
	Collect(ctx context.Context, item Collectable, c Collector) error
	// Remove removes the item from the collection.
	Remove(ctx context.Context, item Collectable, c Collector) error
}

type CardService interface {
	Search(ctx context.Context, name string, collector Collector, page Page) (Cards, error)
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
	filter := NewFilter().WithName(name).WithCollector(c)
	r, err := s.repo.Find(ctx, filter, page)
	if err != nil {
		return Empty(page), aerrors.NewUnknownError(err, "unable-to-execute-search")
	}

	return r, nil
}
