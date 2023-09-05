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
	"errors"
	"fmt"

	"github.com/superseriousbusiness/gotosocial/internal/ap"
	apimodel "github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/id"
	"github.com/superseriousbusiness/gotosocial/internal/messages"
	"github.com/superseriousbusiness/gotosocial/internal/uris"
)

// FaveCreate adds a fave for the requestingAccount, targeting the given status (no-op if fave already exists).
func (p *Processor) FaveCreate(ctx context.Context, requestingAccount *gtsmodel.Account, targetStatusID string) (*apimodel.Status, gtserror.WithCode) {
	targetStatus, existingFave, errWithCode := p.getFaveTarget(ctx, requestingAccount, targetStatusID)
	if errWithCode != nil {
		return nil, errWithCode
	}

	if existingFave != nil {
		// Status is already faveed.
		return p.c.GetAPIStatus(ctx, requestingAccount, targetStatus)
	}

	// Create and store a new fave
	faveID := id.NewULID()
	gtsFave := &gtsmodel.StatusFave{
		ID:              faveID,
		AccountID:       requestingAccount.ID,
		Account:         requestingAccount,
		TargetAccountID: targetStatus.AccountID,
		TargetAccount:   targetStatus.Account,
		StatusID:        targetStatus.ID,
		Status:          targetStatus,
		URI:             uris.GenerateURIForLike(requestingAccount.Username, faveID),
	}

	if err := p.state.DB.PutStatusFave(ctx, gtsFave); err != nil {
		err = fmt.Errorf("FaveCreate: error putting fave in database: %w", err)
		return nil, gtserror.NewErrorInternalError(err)
	}

	// Process new status fave side effects.
	p.state.Workers.EnqueueClientAPI(ctx, messages.FromClientAPI{
		APObjectType:   ap.ActivityLike,
		APActivityType: ap.ActivityCreate,
		GTSModel:       gtsFave,
		OriginAccount:  requestingAccount,
		TargetAccount:  targetStatus.Account,
	})

	return p.c.GetAPIStatus(ctx, requestingAccount, targetStatus)
}

// FaveRemove removes a fave for the requesting account, targeting the given status (no-op if fave doesn't exist).
func (p *Processor) FaveRemove(ctx context.Context, requestingAccount *gtsmodel.Account, targetStatusID string) (*apimodel.Status, gtserror.WithCode) {
	targetStatus, existingFave, errWithCode := p.getFaveTarget(ctx, requestingAccount, targetStatusID)
	if errWithCode != nil {
		return nil, errWithCode
	}

	if existingFave == nil {
		// Status isn't faveed.
		return p.c.GetAPIStatus(ctx, requestingAccount, targetStatus)
	}

	// We have a fave to remove.
	if err := p.state.DB.DeleteStatusFaveByID(ctx, existingFave.ID); err != nil {
		err = fmt.Errorf("FaveRemove: error removing status fave: %w", err)
		return nil, gtserror.NewErrorInternalError(err)
	}

	// Process remove status fave side effects.
	p.state.Workers.EnqueueClientAPI(ctx, messages.FromClientAPI{
		APObjectType:   ap.ActivityLike,
		APActivityType: ap.ActivityUndo,
		GTSModel:       existingFave,
		OriginAccount:  requestingAccount,
		TargetAccount:  targetStatus.Account,
	})

	return p.c.GetAPIStatus(ctx, requestingAccount, targetStatus)
}

// FavedBy returns a slice of accounts that have liked the given status, filtered according to privacy settings.
func (p *Processor) FavedBy(ctx context.Context, requestingAccount *gtsmodel.Account, targetStatusID string) ([]*apimodel.Account, gtserror.WithCode) {
	targetStatus, errWithCode := p.c.GetVisibleTargetStatus(ctx, requestingAccount, targetStatusID)
	if errWithCode != nil {
		return nil, errWithCode
	}

	faves, err := p.state.DB.GetStatusFaves(ctx, targetStatus.ID)
	if err != nil {
		return nil, gtserror.NewErrorNotFound(fmt.Errorf("FavedBy: error seeing who faved status: %s", err))
	}

	// Return a filtered slice of public API account models (this catches and logs errors).
	return p.c.GetVisibleAPIAccounts(ctx, requestingAccount, func(i int) *gtsmodel.Account {
		return faves[i].Account
	}, len(faves)), nil
}

func (p *Processor) getFaveTarget(ctx context.Context, requestingAccount *gtsmodel.Account, targetStatusID string) (*gtsmodel.Status, *gtsmodel.StatusFave, gtserror.WithCode) {
	targetStatus, errWithCode := p.c.GetVisibleTargetStatus(ctx, requestingAccount, targetStatusID)
	if errWithCode != nil {
		return nil, nil, errWithCode
	}

	if !*targetStatus.Likeable {
		err := errors.New("status is not faveable")
		return nil, nil, gtserror.NewErrorForbidden(err, err.Error())
	}

	fave, err := p.state.DB.GetStatusFave(ctx, requestingAccount.ID, targetStatusID)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		err = fmt.Errorf("getFaveTarget: error checking existing fave: %w", err)
		return nil, nil, gtserror.NewErrorInternalError(err)
	}

	return targetStatus, fave, nil
}
