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

	"github.com/superseriousbusiness/gotosocial/internal/cache"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtscontext"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// StatusMuted returns whether given target status is muted for requester in the context of timeline visibility.
// Note this function returns whether the mute value comes with an expiry, useful to know if a cache relies on this value.
func (f *Filter) StatusMuted(ctx context.Context, requester *gtsmodel.Account, status *gtsmodel.Status) (muted bool, withExpiry bool, err error) {
	details, now, err := f.getStatusMuteDetails(ctx, requester, status)
	if err != nil {
		return false, false, gtserror.Newf("error getting status mute details: %w", err)
	}

	// Check if not muted.
	if !details.Mute {
		return false, false, nil
	}

	// Check if mute has no expiry.
	if details.MuteExpiry.IsZero() {
		return true, false, nil
	}

	// Check if the mute has expired.
	muted = details.MuteExpiry.After(now)
	withExpiry = true
	return
}

// StatusNotificationsMuted returns whether notifications are muted for requester when regarding given target status.
// Note this function returns whether the mute value comes with an expiry, useful to know if a cache relies on this value.
func (f *Filter) StatusNotificationsMuted(ctx context.Context, requester *gtsmodel.Account, status *gtsmodel.Status) (muted bool, withExpiry bool, err error) {
	details, now, err := f.getStatusMuteDetails(ctx, requester, status)
	if err != nil {
		return false, false, gtserror.Newf("error getting status mute details: %w", err)
	}

	// Check if not notif muted.
	if !details.Notifications {
		return false, false, nil
	}

	// Check if notif mute has no expiry.
	if details.NotificationExpiry.IsZero() {
		return true, false, nil
	}

	// Check if the notification mute has expired.
	muted = details.NotificationExpiry.After(now)
	withExpiry = true
	return
}

// getStatusMuteDetails returns the combined mute details
func (f *Filter) getStatusMuteDetails(ctx context.Context, requester *gtsmodel.Account, status *gtsmodel.Status) (*cache.CachedMute, time.Time, error) {
	const mtype = cache.MuteTypeStatus

	if requester == nil {
		// Without auth, there will be no possible
		// mute to exist. Always return as 'unmuted'.
		return &cache.CachedMute{}, time.Time{}, nil
	}

	// Get current time.
	now := time.Now()

	// Using cache loader callback, attempt to load cache mute details about a given status.
	details, err := f.state.Caches.Mutes.LoadOne("Type,RequesterID,ItemID", func() (*cache.CachedMute, error) {
		var notifs, muted bool
		var notifExpiry time.Time
		var muteExpiry time.Time

		// Look for a mute by requester against thread.
		threadMute, err := f.getStatusThreadMute(ctx,
			requester,
			status,
		)
		if err != nil {
			return nil, err
		}

		// Mute notifs on thread mute.
		notifs = (threadMute != nil)

		// Look for mutes against related status accounts
		// by requester (e.g. author, mention targets etc).
		userMutes, err := f.getStatusRelatedUserMutes(ctx,
			requester,
			status,
		)
		if err != nil {
			return nil, err
		}

		for _, mute := range userMutes {
			// Check for expiry data given.
			if !mute.ExpiresAt.IsZero() {

				if mute.ExpiresAt.Before(now) {
					// Don't consider expired
					// mute in calculations.
					continue
				}

				// Update mute expiry time if later.
				if mute.ExpiresAt.After(muteExpiry) {
					muteExpiry = mute.ExpiresAt
				}
			}

			// This is non-expired
			// mute, mark as muted!
			muted = true

			// Check if notifs muted.
			if *mute.Notifications {

				// Set notif
				// mute flag.
				notifs = true

				// Update notif expiry time if later.
				if mute.ExpiresAt.After(notifExpiry) {
					notifExpiry = mute.ExpiresAt
				}
			}
		}

		return &cache.CachedMute{
			ItemID:             status.ID,
			RequesterID:        requester.ID,
			Type:               mtype,
			Mute:               muted,
			MuteExpiry:         muteExpiry,
			Notifications:      notifs,
			NotificationExpiry: notifExpiry,
		}, nil
	}, mtype, requester.ID, status.ID)
	if err != nil {
		return nil, now, err
	}

	return details, now, nil
}

func (f *Filter) getStatusThreadMute(ctx context.Context, requester *gtsmodel.Account, status *gtsmodel.Status) (*gtsmodel.ThreadMute, error) {
	if status.ThreadID == "" {
		// Status is not threaded,
		// mute won't exist for it!
		return nil, nil
	}

	// Look for a stored mute from account against thread.
	mute, err := f.state.DB.GetThreadMutedByAccount(ctx,
		status.ThreadID,
		requester.ID,
	)
	if err != nil {
		return nil, gtserror.Newf("db error checking thread mute: %w", err)
	}

	return mute, nil
}

func (f *Filter) getStatusRelatedUserMutes(ctx context.Context, requester *gtsmodel.Account, status *gtsmodel.Status) ([]*gtsmodel.UserMute, error) {
	if status.AccountID == requester.ID {
		// Status is by requester, we don't take
		// into account related attached user mutes.
		return nil, nil
	}

	// Preallocate a slice of worst possible case no. user mutes.
	mutes := make([]*gtsmodel.UserMute, 0, 1+len(status.Mentions))

	// Look for mute against author.
	mute, err := f.state.DB.GetMute(
		gtscontext.SetBarebones(ctx),
		requester.ID,
		status.AccountID,
	)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		return nil, gtserror.Newf("db error getting status author mute: %w", err)
	}

	if mute != nil {
		// Append author mute to total.
		mutes = append(mutes, mute)
	}

	for _, mention := range status.Mentions {
		// Look for mute against any target mentions.
		if mention.TargetAccountID != requester.ID {

			// Look for mute against target.
			mute, err := f.state.DB.GetMute(
				gtscontext.SetBarebones(ctx),
				requester.ID,
				mention.TargetAccountID,
			)
			if err != nil && !errors.Is(err, db.ErrNoEntries) {
				return nil, gtserror.Newf("db error getting mention target mute: %w", err)
			}

			if mute != nil {
				// Append target mute to total.
				mutes = append(mutes, mute)
			}
		}
	}

	return mutes, nil
}
