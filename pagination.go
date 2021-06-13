package pagination

import (
	"database/sql"
	"fmt"
	aa "github.com/aaronland/go-pagination"
	aa_countable "github.com/aaronland/go-pagination/countable"
	_ "log"
	"math"
	"strings"
)

type PaginatedResponse interface {
	Rows() *sql.Rows
	Pagination() aa.Pagination
}

type PaginatedResponseCallback func(PaginatedResponse) error

type DefaultPaginatedResponse struct {
	rows       *sql.Rows
	pagination aa.Pagination
}

func (r *DefaultPaginatedResponse) Rows() *sql.Rows {
	return r.rows
}

func (r *DefaultPaginatedResponse) Pagination() aa.Pagination {
	return r.pagination
}

func QueryPaginatedAll(db *sql.DB, opts aa.PaginationOptions, cb PaginatedResponseCallback, query string, args ...interface{}) error {

	for {

		rsp, err := QueryPaginated(db, opts, query, args...)

		if err != nil {
			return err
		}

		err = cb(rsp)

		if err != nil {
			return err
		}

		pg := rsp.Pagination()

		next := pg.NextPage()

		if next == 0 {
			break
		}

		opts.Page(next)
	}

	return nil
}

func QueryPaginated(db *sql.DB, opts aa.PaginationOptions, query string, args ...interface{}) (PaginatedResponse, error) {

	done_ch := make(chan bool)
	err_ch := make(chan error)
	count_ch := make(chan int64)
	rows_ch := make(chan *sql.Rows)

	var page int
	var per_page int
	var spill int

	go func() {

		defer func() {
			done_ch <- true
		}()

		parts := strings.Split(query, " FROM ")
		parts = strings.Split(parts[1], " LIMIT ")
		parts = strings.Split(parts[0], " ORDER ")

		conditions := parts[0]

		count_query := fmt.Sprintf("SELECT COUNT(%s) FROM %s", opts.Column(), conditions)
		// log.Println("COUNT QUERY", count_query)

		row := db.QueryRow(count_query, args...)

		var count int64
		err := row.Scan(&count)

		if err != nil {
			err_ch <- err
			return
		}

		// log.Println("COUNT", count)
		count_ch <- count
	}()

	go func() {

		defer func() {
			done_ch <- true
		}()

		// please make fewer ((((())))) s
		// (20180409/thisisaaronland)

		page = int(math.Max(1.0, float64(opts.Page())))
		per_page = int(math.Max(1.0, float64(opts.PerPage())))
		spill = int(math.Max(1.0, float64(opts.Spill())))

		if spill >= per_page {
			spill = per_page - 1
		}

		offset := 0
		limit := per_page

		offset = (page - 1) * per_page

		query = fmt.Sprintf("%s LIMIT %d OFFSET %d", query, limit, offset)
		// log.Println("QUERY", query)

		rows, err := db.Query(query, args...)

		if err != nil {
			err_ch <- err
			return
		}

		rows_ch <- rows
	}()

	var total_count int64
	var rows *sql.Rows

	remaining := 2

	for remaining > 0 {

		select {
		case <-done_ch:
			remaining -= 1
		case e := <-err_ch:
			return nil, e
		case i := <-count_ch:
			total_count = i
		case r := <-rows_ch:
			rows = r
		default:
			//
		}
	}

	/*
		pages := int(math.Ceil(float64(total_count) / float64(per_page)))

		next_page := 0
		previous_page := 0

		if pages > 1 {

		if page > 1 {
			previous_page = page  - 1

		}

		if page < pages {
			next_page = page + 1
		}

		}

		pages_range := make([]int, 0)

		var range_min int
		var range_max int
		var range_mid int

		var rfloor int
		var adjmin int
		var adjmax int

		if pages > 10 {

		   range_mid = 7
		   rfloor = int(math.Floor(float64(range_mid) / 2.0))

		   range_min = page - rfloor
		   range_max = page + rfloor

		   if range_min <= 0 {

		   	adjmin = int(math.Abs(float64(range_min)))

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

	*/

	pg, err := aa_countable.NewPaginationFromCountWithOptions(opts, total_count)

	if err != nil {
		return nil, err
	}

	rsp := DefaultPaginatedResponse{
		pagination: pg,
		rows:       rows,
	}

	return &rsp, nil
}
