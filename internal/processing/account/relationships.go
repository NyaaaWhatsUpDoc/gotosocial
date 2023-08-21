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

package account

import (
	"context"
	"errors"

	apimodel "github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/log"
	"github.com/superseriousbusiness/gotosocial/internal/paging"
)

// FollowersGet fetches a list of the target account's followers.
func (p *Processor) FollowersGet(ctx context.Context, requestingAccount *gtsmodel.Account, targetAccountID string, page *paging.Pager) ([]apimodel.Account, gtserror.WithCode) {
	if blocked, err := p.state.DB.IsEitherBlocked(ctx, requestingAccount.ID, targetAccountID); err != nil {
		err = gtserror.Newf("db error checking block: %w", err)
		return nil, gtserror.NewErrorInternalError(err)
	} else if blocked {
		err = gtserror.New("block exists between accounts")
		return nil, gtserror.NewErrorNotFound(err)
	}

	follows, err := p.state.DB.GetAccountFollowers(ctx, targetAccountID, page)
	if err != nil {
		if !errors.Is(err, db.ErrNoEntries) {
			err = gtserror.Newf("db error getting followers: %w", err)
			return nil, gtserror.NewErrorInternalError(err)
		}
		return []apimodel.Account{}, nil
	}

	return p.accountsFromFollows(ctx, follows, requestingAccount.ID)
}

// FollowingGet fetches a list of the accounts that target account is following.
func (p *Processor) FollowingGet(ctx context.Context, requestingAccount *gtsmodel.Account, targetAccountID string, page *paging.Pager) ([]apimodel.Account, gtserror.WithCode) {
	if blocked, err := p.state.DB.IsEitherBlocked(ctx, requestingAccount.ID, targetAccountID); err != nil {
		err = gtserror.Newf("db error checking block: %w", err)
		return nil, gtserror.NewErrorInternalError(err)
	} else if blocked {
		err = gtserror.New("block exists between accounts")
		return nil, gtserror.NewErrorNotFound(err)
	}

	follows, err := p.state.DB.GetAccountFollows(ctx, targetAccountID, page)
	if err != nil {
		if !errors.Is(err, db.ErrNoEntries) {
			err = gtserror.Newf("db error getting followers: %w", err)
			return nil, gtserror.NewErrorInternalError(err)
		}
		return []apimodel.Account{}, nil
	}

	return p.targetAccountsFromFollows(ctx, follows, requestingAccount.ID)
}

// FollowRequestsGet ...
func (p *Processor) FollowRequestsGet(ctx context.Context, requestingAccount *gtsmodel.Account, page *paging.Pager) ([]apimodel.Account, gtserror.WithCode) {
	followRequests, err := p.state.DB.GetAccountFollowRequests(ctx, requestingAccount.ID, page)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		return nil, gtserror.NewErrorInternalError(err)
	}

	accts := make([]apimodel.Account, 0, len(followRequests))
	for _, followRequest := range followRequests {
		if followRequest.Account == nil {
			// The creator of the follow doesn't exist,
			// just skip this one.
			log.WithContext(ctx).WithField("followRequest", followRequest).Warn("follow request had no associated account")
			continue
		}

		apiAcct, err := p.tc.AccountToAPIAccountPublic(ctx, followRequest.Account)
		if err != nil {
			return nil, gtserror.NewErrorInternalError(err)
		}

		accts = append(accts, *apiAcct)
	}

	return accts, nil
}

// RelationshipGet returns a relationship model describing the relationship of the targetAccount to the Authed account.
func (p *Processor) RelationshipGet(ctx context.Context, requestingAccount *gtsmodel.Account, targetAccountID string) (*apimodel.Relationship, gtserror.WithCode) {
	if requestingAccount == nil {
		return nil, gtserror.NewErrorForbidden(gtserror.New("not authed"))
	}

	gtsR, err := p.state.DB.GetRelationship(ctx, requestingAccount.ID, targetAccountID)
	if err != nil {
		return nil, gtserror.NewErrorInternalError(gtserror.Newf("error getting relationship: %s", err))
	}

	r, err := p.tc.RelationshipToAPIRelationship(ctx, gtsR)
	if err != nil {
		return nil, gtserror.NewErrorInternalError(gtserror.Newf("error converting relationship: %s", err))
	}

	return r, nil
}

func (p *Processor) accountsFromFollows(ctx context.Context, follows []*gtsmodel.Follow, requestingAccountID string) ([]apimodel.Account, gtserror.WithCode) {
	accounts := make([]apimodel.Account, 0, len(follows))
	for _, follow := range follows {
		if follow.Account == nil {
			// No account set for some reason; just skip.
			log.WithContext(ctx).WithField("follow", follow).Warn("follow had no associated account")
			continue
		}

		if blocked, err := p.state.DB.IsEitherBlocked(ctx, requestingAccountID, follow.AccountID); err != nil {
			err = gtserror.Newf("db error checking block: %w", err)
			return nil, gtserror.NewErrorInternalError(err)
		} else if blocked {
			continue
		}

		account, err := p.tc.AccountToAPIAccountPublic(ctx, follow.Account)
		if err != nil {
			err = gtserror.Newf("error converting account to api account: %w", err)
			return nil, gtserror.NewErrorInternalError(err)
		}

		accounts = append(accounts, *account)
	}
	return accounts, nil
}

func (p *Processor) targetAccountsFromFollows(ctx context.Context, follows []*gtsmodel.Follow, requestingAccountID string) ([]apimodel.Account, gtserror.WithCode) {
	accounts := make([]apimodel.Account, 0, len(follows))
	for _, follow := range follows {
		if follow.TargetAccount == nil {
			// No account set for some reason; just skip.
			log.WithContext(ctx).WithField("follow", follow).Warn("follow had no associated target account")
			continue
		}

		if blocked, err := p.state.DB.IsEitherBlocked(ctx, requestingAccountID, follow.TargetAccountID); err != nil {
			err = gtserror.Newf("db error checking block: %w", err)
			return nil, gtserror.NewErrorInternalError(err)
		} else if blocked {
			continue
		}

		account, err := p.tc.AccountToAPIAccountPublic(ctx, follow.TargetAccount)
		if err != nil {
			err = gtserror.Newf("error converting account to api account: %w", err)
			return nil, gtserror.NewErrorInternalError(err)
		}

		accounts = append(accounts, *account)
	}
	return accounts, nil
}
