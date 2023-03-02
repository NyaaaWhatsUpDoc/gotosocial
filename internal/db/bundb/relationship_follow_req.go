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

func (r *relationshipDB) GetFollowRequestByID(ctx context.Context, id string) (*gtsmodel.FollowRequest, error) {
	return r.getFollowRequest(
		ctx,
		"ID",
		func(followReq *gtsmodel.FollowRequest) error {
			return r.conn.NewSelect().
				Model(followReq).
				Where("? = ?", bun.Ident("id"), id).
				Scan(ctx)
		},
		id,
	)
}

func (r *relationshipDB) GetFollowRequest(ctx context.Context, sourceAccountID string, targetAccountID string) (*gtsmodel.FollowRequest, error) {
	return r.getFollowRequest(
		ctx,
		"AccountID.TargetAccountID",
		func(followReq *gtsmodel.FollowRequest) error {
			return r.conn.NewSelect().
				Model(followReq).
				Where("? = ?", bun.Ident("account_id"), sourceAccountID).
				Where("? = ?", bun.Ident("target_account_id"), targetAccountID).
				Scan(ctx)
		},
		sourceAccountID,
		targetAccountID,
	)
}

func (r *relationshipDB) GetFollowRequests(ctx context.Context, ids []string) ([]*gtsmodel.FollowRequest, error) {
	// Preallocate slice of expected length.
	followReqs := make([]*gtsmodel.FollowRequest, 0, len(ids))

	for _, id := range ids {
		// Fetch follow request model for this ID.
		followReq, err := r.GetFollowRequestByID(ctx, id)
		if err != nil {
			log.Errorf(ctx, "error getting follow request %q: %v", id, err)
			continue
		}

		// Append to return slice.
		followReqs = append(followReqs, followReq)
	}

	return followReqs, nil
}

func (r *relationshipDB) IsFollowRequested(ctx context.Context, sourceAccountID string, targetAccountID string) (bool, db.Error) {
	followReq, err := r.GetFollowRequest(
		gtscontext.SetBarebones(ctx),
		sourceAccountID,
		targetAccountID,
	)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		return false, err
	}
	return (followReq != nil), nil
}

func (r *relationshipDB) getFollowRequest(ctx context.Context, lookup string, dbQuery func(*gtsmodel.FollowRequest) error, keyParts ...any) (*gtsmodel.FollowRequest, error) {
	// Fetch follow request from database cache with loader callback
	followReq, err := r.state.Caches.GTS.FollowRequest().Load(lookup, func() (*gtsmodel.FollowRequest, error) {
		var followReq gtsmodel.FollowRequest

		// Not cached! Perform database query
		if err := dbQuery(&followReq); err != nil {
			return nil, r.conn.ProcessError(err)
		}

		return &followReq, nil
	}, keyParts...)
	if err != nil {
		// error already processed
		return nil, err
	}

	if gtscontext.Barebones(ctx) {
		// Only a barebones model was requested.
		return followReq, nil
	}

	// Set the follow request source account
	followReq.Account, err = r.state.DB.GetAccountByID(
		gtscontext.SetBarebones(ctx),
		followReq.AccountID,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting follow request source account: %w", err)
	}

	// Set the follow request target account
	followReq.TargetAccount, err = r.state.DB.GetAccountByID(
		gtscontext.SetBarebones(ctx),
		followReq.TargetAccountID,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting follow request target account: %w", err)
	}

	return followReq, nil
}

func (r *relationshipDB) AcceptFollowRequest(ctx context.Context, sourceAccountID string, targetAccountID string) (*gtsmodel.Follow, db.Error) {
	var follow *gtsmodel.Follow

	if err := r.conn.RunInTx(ctx, func(tx bun.Tx) error {
		followReq := new(gtsmodel.FollowRequest)

		// get original follow request
		if err := tx.
			NewSelect().
			Model(followReq).
			Where("? = ?", bun.Ident("follow_request.account_id"), sourceAccountID).
			Where("? = ?", bun.Ident("follow_request.target_account_id"), targetAccountID).
			Scan(ctx); err != nil {
			return err
		}

		// create a new follow to 'replace' the request with
		follow = &gtsmodel.Follow{
			ID:              followReq.ID,
			AccountID:       sourceAccountID,
			TargetAccountID: targetAccountID,
			URI:             followReq.URI,
		}

		// if the follow already exists, just update the URI -- we don't need to do anything else
		if _, err := tx.
			NewInsert().
			Model(follow).
			On("CONFLICT (?,?) DO UPDATE set ? = ?", bun.Ident("account_id"), bun.Ident("target_account_id"), bun.Ident("uri"), follow.URI).
			Exec(ctx); err != nil {
			return err
		}

		// now remove the follow request
		if _, err := tx.
			NewDelete().
			TableExpr("? AS ?", bun.Ident("follow_requests"), bun.Ident("follow_request")).
			Where("? = ?", bun.Ident("follow_request.id"), followReq.ID).
			Exec(ctx); err != nil {
			return err
		}

		return nil
	}); err != nil {
		// already processed
		return nil, err
	}

	// Invalidate follow / request from caches (as request was deleted).
	r.state.Caches.GTS.FollowRequest().Invalidate("ID", follow.ID)
	r.state.Caches.GTS.Follow().Invalidate("ID", follow.ID)

	return follow, nil
}

func (r *relationshipDB) RejectFollowRequest(ctx context.Context, sourceAccountID string, targetAccountID string) db.Error {
	if err := r.conn.RunInTx(ctx, func(tx bun.Tx) error {
		followReq := new(gtsmodel.FollowRequest)

		// get original follow request
		if err := tx.
			NewSelect().
			Model(followReq).
			Where("? = ?", bun.Ident("follow_request.account_id"), sourceAccountID).
			Where("? = ?", bun.Ident("follow_request.target_account_id"), targetAccountID).
			Scan(ctx); err != nil {
			return err
		}

		// now delete it from the database by ID
		if _, err := tx.
			NewDelete().
			TableExpr("? AS ?", bun.Ident("follow_requests"), bun.Ident("follow_request")).
			Where("? = ?", bun.Ident("follow_request.id"), followReq.ID).
			Exec(ctx); err != nil {
			return err
		}

		return nil
	}); err != nil {
		// already processed
		return err
	}

	// Invalidate existing follow request from cache.
	r.state.Caches.GTS.FollowRequest().Invalidate(
		"AccountID.TargetAccountID",
		sourceAccountID,
		targetAccountID,
	)

	return nil
}

func (r *relationshipDB) DeleteFollowRequestByID(ctx context.Context, id string) error {
	if _, err := r.conn.NewDelete().
		Table("follow_requests").
		Where("? = ?", bun.Ident("id"), id).
		Exec(ctx); err != nil {
		return r.conn.ProcessError(err)
	}

	// Drop any old value from cache by this ID
	r.state.Caches.GTS.FollowRequest().Invalidate("ID", id)
	return nil
}

func (r *relationshipDB) DeleteFollowRequestByURI(ctx context.Context, uri string) error {
	if _, err := r.conn.NewDelete().
		Table("follow_requests").
		Where("? = ?", bun.Ident("uri"), uri).
		Exec(ctx); err != nil {
		return r.conn.ProcessError(err)
	}

	// Drop any old value from cache by this URI
	r.state.Caches.GTS.FollowRequest().Invalidate("URI", uri)
	return nil
}

func (r *relationshipDB) DeleteFollowRequestsByOriginAccountID(ctx context.Context, accountID string) error {
	var followReqIDs []string

	if err := r.conn.NewSelect().
		Table("follow_requests").
		ColumnExpr("?", bun.Ident("id")).
		Where("? = ?", bun.Ident("account_id"), accountID).
		Scan(ctx, &followReqIDs); err != nil {
		return r.conn.ProcessError(err)
	}

	for _, id := range followReqIDs {
		if err := r.DeleteFollowRequestByID(ctx, id); err != nil {
			log.Errorf(ctx, "error deleting follow request %q: %v", id, err)
		}
	}

	return nil
}

func (r *relationshipDB) DeleteFollowRequestsByTargetAccountID(ctx context.Context, accountID string) error {
	var followReqIDs []string

	if err := r.conn.NewSelect().
		Table("follow_requests").
		ColumnExpr("?", bun.Ident("id")).
		Where("? = ?", bun.Ident("target_account_id"), accountID).
		Scan(ctx, &followReqIDs); err != nil {
		return r.conn.ProcessError(err)
	}

	for _, id := range followReqIDs {
		if err := r.DeleteFollowRequestByID(ctx, id); err != nil {
			log.Errorf(ctx, "error deleting follow request %q: %v", id, err)
		}
	}

	return nil
}
