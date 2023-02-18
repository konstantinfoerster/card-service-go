package domain

// TODO Should this be part of the domain?
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
	Total   int
	Page    int
}

func NewEmptyResult(page Page) PagedResult {
	return NewPagedResult(nil, 0, page)
}

func NewPagedResult(result []*Card, total int, page Page) PagedResult {
	if result == nil {
		result = make([]*Card, 0)
	}

	return PagedResult{
		Result:  result,
		Total:   total,
		HasMore: HasMore(page, total),
		Page:    page.Page(),
	}
}

func HasMore(page Page, total int) bool {
	count := page.Size() * page.Page()

	return total > count
}

type Repository interface {
	FindByName(name string, page Page) (PagedResult, error)
}
