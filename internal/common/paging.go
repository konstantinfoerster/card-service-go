package common

type Page struct {
	p, s int
}

func (p Page) Page() int {
	return p.p
}

func (p Page) Size() int {
	return p.s
}

func (p Page) Offset() int {
	return (p.Page() - 1) * p.Size()
}

func NewPage(page, size int) Page {
	firstPage := 1
	// TODO: make size configurable
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

type PagedResult[T any] struct {
	Result  []T
	HasMore bool
	Page    int
}

func NewEmptyResult[T any](page Page) PagedResult[T] {
	return NewPagedResult[T](nil, page)
}

func NewPagedResult[T any](data []T, page Page) PagedResult[T] {
	if data == nil {
		data = make([]T, 0)
	}

	return PagedResult[T]{
		Result:  data,
		HasMore: len(data) >= page.Size(),
		Page:    page.Page(),
	}
}
