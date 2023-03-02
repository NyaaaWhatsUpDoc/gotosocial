/*
   GoToSocial
   Copyright (C) 2021-2023 GoToSocial Authors admin@gotosocial.org

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package bundb

import (
	"context"
	"errors"
	"fmt"

	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtscontext"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/log"
	"github.com/uptrace/bun"
)

func (r *relationshipDB) GetFollowByID(ctx context.Context, id string) (*gtsmodel.Follow, error) {
	return r.getFollow(
		ctx,
		"ID",
		func(follow *gtsmodel.Follow) error {
			return r.conn.NewSelect().
				Model(follow).
				Where("? = ?", bun.Ident("id"), id).
				Scan(ctx)
		},
		id,
	)
}

func (r *relationshipDB) GetFollow(ctx context.Context, sourceAccountID string, targetAccountID string) (*gtsmodel.Follow, error) {
	return r.getFollow(
		ctx,
		"AccountID.TargetAccountID",
		func(follow *gtsmodel.Follow) error {
			return r.conn.NewSelect().
				Model(follow).
				Where("? = ?", bun.Ident("account_id"), sourceAccountID).
				Where("? = ?", bun.Ident("target_account_id"), targetAccountID).
				Scan(ctx)
		},
		sourceAccountID,
		targetAccountID,
	)
}

func (r *relationshipDB) GetFollows(ctx context.Context, ids []string) ([]*gtsmodel.Follow, error) {
	// Preallocate slice of expected length.
	follows := make([]*gtsmodel.Follow, 0, len(ids))

	for _, id := range ids {
		// Fetch follow model for this ID.
		follow, err := r.GetFollowByID(ctx, id)
		if err != nil {
			log.Errorf(ctx, "error getting follow %q: %v", id, err)
			continue
		}

		// Append to return slice.
		follows = append(follows, follow)
	}

	return follows, nil
}

func (r *relationshipDB) IsFollowing(ctx context.Context, sourceAccountID string, targetAccountID string) (bool, db.Error) {
	follow, err := r.GetFollow(
		gtscontext.SetBarebones(ctx),
		sourceAccountID,
		targetAccountID,
	)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		return false, err
	}
	return (follow != nil), nil
}

func (r *relationshipDB) IsMutualFollowing(ctx context.Context, accountID1 string, accountID2 string) (bool, db.Error) {
	// make sure account 1 follows account 2
	f1, err := r.IsFollowing(
		gtscontext.SetBarebones(ctx),
		accountID1,
		accountID2,
	)
	if err != nil || !f1 {
		return false, err
	}

	// make sure account 2 follows account 1
	f2, err := r.IsFollowing(
		gtscontext.SetBarebones(ctx),
		accountID2,
		accountID1,
	)
	if err != nil || !f2 {
		return false, err
	}

	return true, nil
}

func (r *relationshipDB) getFollow(ctx context.Context, lookup string, dbQuery func(*gtsmodel.Follow) error, keyParts ...any) (*gtsmodel.Follow, error) {
	// Fetch follow from database cache with loader callback
	follow, err := r.state.Caches.GTS.Follow().Load(lookup, func() (*gtsmodel.Follow, error) {
		var follow gtsmodel.Follow

		// Not cached! Perform database query
		if err := dbQuery(&follow); err != nil {
			return nil, r.conn.ProcessError(err)
		}

		return &follow, nil
	}, keyParts...)
	if err != nil {
		// error already processed
		return nil, err
	}

	if gtscontext.Barebones(ctx) {
		// Only a barebones model was requested.
		return follow, nil
	}

	// Set the follow source account
	follow.Account, err = r.state.DB.GetAccountByID(
		gtscontext.SetBarebones(ctx),
		follow.AccountID,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting follow source account: %w", err)
	}

	// Set the follow target account
	follow.TargetAccount, err = r.state.DB.GetAccountByID(
		gtscontext.SetBarebones(ctx),
		follow.TargetAccountID,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting follow target account: %w", err)
	}

	return follow, nil
}

func (r *relationshipDB) DeleteFollowByID(ctx context.Context, id string) error {
	if _, err := r.conn.NewDelete().
		Table("follows").
		Where("? = ?", bun.Ident("id"), id).
		Exec(ctx); err != nil {
		return r.conn.ProcessError(err)
	}

	// Drop any old value from cache by this ID
	r.state.Caches.GTS.Follow().Invalidate("ID", id)
	return nil
}

func (r *relationshipDB) DeleteFollowByURI(ctx context.Context, uri string) error {
	if _, err := r.conn.NewDelete().
		Table("follows").
		Where("? = ?", bun.Ident("uri"), uri).
		Exec(ctx); err != nil {
		return r.conn.ProcessError(err)
	}

	// Drop any old value from cache by this URI
	r.state.Caches.GTS.Follow().Invalidate("URI", uri)
	return nil
}

func (r *relationshipDB) DeleteFollowsByOriginAccountID(ctx context.Context, accountID string) error {
	var followIDs []string

	if err := r.conn.NewSelect().
		Table("follows").
		ColumnExpr("?", bun.Ident("id")).
		Where("? = ?", bun.Ident("account_id"), accountID).
		Scan(ctx, &followIDs); err != nil {
		return r.conn.ProcessError(err)
	}

	for _, id := range followIDs {
		if err := r.DeleteFollowByID(ctx, id); err != nil {
			log.Errorf(ctx, "error deleting follow %q: %v", id, err)
		}
	}

	return nil
}

func (r *relationshipDB) DeleteFollowsByTargetAccountID(ctx context.Context, accountID string) error {
	var followIDs []string

	if err := r.conn.NewSelect().
		Table("follows").
		ColumnExpr("?", bun.Ident("id")).
		Where("? = ?", bun.Ident("target_account_id"), accountID).
		Scan(ctx, &followIDs); err != nil {
		return r.conn.ProcessError(err)
	}

	for _, id := range followIDs {
		if err := r.DeleteFollowByID(ctx, id); err != nil {
			log.Errorf(ctx, "error deleting follow %q: %v", id, err)
		}
	}

	return nil
}
