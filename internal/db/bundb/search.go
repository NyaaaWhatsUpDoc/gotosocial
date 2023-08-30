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

	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/log"
	"github.com/superseriousbusiness/gotosocial/internal/paging"
	"github.com/superseriousbusiness/gotosocial/internal/state"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
)

// todo: currently we pass an 'offset' parameter into functions owned by this struct,
// which is ignored.
//
// The idea of 'offset' is to allow callers to page through results without supplying
// maxID or minID params; they simply use the offset as more or less a 'page number'.
// This works fine when you're dealing with something like Elasticsearch, but for
// SQLite or Postgres 'LIKE' queries it doesn't really, because for each higher offset
// you have to calculate the value of all the previous offsets as well *within the
// execution time of the query*. It's MUCH more efficient to page using maxID and
// minID for queries like this. For now, then, we just ignore the offset and hope that
// the caller will page using maxID and minID instead.
//
// In future, however, it would be good to support offset in a way that doesn't totally
// destroy database queries. One option would be to cache previous offsets when paging
// down (which is the most common use case).
//
// For example, say a caller makes a call with offset 0: we run the query as normal,
// and in a 10 minute cache or something, store the next maxID value as it would be for
// offset 1, for the supplied query, limit, following, etc. Then when they call for
// offset 1, instead of supplying 'offset' in the query and causing slowdown, we check
// the cache to see if we have the next maxID value stored for that query, and use that
// instead. If a caller out of the blue requests offset 4 or something, on an empty cache,
// we could run the previous 4 queries and store the offsets for those before making the
// 5th call for page 4.
//
// This isn't ideal, of course, but at least we could cover the most common use case of
// a caller paging down through results.
type searchDB struct {
	db    *DB
	state *state.State
}

// Query example (SQLite):
//
//	SELECT "account"."id" FROM "accounts" AS "account"
//	WHERE (("account"."domain" IS NULL) OR ("account"."domain" != "account"."username"))
//	AND ("account"."id" < 'ZZZZZZZZZZZZZZZZZZZZZZZZZZ')
//	AND ("account"."id" IN (SELECT "target_account_id" FROM "follows" WHERE ("account_id" = '016T5Q3SQKBT337DAKVSKNXXW1')))
//	AND ((SELECT LOWER("account"."username" || COALESCE("account"."display_name", '') || COALESCE("account"."note", '')) AS "account_text") LIKE '%turtle%' ESCAPE '\')
//	ORDER BY "account"."id" DESC LIMIT 10
func (s *searchDB) SearchForAccounts(
	ctx context.Context,
	accountID string,
	query string,
	page *paging.Page[string],
	following bool,
	offset int,
) ([]*gtsmodel.Account, error) {
	q := s.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("accounts"), bun.Ident("account")).
		// Select only IDs from table.
		Column("account.id").
		// Try to ignore instance accounts. Account domain must
		// be either nil or, if set, not equal to the account's
		// username (which is commonly used to indicate it's an
		// instance service account).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				Where("? IS NULL", bun.Ident("account.domain")).
				WhereOr("? != ?", bun.Ident("account.domain"), bun.Ident("account.username"))
		})

	if following {
		// Select only from accounts followed by accountID.
		q = q.Where(
			"? IN (?)",
			bun.Ident("account.id"),
			s.followedAccounts(accountID),
		)
	}

	// Select account text as subquery.
	accountTextSubq := s.accountText(following)

	// Search using LIKE for matches of query
	// string within accountText subquery.
	q = whereLike(q, accountTextSubq, query)

	// Scan query page, returning slice of IDs.
	// The page will add to query (if not nil):
	// - less than max
	// - greater than min
	// - order (default = DESC)
	// - limit
	accountIDs, err := scanQueryPage(ctx, q, page, "account.id")
	if err != nil {
		return nil, err
	}

	if len(accountIDs) == 0 {
		return nil, nil
	}

	accounts := make([]*gtsmodel.Account, 0, len(accountIDs))
	for _, id := range accountIDs {
		// Fetch account from db for ID
		account, err := s.state.DB.GetAccountByID(ctx, id)
		if err != nil {
			log.Errorf(ctx, "error fetching account %q: %v", id, err)
			continue
		}

		// Append account to slice
		accounts = append(accounts, account)
	}

	return accounts, nil
}

// followedAccounts returns a subquery that selects only IDs
// of accounts that are followed by the given accountID.
func (s *searchDB) followedAccounts(accountID string) *bun.SelectQuery {
	return s.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("follows"), bun.Ident("follow")).
		Column("follow.target_account_id").
		Where("? = ?", bun.Ident("follow.account_id"), accountID)
}

// statusText returns a subquery that selects a concatenation
// of account username and display name as "account_text". If
// `following` is true, then account note will also be included
// in the concatenation.
func (s *searchDB) accountText(following bool) *bun.SelectQuery {
	var (
		accountText = s.db.NewSelect()
		query       string
		args        []interface{}
	)

	if following {
		// If querying for accounts we follow,
		// include note in text search params.
		args = []interface{}{
			bun.Ident("account.username"),
			bun.Ident("account.display_name"), "",
			bun.Ident("account.note"), "",
			bun.Ident("account_text"),
		}
	} else {
		// If querying for accounts we're not following,
		// don't include note in text search params.
		args = []interface{}{
			bun.Ident("account.username"),
			bun.Ident("account.display_name"), "",
			bun.Ident("account_text"),
		}
	}

	// SQLite and Postgres use different syntaxes for
	// concatenation, and we also need to use a
	// different number of placeholders depending on
	// following/not following. COALESCE calls ensure
	// that we're not trying to concatenate null values.
	d := s.db.Dialect().Name()
	switch {

	case d == dialect.SQLite && following:
		query = "LOWER(? || COALESCE(?, ?) || COALESCE(?, ?)) AS ?"

	case d == dialect.SQLite && !following:
		query = "LOWER(? || COALESCE(?, ?)) AS ?"

	case d == dialect.PG && following:
		query = "LOWER(CONCAT(?, COALESCE(?, ?), COALESCE(?, ?))) AS ?"

	case d == dialect.PG && !following:
		query = "LOWER(CONCAT(?, COALESCE(?, ?))) AS ?"

	default:
		panic("db conn was neither pg not sqlite")
	}

	return accountText.ColumnExpr(query, args...)
}

// Query example (SQLite):
//
//	SELECT "status"."id"
//	FROM "statuses" AS "status"
//	WHERE ("status"."boost_of_id" IS NULL)
//	AND (("status"."account_id" = '01F8MH1H7YV1Z7D2C8K2730QBF') OR ("status"."in_reply_to_account_id" = '01F8MH1H7YV1Z7D2C8K2730QBF'))
//	AND ("status"."id" < 'ZZZZZZZZZZZZZZZZZZZZZZZZZZ')
//	AND ((SELECT LOWER("status"."content" || COALESCE("status"."content_warning", '')) AS "status_text") LIKE '%hello%' ESCAPE '\')
//	ORDER BY "status"."id" DESC LIMIT 10
func (s *searchDB) SearchForStatuses(
	ctx context.Context,
	accountID string,
	query string,
	page *paging.Page[string],
	offset int,
) ([]*gtsmodel.Status, error) {
	q := s.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("statuses"), bun.Ident("status")).
		// Select only IDs from table
		Column("status.id").
		// Ignore boosts.
		Where("? IS NULL", bun.Ident("status.boost_of_id")).
		// Select only statuses created by
		// accountID or replying to accountID.
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				Where("? = ?", bun.Ident("status.account_id"), accountID).
				WhereOr("? = ?", bun.Ident("status.in_reply_to_account_id"), accountID)
		})

	// Select status text as subquery.
	statusTextSubq := s.statusText()

	// Search using LIKE for matches of query
	// string within statusText subquery.
	q = whereLike(q, statusTextSubq, query)

	// Scan query page, returning slice of IDs.
	// The page will add to query (if not nil):
	// - less than max
	// - greater than min
	// - order (default = DESC)
	// - limit
	statusIDs, err := scanQueryPage(ctx, q, page, "status.id")
	if err != nil {
		return nil, err
	}

	if len(statusIDs) == 0 {
		return nil, nil
	}

	statuses := make([]*gtsmodel.Status, 0, len(statusIDs))
	for _, id := range statusIDs {
		// Fetch status from db for ID
		status, err := s.state.DB.GetStatusByID(ctx, id)
		if err != nil {
			log.Errorf(ctx, "error fetching status %q: %v", id, err)
			continue
		}

		// Append status to slice
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// statusText returns a subquery that selects a concatenation
// of status content and content warning as "status_text".
func (s *searchDB) statusText() *bun.SelectQuery {
	statusText := s.db.NewSelect()

	// SQLite and Postgres use different
	// syntaxes for concatenation.
	switch s.db.Dialect().Name() {

	case dialect.SQLite:
		statusText = statusText.ColumnExpr(
			"LOWER(? || COALESCE(?, ?)) AS ?",
			bun.Ident("status.content"), bun.Ident("status.content_warning"), "",
			bun.Ident("status_text"))

	case dialect.PG:
		statusText = statusText.ColumnExpr(
			"LOWER(CONCAT(?, COALESCE(?, ?))) AS ?",
			bun.Ident("status.content"), bun.Ident("status.content_warning"), "",
			bun.Ident("status_text"))

	default:
		panic("db conn was neither pg not sqlite")
	}

	return statusText
}

// Query example (SQLite):
//
//	SELECT "tag"."id" FROM "tags" AS "tag"
//	WHERE ("tag"."id" < 'ZZZZZZZZZZZZZZZZZZZZZZZZZZ')
//	AND (("tag"."name") LIKE 'welcome%' ESCAPE '\')
//	ORDER BY "tag"."id" DESC LIMIT 10
func (s *searchDB) SearchForTags(
	ctx context.Context,
	query string,
	page *paging.Page[string],
	offset int,
) ([]*gtsmodel.Tag, error) {
	q := s.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("tags"), bun.Ident("tag")).
		// Select only IDs from table
		Column("tag.id")

	// Normalize tag 'name' string.
	name := strings.TrimSpace(query)
	name = strings.ToLower(name)

	// Search using LIKE for tags that start with `name`.
	q = whereStartsLike(q, bun.Ident("tag.name"), name)

	// Scan query page, returning slice of IDs.
	// The page will add to query (if not nil):
	// - less than max
	// - greater than min
	// - order (default = DESC)
	// - limit
	tagIDs, err := scanQueryPage(ctx, q, page, "tag.id")
	if err != nil {
		return nil, err
	}

	if len(tagIDs) == 0 {
		return nil, nil
	}

	tags := make([]*gtsmodel.Tag, 0, len(tagIDs))
	for _, id := range tagIDs {
		// Fetch tag from db for ID
		tag, err := s.state.DB.GetTag(ctx, id)
		if err != nil {
			log.Errorf(ctx, "error fetching tag %q: %v", id, err)
			continue
		}

		// Append status to slice
		tags = append(tags, tag)
	}

	return tags, nil
}
