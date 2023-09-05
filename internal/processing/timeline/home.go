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

	apimodel "github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/oauth"
	"github.com/superseriousbusiness/gotosocial/internal/paging"
	"github.com/superseriousbusiness/gotosocial/internal/state"
	"github.com/superseriousbusiness/gotosocial/internal/timeline"
	"github.com/superseriousbusiness/gotosocial/internal/typeutils"
	"github.com/superseriousbusiness/gotosocial/internal/visibility"
)

// HomeTimelineGrab returns a function that satisfies GrabFunction for home timelines.
func HomeTimelineGrab(state *state.State) timeline.GrabFunction {
	return func(ctx context.Context, accountID string, page *paging.Page[string]) ([]timeline.Timelineable, bool, error) {
		statuses, err := state.DB.GetHomeTimeline(ctx, accountID, page, false)
		if err != nil && !errors.Is(err, db.ErrNoEntries) {
			err = gtserror.Newf("error getting statuses from db: %w", err)
			return nil, false, err
		}

		count := len(statuses)
		if count == 0 {
			// We just don't have enough statuses
			// left in the db so return stop = true.
			return nil, true, nil
		}

		items := make([]timeline.Timelineable, count)
		for i, s := range statuses {
			items[i] = s
		}

		return items, false, nil
	}
}

// HomeTimelineFilter returns a function that satisfies FilterFunction for home timelines.
func HomeTimelineFilter(state *state.State, filter *visibility.Filter) timeline.FilterFunction {
	return func(ctx context.Context, accountID string, item timeline.Timelineable) (shouldIndex bool, err error) {
		status, ok := item.(*gtsmodel.Status)
		if !ok {
			err = gtserror.New("could not convert item to *gtsmodel.Status")
			return false, err
		}

		requestingAccount, err := state.DB.GetAccountByID(ctx, accountID)
		if err != nil {
			err = gtserror.Newf("error getting account with id %s: %w", accountID, err)
			return false, err
		}

		timelineable, err := filter.StatusHomeTimelineable(ctx, requestingAccount, status)
		if err != nil {
			err = gtserror.Newf("error checking hometimelineability of status %s for account %s: %w", status.ID, accountID, err)
			return false, err
		}

		return timelineable, nil
	}
}

// HomeTimelineStatusPrepare returns a function that satisfies PrepareFunction for home timelines.
func HomeTimelineStatusPrepare(state *state.State, tc typeutils.TypeConverter) timeline.PrepareFunction {
	return func(ctx context.Context, accountID string, itemID string) (timeline.Preparable, error) {
		status, err := state.DB.GetStatusByID(ctx, itemID)
		if err != nil {
			err = gtserror.Newf("error getting status with id %s: %w", itemID, err)
			return nil, err
		}

		requestingAccount, err := state.DB.GetAccountByID(ctx, accountID)
		if err != nil {
			err = gtserror.Newf("error getting account with id %s: %w", accountID, err)
			return nil, err
		}

		return tc.StatusToAPIStatus(ctx, status, requestingAccount)
	}
}

func (p *Processor) HomeTimelineGet(ctx context.Context, authed *oauth.Auth, page *paging.Page[string], local bool) (*apimodel.PageableResponse, gtserror.WithCode) {
	statuses, err := p.state.Timelines.Home.GetTimeline(ctx, authed.Account.ID, page, local)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		err = gtserror.Newf("error getting statuses: %w", err)
		return nil, gtserror.NewErrorInternalError(err)
	}

	count := len(statuses)
	if count == 0 {
		return paging.EmptyResponse(), nil
	}

	var (
		items          = make([]interface{}, count)
		nextMaxIDValue = statuses[count-1].GetID()
		prevMinIDValue = statuses[0].GetID()
	)

	for i := range statuses {
		items[i] = statuses[i]
	}

	return paging.PackageResponse(paging.ResponseParams[string]{
		Items: items,
		Path:  "/api/v1/timelines/home",
		Next:  page.Next(nextMaxIDValue),
		Prev:  page.Prev(prevMinIDValue),
	}), nil
}
