package domain

import "fmt"

type Card struct {
	// ID unique card id
	ID int
	// Name the card face name
	Name string
	// Image the full image URL
	Image string
	// Amount how often this card is in a collection
	Amount int
}

type Page struct {
	p, s int
}

func (p Page) Page() int {
	return p.p
}

func (p Page) Size() int {
	return p.s
}

func NewPage(page, size int) Page {
	firstPage := 1
	// TODO make size configurable
	defaultPageSize := 10
	maxPageSize := 100

	if page <= 0 {
		page = firstPage
	}
	if size == 0 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}

	return Page{
		p: page,
		s: size,
	}
}

type PagedResult struct {
	Result  []*Card
	HasMore bool
	Page    int
}

func NewEmptyResult(page Page) PagedResult {
	return NewPagedResult(nil, page)
}

func NewPagedResult(result []*Card, page Page) PagedResult {
	if result == nil {
		result = make([]*Card, 0)
	}

	return PagedResult{
		Result:  result,
		HasMore: HasMore(page, len(result)),
		Page:    page.Page(),
	}
}

func HasMore(page Page, resultSize int) bool {
	return resultSize >= page.Size()
}

var ErrCardNotFound = fmt.Errorf("card not found")

type CardRepository interface {
	ByID(id int) (*Card, error)
	FindByName(name string, page Page) (PagedResult, error)
	FindByNameAndCollector(name string, page Page, collector Collector) (PagedResult, error)
}
