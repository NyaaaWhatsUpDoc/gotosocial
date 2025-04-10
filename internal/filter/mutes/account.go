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

package mutes

import (
	"context"
	"errors"
	"time"

	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtscontext"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

func (f *Filter) NotifyFromAccount(ctx context.Context, requester *gtsmodel.Account, account *gtsmodel.Account) (bool, error) {
	if requester == nil {
		// Un-authed so no account
		// is possible to be muted.
		return true, nil
	}

	// Look for mute against target.
	mute, err := f.state.DB.GetMute(
		gtscontext.SetBarebones(ctx),
		requester.ID,
		account.ID,
	)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		return false, gtserror.Newf("db error getting user mute: %w", err)
	}

	// Get current time.
	now := time.Now()

	return *mute.Notifications && mute.Expired(now), nil
}
