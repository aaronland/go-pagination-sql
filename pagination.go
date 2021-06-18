package sql

import (
	gosql "database/sql"
	"fmt"
	"github.com/aaronland/go-pagination"
	"github.com/aaronland/go-pagination/countable"
	_ "log"
	"math"
	"strings"
)

type PaginatedResponse interface {
	Rows() *sql.Rows
	Pagination() pagination.Pagination
}

type PaginatedResponseCallback func(PaginatedResponse) error

type DefaultPaginatedResponse struct {
	rows       *gosql.Rows
	pagination pagination.Pagination
}

func (r *DefaultPaginatedResponse) Rows() *gosql.Rows {
	return r.rows
}

func (r *DefaultPaginatedResponse) Pagination() pagination.Pagination {
	return r.pagination
}

func QueryPaginatedAll(db *sql.DB, opts pagination.PaginationOptions, cb PaginatedResponseCallback, query string, args ...interface{}) error {

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

func QueryPaginated(db *sql.DB, opts pagination.PaginationOptions, query string, args ...interface{}) (PaginatedResponse, error) {

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

	pg, err := countable.NewPaginationFromCountWithOptions(opts, total_count)

	if err != nil {
		return nil, err
	}

	rsp := DefaultPaginatedResponse{
		pagination: pg,
		rows:       rows,
	}

	return &rsp, nil
}
