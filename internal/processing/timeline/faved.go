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

package timeline

import (
	"context"
	"errors"
	"fmt"

	apimodel "github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/log"
	"github.com/superseriousbusiness/gotosocial/internal/oauth"
	"github.com/superseriousbusiness/gotosocial/internal/paging"
)

func (p *Processor) FavedTimelineGet(ctx context.Context, authed *oauth.Auth, page *paging.Page[string]) (*apimodel.PageableResponse, gtserror.WithCode) {
	faves, err := p.state.DB.GetFavedTimeline(ctx, authed.Account.ID, page)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		err = fmt.Errorf("FavedTimelineGet: db error getting statuses: %w", err)
		return nil, gtserror.NewErrorInternalError(err)
	}

	count := len(faves)
	if count == 0 {
		return paging.EmptyResponse(), nil
	}

	// Func to fetch relevant status for fave at index.
	getFaveStatusIdx := func(i int) *gtsmodel.Status {
		statusID := faves[i].StatusID

		// NOTE: passing in an anonymous function here that is expected
		// to access members of a slice, but instead converts on-the-fly
		// favourites -> statuses isn't *ideal*. Buuuuuuut the alternative
		// is converting all the favourites to a slice of status IDs, fetching
		// all of those as statuses, then finally performing this same code
		// minus the `GetStatusByID()`. So this way ends up more concise.
		status, err := p.state.DB.GetStatusByID(ctx, statusID)
		if err != nil {
			log.Errorf(ctx, "error getting status for fave: %v", err)
			return nil
		}

		return status
	}

	// Get a filtered slice of frontend API status models.
	items, minID, maxID := p.c.GetVisibleAPIStatusesPaged(ctx,
		authed.Account,
		getFaveStatusIdx,
		len(faves),
	)

	return paging.PackageResponse(paging.ResponseParams[string]{
		Items: items,
		Path:  "/api/v1/favourites",
		Next:  page.Next(maxID),
		Prev:  page.Prev(minID),
	}), nil
}
