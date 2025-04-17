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

package cache

import (
	"time"

	"codeberg.org/gruf/go-structr"
	"github.com/superseriousbusiness/gotosocial/internal/config"
	"github.com/superseriousbusiness/gotosocial/internal/log"
)

type MutesCache struct {
	StructCache[*CachedMute]
}

func (c *Caches) initMutes() {
	// Calculate maximum cache size.
	cap := calculateResultCacheMax(
		sizeofVisibility(), // model in-mem size.
		config.GetCacheMutesMemRatio(),
	)

	log.Infof(nil, "Mutes cache size = %d", cap)

	copyF := func(m1 *CachedMute) *CachedMute {
		m2 := new(CachedMute)
		*m2 = *m1
		return m2
	}

	c.Mutes.Init(structr.CacheConfig[*CachedMute]{
		Indices: []structr.IndexConfig{
			{Fields: "ItemID", Multiple: true},
			{Fields: "RequesterID", Multiple: true},
			{Fields: "RequesterID,ItemID"},
		},
		MaxSize:   cap,
		IgnoreErr: ignoreErrors,
		Copy:      copyF,
	})
}

// CachedMute ...
type CachedMute struct {

	// ItemID is the ID of the item
	// in question (status / account).
	ItemID string

	// RequesterID is the ID of the requesting
	// account for this user mute lookup.
	RequesterID string

	// Mute indicates whether ItemID
	// is muted by RequesterID.
	Mute bool

	// MuteExpiry stores the time at which
	// (if any) the stored mute value expires.
	MuteExpiry time.Time

	// Notifications indicates whether
	// this mute should prevent notifications
	// being shown for ItemID to RequesterID.
	Notifications bool

	// NotificationExpiry stores the time at which
	// (if any) the stored notification value expires.
	NotificationExpiry time.Time
}

// MuteExpired returns whether the mute value has expired.
func (m *CachedMute) MuteExpired(now time.Time) bool {
	return !m.MuteExpiry.IsZero() && !m.MuteExpiry.After(now)
}

// NotificationExpired returns whether the notification mute value has expired.
func (m *CachedMute) NotificationExpired(now time.Time) bool {
	return !m.NotificationExpiry.IsZero() && !m.NotificationExpiry.After(now)
}
