package sql

import (
	"context"
	gosql "database/sql"
	"fmt"
	"regexp"
	"strings"
)

func CountRowsQuery(input string, count_fields ...string) string {

	output := input

	re_orderby := regexp.MustCompile(` ORDER BY .*$`)
	re_limit := regexp.MustCompile(` LIMIT .*$`)
	re_select := regexp.MustCompile(`(?i)^SELECT (.*?) FROM`)

	output = re_orderby.ReplaceAllString(output, "")
	output = re_limit.ReplaceAllString(output, "")

	if len(count_fields) > 0 {

		repl_str := fmt.Sprintf("SELECT COUNT(%s) AS count FROM", strings.Join(count_fields, ","))
		output = re_select.ReplaceAllString(output, repl_str)

	} else {

		output = re_select.ReplaceAllStringFunc(output, func(m string) string {
			return fmt.Sprintf("SELECT COUNT(%s) AS count FROM", "*")
		})

	}

	return output
}

func CountRows(ctx context.Context, db *gosql.DB, q string, args ...any) (int64, error) {

	count_q := CountRowsQuery(q)
	row := db.QueryRowContext(ctx, count_q, args...)

	var count int64
	err := row.Scan(&count)

	return count, err
}

func CountRowsWithCountFields(ctx context.Context, db *gosql.DB, count_fields []string, q string, args ...any) (int64, error) {

	count_q := CountRowsQuery(q, count_fields...)
	row := db.QueryRowContext(ctx, count_q, args...)

	var count int64
	err := row.Scan(&count)

	return count, err
}
