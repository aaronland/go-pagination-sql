package countable

import (
	"fmt"
	"github.com/aaronland/go-pagination"
	"math"
	"net/url"
)

type CountablePagination struct {
	pagination.Pagination
	total         int64
	per_page      int64
	page          int64
	pages         int64
	next_page     int64
	previous_page int64
	pages_range   []int64
}

func (p *CountablePagination) Total() int64 {
	return p.total
}

func (p *CountablePagination) PerPage() int64 {
	return p.per_page
}

func (p *CountablePagination) Page() int64 {
	return p.page
}

func (p *CountablePagination) Pages() int64 {
	return p.pages
}

func (p *CountablePagination) NextPage() int64 {
	return p.next_page
}

func (p *CountablePagination) NextURL(u *url.URL) string {

	next := p.NextPage()

	if next == 0 {
		return "#"
	}

	q := u.Query()

	q.Set("page", fmt.Sprintf("%d", next))
	u.RawQuery = q.Encode()

	return u.String()
}

func (p *CountablePagination) PreviousURL(u *url.URL) string {

	previous := p.PreviousPage()

	if previous == 0 {
		return "#"
	}

	q := u.Query()

	q.Set("page", fmt.Sprintf("%d", previous))
	u.RawQuery = q.Encode()

	return u.String()
}

func (p *CountablePagination) PreviousPage() int64 {
	return p.previous_page
}

func (p *CountablePagination) Range() []int64 {
	return p.pages_range
}

func NewPaginationFromCount(total_count int64) (pagination.Pagination, error) {

	opts, err := NewCountablePaginationOptions()

	if err != nil {
		return nil, err
	}

	return NewPaginationFromCountWithOptions(opts, total_count)
}

func NewPaginationFromCountWithOptions(opts pagination.PaginationOptions, total_count int64) (pagination.Pagination, error) {

	page := int64(math.Max(1.0, float64(opts.Page())))
	per_page := int64(math.Max(1.0, float64(opts.PerPage())))

	pages := pagination.PagesForCount(opts, total_count)

	next_page := int64(0)
	previous_page := int64(0)

	if pages > 1 {

		if page > 1 {
			previous_page = page - 1

		}

		if page < pages {
			next_page = page + 1
		}

	}

	pages_range := make([]int64, 0)

	var range_min int64
	var range_max int64
	var range_mid int64

	var rfloor int64
	var adjmin int64
	var adjmax int64

	if pages > 10 {

		range_mid = 7
		rfloor = int64(math.Floor(float64(range_mid) / 2.0))

		range_min = page - rfloor
		range_max = page + rfloor

		if range_min <= 0 {

			adjmin = int64(math.Abs(float64(range_min)))

			range_min = 1
			range_max = page + adjmin + 1
		}

		if range_max >= pages {

			adjmax = range_max - pages

			range_min = range_min - adjmax
			range_max = pages
		}

		for i := range_min; range_min <= range_max; range_min++ {
			pages_range = append(pages_range, i)
		}
	}

	pg := &CountablePagination{
		total:         total_count,
		per_page:      per_page,
		page:          page,
		pages:         pages,
		next_page:     next_page,
		previous_page: previous_page,
		pages_range:   pages_range,
	}

	return pg, nil
}
