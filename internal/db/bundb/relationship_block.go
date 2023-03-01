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

func (r *relationshipDB) IsBlocked(ctx context.Context, sourceAccountID string, targetAccountID string) (bool, db.Error) {
	block, err := r.GetBlock(
		gtscontext.SetBarebones(ctx),
		sourceAccountID,
		targetAccountID,
	)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		return false, err
	}
	return (block != nil), nil
}

func (r *relationshipDB) IsMutualBlocked(ctx context.Context, accountID1 string, accountID2 string) (bool, error) {
	// Look for a block in direction of account1->account2
	b1, err := r.IsBlocked(ctx, accountID1, accountID2)
	if err != nil || b1 {
		return true, err
	}

	// Look for a block in direction of account2->account1
	b2, err := r.IsBlocked(ctx, accountID2, accountID1)
	if err != nil || b2 {
		return true, err
	}

	return false, nil
}

func (r *relationshipDB) GetBlock(ctx context.Context, sourceAccountID string, targetAccountID string) (*gtsmodel.Block, db.Error) {
	// Fetch block from cache with loader callback
	block, err := r.state.Caches.GTS.Block().Load("AccountID.TargetAccountID", func() (*gtsmodel.Block, error) {
		var block gtsmodel.Block

		q := r.conn.NewSelect().Model(&block).
			Where("? = ?", bun.Ident("block.account_id"), sourceAccountID).
			Where("? = ?", bun.Ident("block.target_account_id"), targetAccountID)
		if err := q.Scan(ctx); err != nil {
			return nil, r.conn.ProcessError(err)
		}

		return &block, nil
	}, sourceAccountID, targetAccountID)
	if err != nil {
		// already processe
		return nil, err
	}

	if gtscontext.Barebones(ctx) {
		// Only a barebones model was requested.
		return block, nil
	}

	// Set the block source account
	block.Account, err = r.state.DB.GetAccountByID(
		gtscontext.SetBarebones(ctx),
		block.AccountID,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting block source account: %w", err)
	}

	// Set the block target account
	block.TargetAccount, err = r.state.DB.GetAccountByID(
		gtscontext.SetBarebones(ctx),
		block.TargetAccountID,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting block target account: %w", err)
	}

	return block, nil
}

func (r *relationshipDB) PutBlock(ctx context.Context, block *gtsmodel.Block) db.Error {
	return r.state.Caches.GTS.Block().Store(block, func() error {
		_, err := r.conn.NewInsert().Model(block).Exec(ctx)
		return r.conn.ProcessError(err)
	})
}

func (r *relationshipDB) DeleteBlockByID(ctx context.Context, id string) db.Error {
	if _, err := r.conn.
		NewDelete().
		TableExpr("? AS ?", bun.Ident("blocks"), bun.Ident("block")).
		Where("? = ?", bun.Ident("block.id"), id).
		Exec(ctx); err != nil {
		return r.conn.ProcessError(err)
	}

	// Drop any old value from cache by this ID
	r.state.Caches.GTS.Block().Invalidate("ID", id)
	return nil
}

func (r *relationshipDB) DeleteBlockByURI(ctx context.Context, uri string) db.Error {
	if _, err := r.conn.
		NewDelete().
		TableExpr("? AS ?", bun.Ident("blocks"), bun.Ident("block")).
		Where("? = ?", bun.Ident("block.uri"), uri).
		Exec(ctx); err != nil {
		return r.conn.ProcessError(err)
	}

	// Drop any old value from cache by this URI
	r.state.Caches.GTS.Block().Invalidate("URI", uri)
	return nil
}

func (r *relationshipDB) DeleteBlocksByOriginAccountID(ctx context.Context, originAccountID string) db.Error {
	blockIDs := []string{}

	q := r.conn.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("blocks"), bun.Ident("block")).
		Column("block.id").
		Where("? = ?", bun.Ident("block.account_id"), originAccountID)

	if err := q.Scan(ctx, &blockIDs); err != nil {
		return r.conn.ProcessError(err)
	}

	for _, id := range blockIDs {
		if err := r.DeleteBlockByID(ctx, id); err != nil {
			log.Errorf(ctx, "error deleting block %q: %v", id, err)
		}
	}

	return nil
}

func (r *relationshipDB) DeleteBlocksByTargetAccountID(ctx context.Context, targetAccountID string) db.Error {
	blockIDs := []string{}

	q := r.conn.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("blocks"), bun.Ident("block")).
		Column("block.id").
		Where("? = ?", bun.Ident("block.target_account_id"), targetAccountID)

	if err := q.Scan(ctx, &blockIDs); err != nil {
		return r.conn.ProcessError(err)
	}

	for _, id := range blockIDs {
		if err := r.DeleteBlockByID(ctx, id); err != nil {
			log.Errorf(ctx, "error deleting block %q: %v", id, err)
		}
	}

	return nil
}
