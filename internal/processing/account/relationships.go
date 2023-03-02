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

package account

import (
	"context"
	"errors"
	"fmt"

	apimodel "github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// FollowersGet fetches a list of the target account's followers.
func (p *Processor) FollowersGet(ctx context.Context, requestingAccount *gtsmodel.Account, targetAccountID string) ([]apimodel.Account, gtserror.WithCode) {
	if blocked, err := p.state.DB.IsMutualBlocked(ctx, requestingAccount.ID, targetAccountID); err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	} else if blocked {
		return nil, gtserror.NewErrorNotFound(fmt.Errorf("block exists between accounts"))
	}

	accounts := []apimodel.Account{}
	follows, err := p.state.DB.GetAccountFollowedBys(ctx, targetAccountID)
	if err != nil {
		if err == db.ErrNoEntries {
			return accounts, nil
		}
		return nil, gtserror.NewErrorInternalError(err)
	}

	for _, f := range follows {
		blocked, err := p.state.DB.IsMutualBlocked(ctx, requestingAccount.ID, f.AccountID)
		if err != nil {
			return nil, gtserror.NewErrorInternalError(err)
		}
		if blocked {
			continue
		}

		if f.Account == nil {
			a, err := p.state.DB.GetAccountByID(ctx, f.AccountID)
			if err != nil {
				if err == db.ErrNoEntries {
					continue
				}
				return nil, gtserror.NewErrorInternalError(err)
			}
			f.Account = a
		}

		account, err := p.tc.AccountToAPIAccountPublic(ctx, f.Account)
		if err != nil {
			return nil, gtserror.NewErrorInternalError(err)
		}
		accounts = append(accounts, *account)
	}
	return accounts, nil
}

// FollowingGet fetches a list of the accounts that target account is following.
func (p *Processor) FollowingGet(ctx context.Context, requestingAccount *gtsmodel.Account, targetAccountID string) ([]apimodel.Account, gtserror.WithCode) {
	if blocked, err := p.state.DB.IsMutualBlocked(ctx, requestingAccount.ID, targetAccountID); err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	} else if blocked {
		return nil, gtserror.NewErrorNotFound(fmt.Errorf("block exists between accounts"))
	}

	accounts := []apimodel.Account{}
	follows, err := p.state.DB.GetAccountFollows(ctx, targetAccountID)
	if err != nil {
		if err == db.ErrNoEntries {
			return accounts, nil
		}
		return nil, gtserror.NewErrorInternalError(err)
	}

	for _, f := range follows {
		blocked, err := p.state.DB.IsMutualBlocked(ctx, requestingAccount.ID, f.AccountID)
		if err != nil {
			return nil, gtserror.NewErrorInternalError(err)
		}
		if blocked {
			continue
		}

		if f.TargetAccount == nil {
			a, err := p.state.DB.GetAccountByID(ctx, f.TargetAccountID)
			if err != nil {
				if err == db.ErrNoEntries {
					continue
				}
				return nil, gtserror.NewErrorInternalError(err)
			}
			f.TargetAccount = a
		}

		account, err := p.tc.AccountToAPIAccountPublic(ctx, f.TargetAccount)
		if err != nil {
			return nil, gtserror.NewErrorInternalError(err)
		}
		accounts = append(accounts, *account)
	}
	return accounts, nil
}

// RelationshipGet returns a relationship model describing the relationship of the targetAccount to the Authed account.
func (p *Processor) RelationshipGet(ctx context.Context, requestingAccount *gtsmodel.Account, targetAccountID string) (*apimodel.Relationship, gtserror.WithCode) {
	if requestingAccount == nil {
		return nil, gtserror.NewErrorForbidden(errors.New("not authed"))
	}

	gtsR, err := p.state.DB.GetRelationship(ctx, requestingAccount.ID, targetAccountID)
	if err != nil {
		return nil, gtserror.NewErrorInternalError(fmt.Errorf("error getting relationship: %s", err))
	}

	r, err := p.tc.RelationshipToAPIRelationship(ctx, gtsR)
	if err != nil {
		return nil, gtserror.NewErrorInternalError(fmt.Errorf("error converting relationship: %s", err))
	}

	return r, nil
}
