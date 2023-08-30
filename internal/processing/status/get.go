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

package status

import (
	"context"
	"sort"

	apimodel "github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// Get gets the given status, taking account of privacy settings and blocks etc.
func (p *Processor) Get(ctx context.Context, requestingAccount *gtsmodel.Account, targetStatusID string) (*apimodel.Status, gtserror.WithCode) {
	targetStatus, errWithCode := p.c.GetVisibleTargetStatus(ctx, requestingAccount, targetStatusID)
	if errWithCode != nil {
		return nil, errWithCode
	}
	return p.c.GetAPIStatus(ctx, requestingAccount, targetStatus)
}

// ContextGet returns the context (previous and following posts) from the given status ID.
func (p *Processor) ContextGet(ctx context.Context, requestingAccount *gtsmodel.Account, targetStatusID string) (*apimodel.Context, gtserror.WithCode) {
	targetStatus, errWithCode := p.c.GetVisibleTargetStatus(ctx, requestingAccount, targetStatusID)
	if errWithCode != nil {
		return nil, errWithCode
	}

	// Fetch parent statuses for the target status.
	parents, err := p.state.DB.GetStatusParents(ctx, targetStatus, false)
	if err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}

	// Ensure the status parents sorted by ID.
	sort.Slice(parents, func(i int, j int) bool {
		return parents[i].ID < parents[j].ID
	})

	// Convert parent statuses to frontend API models and filter for visibility to requester.
	ancestors := p.c.GetVisibleAPIStatuses(ctx, requestingAccount, func(i int) *gtsmodel.Status {
		return parents[i]
	}, len(parents))

	// Fetch child statuses for the target status.
	children, err := p.state.DB.GetStatusChildren(ctx, targetStatus, false, "")
	if err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}

	// Ensure the status children sorted by ID.
	sort.Slice(children, func(i int, j int) bool {
		return children[i].ID < children[j].ID
	})

	// Convert child statuses to frontend API models and filter for visibility to requester.
	descendents := p.c.GetVisibleAPIStatuses(ctx, requestingAccount, func(i int) *gtsmodel.Status {
		return children[i]
	}, len(children))

	return &apimodel.Context{
		Ancestors:   ancestors,
		Descendants: descendents,
	}, nil
}
