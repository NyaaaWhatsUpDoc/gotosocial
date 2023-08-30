// GoToSocial
// Copyright (C) GoToSocial Authors admin@gotosocial.org
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package bundb

import (
	"context"
	"strings"

	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/paging"
	"github.com/uptrace/bun"
)

// likeEscaper is a thread-safe string replacer which escapes
// common SQLite + Postgres `LIKE` wildcard chars using the
// escape character `\`. Initialized as a var in this package
// so it can be reused.
var likeEscaper = strings.NewReplacer(
	`\`, `\\`, // Escape char.
	`%`, `\%`, // Zero or more char.
	`_`, `\_`, // Exactly one char.
)

// whereLike appends a WHERE clause to the
// given SelectQuery, which searches for
// matches of `search` in the given subQuery
// using LIKE.
func whereLike(
	query *bun.SelectQuery,
	subject interface{},
	search string,
) *bun.SelectQuery {
	// Escape existing wildcard + escape
	// chars in the search query string.
	search = likeEscaper.Replace(search)

	// Add our own wildcards back in; search
	// zero or more chars around the query.
	search = `%` + search + `%`

	// Append resulting WHERE
	// clause to the main query.
	return query.Where(
		"(?) LIKE ? ESCAPE ?",
		subject, search, `\`,
	)
}

// whereStartsLike is like whereLike,
// but only searches for strings that
// START WITH `search`.
func whereStartsLike(
	query *bun.SelectQuery,
	subject interface{},
	search string,
) *bun.SelectQuery {
	// Escape existing wildcard + escape
	// chars in the search query string.
	search = likeEscaper.Replace(search)

	// Add our own wildcards back in; search
	// zero or more chars after the query.
	search += `%`

	// Append resulting WHERE
	// clause to the main query.
	return query.Where(
		"(?) LIKE ? ESCAPE ?",
		subject, search, `\`,
	)
}

// scanQueryPage appends paging parameters to a SELECT query based on given `page` and scanned `col` name. This automatically handles
// the case of nil `page, adding` min, max, limit and order to the query. Default ordering is DESC, and the returned slice is ALWAYS
// returned in descending order, even if the database query itself was ascending, as that's how our codebase handles model slices.
func scanQueryPage[T comparable](ctx context.Context, q *bun.SelectQuery, page *paging.Page[T], col bun.Ident) ([]T, error) {
	// Zero value of T.
	var zero T

	// Get paging parameters.
	min := page.GetMin()
	max := page.GetMax()
	limit := page.GetLimit()
	order := page.GetOrder()

	// Preallocate a destination slice.
	slice := make([]T, 0, limit)

	if min != zero {
		// Set page min column value.
		q = q.Where("? > ?", col, min)
	}

	if max != zero {
		// Set page max column value.
		q = q.Where("? < ?", col, max)
	}

	if limit > 0 {
		// Set page limit.
		q = q.Limit(limit)
	}

	if !order.Ascending() {
		// Default is descending.
		q = q.OrderExpr("? DESC", col)
	} else {
		// Page requires ascending.
		q = q.OrderExpr("? ASC", col)
	}

	// Perform the query, scanning result into slice.
	if err := q.Scan(ctx, &slice); err != nil {
		return nil, err
	}

	if order.Ascending() {
		// If we're paging up, we still want objects
		// to be sorted by col DESC, so reverse slice.
		// https://zchee.github.io/golang-wiki/SliceTricks/#reversing
		for l, r := 0, len(slice)-1; l < r; l, r = l+1, r-1 {
			slice[l], slice[r] = slice[r], slice[l]
		}
	}

	return slice, nil
}

// updateWhere parses []db.Where and adds it to the given update query.
func updateWhere(q *bun.UpdateQuery, where []db.Where) {
	for _, w := range where {
		query, args := parseWhere(w)
		q = q.Where(query, args...)
	}
}

// selectWhere parses []db.Where and adds it to the given select query.
func selectWhere(q *bun.SelectQuery, where []db.Where) {
	for _, w := range where {
		query, args := parseWhere(w)
		q = q.Where(query, args...)
	}
}

// deleteWhere parses []db.Where and adds it to the given where query.
func deleteWhere(q *bun.DeleteQuery, where []db.Where) {
	for _, w := range where {
		query, args := parseWhere(w)
		q = q.Where(query, args...)
	}
}

// parseWhere looks through the options on a single db.Where entry, and
// returns the appropriate query string and arguments.
func parseWhere(w db.Where) (query string, args []interface{}) {
	if w.Not {
		if w.Value == nil {
			query = "? IS NOT NULL"
			args = []interface{}{bun.Ident(w.Key)}
			return
		}

		query = "? != ?"
		args = []interface{}{bun.Ident(w.Key), w.Value}
		return
	}

	if w.Value == nil {
		query = "? IS NULL"
		args = []interface{}{bun.Ident(w.Key)}
		return
	}

	query = "? = ?"
	args = []interface{}{bun.Ident(w.Key), w.Value}
	return
}
