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
	"github.com/superseriousbusiness/gotosocial/internal/paging"
	"github.com/superseriousbusiness/gotosocial/internal/text"
)

// TagTimelineGet gets a pageable timeline for the given
// tagName and given paging parameters. It will ensure
// that each status in the timeline is actually visible
// to requestingAcct before returning it.
func (p *Processor) TagTimelineGet(
	ctx context.Context,
	requestingAcct *gtsmodel.Account,
	tagName string,
	page *paging.Page[string],
) (*apimodel.PageableResponse, gtserror.WithCode) {
	tag, errWithCode := p.getTag(ctx, tagName)
	if errWithCode != nil {
		return nil, errWithCode
	}

	if tag == nil || !*tag.Useable || !*tag.Listable {
		// Obey mastodon API by returning 404 for this.
		err := fmt.Errorf("tag was not found, or not useable/listable on this instance")
		return nil, gtserror.NewErrorNotFound(err, err.Error())
	}

	statuses, err := p.state.DB.GetTagTimeline(ctx, tag.ID, page)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		err = gtserror.Newf("db error getting statuses: %w", err)
		return nil, gtserror.NewErrorInternalError(err)
	}

	return p.packageTagResponse(
		ctx,
		requestingAcct,
		statuses,
		// Use API URL for tag.
		"/api/v1/timelines/tag/"+tagName,
		page,
	)
}

func (p *Processor) getTag(ctx context.Context, tagName string) (*gtsmodel.Tag, gtserror.WithCode) {
	// Normalize + validate tag name.
	tagNameNormal, ok := text.NormalizeHashtag(tagName)
	if !ok {
		err := gtserror.Newf("string '%s' could not be normalized to a valid hashtag", tagName)
		return nil, gtserror.NewErrorBadRequest(err, err.Error())
	}

	// Ensure we have tag with this name in the db.
	tag, err := p.state.DB.GetTagByName(ctx, tagNameNormal)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		// Real db error.
		err = gtserror.Newf("db error getting tag by name: %w", err)
		return nil, gtserror.NewErrorInternalError(err)
	}

	return tag, nil
}

func (p *Processor) packageTagResponse(
	ctx context.Context,
	requestingAcct *gtsmodel.Account,
	statuses []*gtsmodel.Status,
	requestPath string,
	page *paging.Page[string],
) (*apimodel.PageableResponse, gtserror.WithCode) {
	count := len(statuses)
	if count == 0 {
		return paging.EmptyResponse(), nil
	}

	var (
		items = make([]interface{}, 0, count)

		// Set next + prev values before filtering and API
		// converting, so caller can still page properly.
		nextMaxIDValue = statuses[count-1].ID
		prevMinIDValue = statuses[0].ID
	)

	for _, s := range statuses {
		timelineable, err := p.filter.StatusTagTimelineable(ctx, requestingAcct, s)
		if err != nil {
			log.Errorf(ctx, "error checking status visibility: %v", err)
			continue
		}

		if !timelineable {
			continue
		}

		apiStatus, err := p.tc.StatusToAPIStatus(ctx, s, requestingAcct)
		if err != nil {
			log.Errorf(ctx, "error converting to api status: %v", err)
			continue
		}

		items = append(items, apiStatus)
	}

	return paging.PackageResponse(paging.ResponseParams[string]{
		Items: items,
		Path:  requestPath,
		Next:  page.Next(nextMaxIDValue),
		Prev:  page.Prev(prevMinIDValue),
	}), nil
}
