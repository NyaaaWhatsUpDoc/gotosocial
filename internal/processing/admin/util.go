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

package admin

import (
	"context"
	"errors"
	"time"

	apimodel "github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/paging"
)

// apiDomainBlock is a cheeky shortcut for returning
// the API version of the given domainBlock, or an
// appropriate error if something goes wrong.
func (p *Processor) apiDomainBlock(
	ctx context.Context,
	domainBlock *gtsmodel.DomainBlock,
) (*apimodel.DomainBlock, gtserror.WithCode) {
	apiDomainBlock, err := p.tc.DomainBlockToAPIDomainBlock(ctx, domainBlock, false)
	if err != nil {
		err = gtserror.Newf("error converting domain block for %s to api model : %w", domainBlock.Domain, err)
		return nil, gtserror.NewErrorInternalError(err)
	}

	return apiDomainBlock, nil
}

// stubbifyInstance renders the given instance as a stub,
// removing most information from it and marking it as
// suspended.
//
// For caller's convenience, this function returns the db
// names of all columns that are updated by it.
func stubbifyInstance(instance *gtsmodel.Instance, domainBlockID string) []string {
	instance.Title = ""
	instance.SuspendedAt = time.Now()
	instance.DomainBlockID = domainBlockID
	instance.ShortDescription = ""
	instance.Description = ""
	instance.Terms = ""
	instance.ContactEmail = ""
	instance.ContactAccountUsername = ""
	instance.ContactAccountID = ""
	instance.Version = ""

	return []string{
		"title",
		"suspended_at",
		"domain_block_id",
		"short_description",
		"description",
		"terms",
		"contact_email",
		"contact_account_username",
		"contact_account_id",
		"version",
	}
}

// rangeDomainAccounts iterates through all accounts
// originating from the given domain, and calls the
// provided range function on each account.
//
// If an error is returned while selecting accounts,
// the loop will stop and return the error.
func (p *Processor) rangeDomainAccounts(
	ctx context.Context,
	domain string,
	rangeF func(*gtsmodel.Account),
) error {
	// page for iterative account fetching
	// from a previous maximum account ID.
	page := paging.Page[string]{
		Limit: 50, // to prevent spiking mem/cpu
	}

	for {
		// Get (next) page of accounts.
		accounts, err := p.state.DB.GetInstanceAccounts(ctx, domain, &page)
		if err != nil && !errors.Is(err, db.ErrNoEntries) {
			// Real db error.
			return gtserror.Newf("db error getting instance accounts: %w", err)
		}

		if len(accounts) == 0 {
			// No accounts left, we're done.
			return nil
		}

		// Use last attachment as next page maxID value.
		page.Max.Value = accounts[len(accounts)-1].ID

		// Call provided range function.
		for _, account := range accounts {
			rangeF(account)
		}
	}
}
