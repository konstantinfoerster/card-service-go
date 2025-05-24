package cards

type Page struct {
	p, s int
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

func (p Page) Page() int {
	return p.p
}

func (p Page) Size() int {
	return p.s
}

func (p Page) Offset() int {
	return (p.Page() - 1) * p.Size()
}
func DefaultPage() Page {
	return NewPage(0, 0)
}

type PagedResult[T any] struct {
	Result  []T
	Size    int
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
		Size:    len(data),
		HasMore: len(data) >= page.Size(),
		Page:    page.Page(),
	}
}
