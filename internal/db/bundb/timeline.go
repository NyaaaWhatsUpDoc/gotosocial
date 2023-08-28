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
	var (
		// Get paging parameters.
		minID, _ = page.GetMin()
		maxID, _ = page.GetMax()
		limit, _ = page.GetLimit()
		order, _ = page.GetOrder()

		// Make educated guess for slice size
		statusIDs = make([]string, 0, limit)

		// check requested return order based on paging
		frontToBack = (order == paging.OrderAscending)
	)

	q := t.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("statuses"), bun.Ident("status")).
		// Select only IDs from table
		Column("status.id")

	if maxID != "" {
		// return only statuses LOWER (ie., older) than maxID
		q = q.Where("? < ?", bun.Ident("status.id"), maxID)
	}

	if minID != "" {
		// return only statuses HIGHER (ie., newer) than minID
		q = q.Where("? > ?", bun.Ident("status.id"), minID)
	}

	if local {
		// return only statuses posted by local account havers
		q = q.Where("? = ?", bun.Ident("status.local"), local)
	}

	if limit > 0 {
		// limit amount of statuses returned
		q = q.Limit(limit)
	}

	if frontToBack {
		// Page down.
		q = q.Order("status.id DESC")
	} else {
		// Page up.
		q = q.Order("status.id ASC")
	}

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
	)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		return nil, gtserror.Newf("db error getting follows for account %s: %w", accountID, err)
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

	if err := q.Scan(ctx, &statusIDs); err != nil {
		return nil, err
	}

	if len(statusIDs) == 0 {
		return nil, nil
	}

	// If we're paging up, we still want statuses
	// to be sorted by ID desc, so reverse ids slice.
	// https://zchee.github.io/golang-wiki/SliceTricks/#reversing
	if !frontToBack {
		for l, r := 0, len(statusIDs)-1; l < r; l, r = l+1, r-1 {
			statusIDs[l], statusIDs[r] = statusIDs[r], statusIDs[l]
		}
	}

	// Fetch statuses for the fetched (+ sorted) IDs.
	return t.state.DB.GetStatusesByIDs(ctx, statusIDs)
}

func (t *timelineDB) GetPublicTimeline(ctx context.Context, page *paging.Page[string], local bool) ([]*gtsmodel.Status, error) {
	var (
		// Get paging parameters.
		minID, _ = page.GetMin()
		maxID, _ = page.GetMax()
		limit, _ = page.GetLimit()
		order, _ = page.GetOrder()

		// Make educated guess for slice size
		statusIDs = make([]string, 0, limit)

		// check requested return order based on paging
		frontToBack = (order == paging.OrderAscending)
	)

	q := t.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("statuses"), bun.Ident("status")).
		Column("status.id").
		// Public only.
		Where("? = ?", bun.Ident("status.visibility"), gtsmodel.VisibilityPublic).
		// Ignore boosts.
		Where("? IS NULL", bun.Ident("status.boost_of_id"))

	if maxID != "" {
		// return only statuses LOWER (ie., older) than maxID
		q = q.Where("? < ?", bun.Ident("status.id"), maxID)
	}

	if minID != "" {
		// return only statuses HIGHER (ie., newer) than minID
		q = q.Where("? > ?", bun.Ident("status.id"), minID)
	}

	if local {
		// return only statuses posted by local account havers
		q = q.Where("? = ?", bun.Ident("status.local"), local)
	}

	if limit > 0 {
		// limit amount of statuses returned
		q = q.Limit(limit)
	}

	if frontToBack {
		// Page down.
		q = q.Order("status.id DESC")
	} else {
		// Page up.
		q = q.Order("status.id ASC")
	}

	if err := q.Scan(ctx, &statusIDs); err != nil {
		return nil, err
	}

	if len(statusIDs) == 0 {
		return nil, nil
	}

	// If we're paging up, we still want statuses
	// to be sorted by ID desc, so reverse ids slice.
	// https://zchee.github.io/golang-wiki/SliceTricks/#reversing
	if !frontToBack {
		for l, r := 0, len(statusIDs)-1; l < r; l, r = l+1, r-1 {
			statusIDs[l], statusIDs[r] = statusIDs[r], statusIDs[l]
		}
	}

	// Fetch statuses for the fetched (+ sorted) IDs.
	return t.state.DB.GetStatusesByIDs(ctx, statusIDs)
}

// TODO optimize this query and the logic here, because it's slow as balls -- it takes like a literal second to return with a limit of 20!
// It might be worth serving it through a timeline instead of raw DB queries, like we do for Home feeds.
func (t *timelineDB) GetFavedTimeline(ctx context.Context, accountID string, page *paging.Page[string]) ([]*gtsmodel.StatusFave, error) {
	var (
		// Get paging parameters.
		minID, _ = page.GetMin()
		maxID, _ = page.GetMax()
		limit, _ = page.GetLimit()
		order, _ = page.GetOrder()

		// Make educated guess for slice size
		faveIDs = make([]string, 0, limit)

		// check requested return order based on paging
		frontToBack = (order == paging.OrderAscending)
	)

	fq := t.db.
		NewSelect().
		Model(&faveIDs).
		Table("status_faves").
		Column("id").
		Where("? = ?", bun.Ident("account_id"), accountID).
		Order("id DESC")

	if maxID != "" {
		// return only status faves LOWER (ie., older) than maxID
		fq = fq.Where("? < ?", bun.Ident("status_fave.id"), maxID)
	}

	if minID != "" {
		// return only status faves HIGHER (ie., newer) than minID
		fq = fq.Where("? > ?", bun.Ident("status_fave.id"), minID)
	}

	if limit > 0 {
		// limit amount of faves returned
		fq = fq.Limit(limit)
	}

	if frontToBack {
		// Page down.
		fq = fq.Order("status.id DESC")
	} else {
		// Page up.
		fq = fq.Order("status.id ASC")
	}

	err := fq.Scan(ctx, &faveIDs)
	if err != nil {
		return nil, err
	}

	if len(faveIDs) == 0 {
		return nil, db.ErrNoEntries
	}

	// If we're paging up, we still want faves
	// to be sorted by ID desc, so reverse ids slice.
	// https://zchee.github.io/golang-wiki/SliceTricks/#reversing
	if !frontToBack {
		for l, r := 0, len(faveIDs)-1; l < r; l, r = l+1, r-1 {
			faveIDs[l], faveIDs[r] = faveIDs[r], faveIDs[l]
		}
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
	var (
		// Get paging parameters.
		minID, _ = page.GetMin()
		maxID, _ = page.GetMax()
		limit, _ = page.GetLimit()
		order, _ = page.GetOrder()

		// Make educated guess for slice size
		statusIDs = make([]string, 0, limit)

		// check requested return order based on paging
		frontToBack = (order == paging.OrderAscending)
	)

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

	if maxID != "" {
		// return only statuses LOWER (ie., older) than maxID
		q = q.Where("? < ?", bun.Ident("status.id"), maxID)
	}

	if minID != "" {
		// return only statuses HIGHER (ie., newer) than minID
		q = q.Where("? > ?", bun.Ident("status.id"), minID)
	}

	if limit > 0 {
		// limit amount of statuses returned
		q = q.Limit(limit)
	}

	if frontToBack {
		// Page down.
		q = q.Order("status.id DESC")
	} else {
		// Page up.
		q = q.Order("status.id ASC")
	}

	if err := q.Scan(ctx, &statusIDs); err != nil {
		return nil, err
	}

	if len(statusIDs) == 0 {
		return nil, nil
	}

	// If we're paging up, we still want statuses
	// to be sorted by ID desc, so reverse ids slice.
	// https://zchee.github.io/golang-wiki/SliceTricks/#reversing
	if !frontToBack {
		for l, r := 0, len(statusIDs)-1; l < r; l, r = l+1, r-1 {
			statusIDs[l], statusIDs[r] = statusIDs[r], statusIDs[l]
		}
	}

	// Fetch statuses for the fetched (+ sorted) IDs.
	return t.state.DB.GetStatusesByIDs(ctx, statusIDs)
}

func (t *timelineDB) GetTagTimeline(
	ctx context.Context,
	tagID string,
	page *paging.Page[string],
) ([]*gtsmodel.Status, error) {
	var (
		// Get paging parameters.
		minID, _ = page.GetMin()
		maxID, _ = page.GetMax()
		limit, _ = page.GetLimit()
		order, _ = page.GetOrder()

		// Make educated guess for slice size
		statusIDs = make([]string, 0, limit)

		// check requested return order based on paging
		frontToBack = (order == paging.OrderAscending)
	)

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

	if maxID != "" {
		// return only statuses LOWER (ie., older) than maxID
		q = q.Where("? < ?", bun.Ident("status_to_tag.status_id"), maxID)
	}

	if minID != "" {
		// return only statuses HIGHER (ie., newer) than minID
		q = q.Where("? > ?", bun.Ident("status_to_tag.status_id"), minID)
	}

	if limit > 0 {
		// limit amount of statuses returned
		q = q.Limit(limit)
	}

	if frontToBack {
		// Page down.
		q = q.Order("status_to_tag.status_id DESC")
	} else {
		// Page up.
		q = q.Order("status_to_tag.status_id ASC")
	}

	if err := q.Scan(ctx, &statusIDs); err != nil {
		return nil, err
	}

	if len(statusIDs) == 0 {
		return nil, nil
	}

	// If we're paging up, we still want statuses
	// to be sorted by ID desc, so reverse ids slice.
	// https://zchee.github.io/golang-wiki/SliceTricks/#reversing
	if !frontToBack {
		for l, r := 0, len(statusIDs)-1; l < r; l, r = l+1, r-1 {
			statusIDs[l], statusIDs[r] = statusIDs[r], statusIDs[l]
		}
	}

	// Fetch statuses for the fetched (+ sorted) IDs.
	return t.state.DB.GetStatusesByIDs(ctx, statusIDs)
}
