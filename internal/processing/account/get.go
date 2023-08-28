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
	"fmt"

	apimodel "github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// Get processes the given request for account information.
func (p *Processor) Get(ctx context.Context, requestingAccount *gtsmodel.Account, targetAccountID string) (*apimodel.Account, gtserror.WithCode) {
	targetAcc, visible, errWithCode := p.c.GetTargetAccountByID(ctx, requestingAccount, targetAccountID)
	if errWithCode != nil {
		return nil, errWithCode
	}
	if !visible {
		return p.c.GetAPIAccountBlocked(ctx, targetAcc)
	}
	return p.c.GetAPIAccount(ctx, requestingAccount, targetAcc)
}

// GetLocalByUsername processes the given request for account information targeting a local account by username.
func (p *Processor) GetLocalByUsername(ctx context.Context, requestingAccount *gtsmodel.Account, username string) (*apimodel.Account, gtserror.WithCode) {
	targetAcc, visible, errWithCode := p.c.GetTargetAccountBy(ctx, requestingAccount, func() (*gtsmodel.Account, error) {
		return p.state.DB.GetAccountByUsernameDomain(ctx, username, "")
	})
	if errWithCode != nil {
		return nil, errWithCode
	}
	if !visible {
		return p.c.GetAPIAccountBlocked(ctx, targetAcc)
	}
	return p.c.GetAPIAccount(ctx, requestingAccount, targetAcc)
}

// GetCustomCSSForUsername returns custom css for the given local username.
func (p *Processor) GetCustomCSSForUsername(ctx context.Context, username string) (string, gtserror.WithCode) {
	customCSS, err := p.state.DB.GetAccountCustomCSSByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, db.ErrNoEntries) {
			return "", gtserror.NewErrorNotFound(errors.New("account not found"))
		}
		return "", gtserror.NewErrorInternalError(fmt.Errorf("db error: %w", err))
	}
	return customCSS, nil
}
