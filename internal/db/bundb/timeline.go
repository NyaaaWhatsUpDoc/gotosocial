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
	"errors"
	"fmt"

	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtscontext"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/log"
	"github.com/superseriousbusiness/gotosocial/internal/paging"
	"github.com/superseriousbusiness/gotosocial/internal/state"
	"github.com/uptrace/bun"
)

type timelineDB struct {
	db    *DB
	state *state.State
}

func (t *timelineDB) GetHomeTimeline(ctx context.Context, accountID string, page *paging.Page[string], local bool) ([]*gtsmodel.Status, error) {
	// As this is the home timeline, it should be
	// populated by statuses from accounts followed
	// by accountID, and posts from accountID itself.
	//
	// So, begin by seeing who accountID follows.
	// It should be a little cheaper to do this in
	// a separate query like this, rather than using
	// a join, since followIDs are cached in memory.
	follows, err := t.state.DB.GetAccountFollows(
		gtscontext.SetBarebones(ctx),
		accountID,
		nil, // all account follows
	)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		return nil, gtserror.Newf("db error getting follows for account %s: %w", accountID, err)
	}

	q := t.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("statuses"), bun.Ident("status")).
		// Select only IDs from table
		Column("status.id")

	if local {
		// return only statuses posted by local account havers
		q = q.Where("? = ?", bun.Ident("status.local"), local)
	}

	// Extract just the accountID from each follow.
	targetAccountIDs := make([]string, len(follows)+1)
	for i, f := range follows {
		targetAccountIDs[i] = f.TargetAccountID
	}

	// Add accountID itself as a pseudo follow so that
	// accountID can see its own posts in the timeline.
	targetAccountIDs[len(targetAccountIDs)-1] = accountID

	// Select only statuses authored by
	// accounts with IDs in the slice.
	q = q.Where(
		"? IN (?)",
		bun.Ident("status.account_id"),
		bun.In(targetAccountIDs),
	)

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

	// Fetch statuses for the fetched (+ sorted) IDs.
	return t.state.DB.GetStatusesByIDs(ctx, statusIDs)
}

func (t *timelineDB) GetPublicTimeline(ctx context.Context, page *paging.Page[string], local bool) ([]*gtsmodel.Status, error) {
	q := t.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("statuses"), bun.Ident("status")).
		Column("status.id").
		// Public only.
		Where("? = ?", bun.Ident("status.visibility"), gtsmodel.VisibilityPublic).
		// Ignore boosts.
		Where("? IS NULL", bun.Ident("status.boost_of_id"))

	if local {
		// return only statuses posted by local account havers
		q = q.Where("? = ?", bun.Ident("status.local"), local)
	}

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

	// Fetch statuses for the fetched (+ sorted) IDs.
	return t.state.DB.GetStatusesByIDs(ctx, statusIDs)
}

// TODO optimize this query and the logic here, because it's slow as balls -- it takes like a literal second to return with a limit of 20!
// It might be worth serving it through a timeline instead of raw DB queries, like we do for Home feeds.
func (t *timelineDB) GetFavedTimeline(ctx context.Context, accountID string, page *paging.Page[string]) ([]*gtsmodel.StatusFave, error) {
	q := t.db.
		NewSelect().
		Table("status_faves").
		Column("id").
		Where("? = ?", bun.Ident("account_id"), accountID)

	// Scan query page, returning slice of IDs.
	// The page will add to query (if not nil):
	// - less than max
	// - greater than min
	// - order (default = DESC)
	// - limit
	faveIDs, err := scanQueryPage(ctx, q, page, "status_faves.status_id")
	if err != nil {
		return nil, err
	}

	if len(faveIDs) == 0 {
		return nil, nil
	}

	// Fetch favourite models for all of the IDs.
	faves := make([]*gtsmodel.StatusFave, 0, len(faveIDs))
	for _, id := range faveIDs {
		fave, err := t.state.DB.GetStatusFaveByID(
			// we only need a barebones model.
			gtscontext.SetBarebones(ctx),
			id,
		)
		if err != nil {
			log.Errorf(ctx, "error getting status fave: %v", err)
			continue
		}
		faves = append(faves, fave)
	}

	return faves, nil
}

func (t *timelineDB) GetListTimeline(
	ctx context.Context,
	listID string,
	page *paging.Page[string],
) ([]*gtsmodel.Status, error) {
	// Fetch all listEntries entries from the database.
	listEntries, err := t.state.DB.GetListEntries(
		// Don't need actual follows
		// for this, just the IDs.
		gtscontext.SetBarebones(ctx),
		listID,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting entries for list %s: %w", listID, err)
	}

	// Extract just the IDs of each follow.
	followIDs := make([]string, 0, len(listEntries))
	for _, listEntry := range listEntries {
		followIDs = append(followIDs, listEntry.FollowID)
	}

	// Select target account IDs from follows.
	subQ := t.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("follows"), bun.Ident("follow")).
		Column("follow.target_account_id").
		Where("? IN (?)", bun.Ident("follow.id"), bun.In(followIDs))

	// Select only status IDs created
	// by one of the followed accounts.
	q := t.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("statuses"), bun.Ident("status")).
		// Select only IDs from table
		Column("status.id").
		Where("? IN (?)", bun.Ident("status.account_id"), subQ)

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

	// Fetch statuses for the fetched (+ sorted) IDs.
	return t.state.DB.GetStatusesByIDs(ctx, statusIDs)
}

func (t *timelineDB) GetTagTimeline(
	ctx context.Context,
	tagID string,
	page *paging.Page[string],
) ([]*gtsmodel.Status, error) {
	q := t.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("status_to_tags"), bun.Ident("status_to_tag")).
		Column("status_to_tag.status_id").
		// Join with statuses for filtering.
		Join(
			"INNER JOIN ? AS ? ON ? = ?",
			bun.Ident("statuses"), bun.Ident("status"),
			bun.Ident("status.id"), bun.Ident("status_to_tag.status_id"),
		).
		// Public only.
		Where("? = ?", bun.Ident("status.visibility"), gtsmodel.VisibilityPublic).
		// This tag only.
		Where("? = ?", bun.Ident("status_to_tag.tag_id"), tagID)

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

	// Fetch statuses for the fetched (+ sorted) IDs.
	return t.state.DB.GetStatusesByIDs(ctx, statusIDs)
}
