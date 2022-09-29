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
	page = page - 1
	if page < 0 {
		page = 0
	}
	if size == 0 {
		size = 10
	}
	if size > 200 {
		size = 200
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
}

func NewPagedResult(result []*Card, total int, page Page) PagedResult {
	return PagedResult{
		Result:  result,
		Total:   total,
		HasMore: HasMore(page, total),
	}
}

func HasMore(page Page, total int) bool {
	count := page.Size() * (page.Page() + 1)
	return total > count
}

type Repository interface {
	FindByName(name string, page Page) (PagedResult, error)
}
