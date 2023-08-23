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

package paging

import (
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

// Pager provides a means of paging serialized IDs,
// using the terminology of our API endpoint queries.
type Pager struct {
	// SinceID will limit the returned
	// page of IDs to contain newer than
	// since ID (excluding it). Result
	// will be returned DESCENDING.
	SinceID string

	// MinID will limit the returned
	// page of IDs to contain newer than
	// min ID (excluding it). Result
	// will be returned ASCENDING.
	MinID string

	// MaxID will limit the returned
	// page of IDs to contain older
	// than (excluding) this max ID.
	MaxID string

	// Limit will limit the returned
	// page of IDs to at most 'limit'.
	Limit int
}

func NextPage(maxID string, limit int) *Pager {
	return &Pager{MaxID: maxID, Limit: limit}
}

func PrevPage(minID string, limit int) *Pager {
	return &Pager{MinID: minID, Limit: limit}
}

// Next creates a new Pager instance for the next returnable page,
// using given max ID value. This preserves original limit value.
func (p *Pager) Next(maxID string) *Pager {
	if maxID == "" {
		return nil // no paging to do
	}

	if p == nil {
		// No previous page given.
		return &Pager{MaxID: maxID}
	}

	// Create new page.
	p2 := new(Pager)

	// Set original limit.
	p2.Limit = p.Limit

	// Set "max_id".
	p2.MaxID = maxID

	return p2
}

// Prev creates a new Pager instance for the prev returnable page, using
// given min ID value. This preserves original limit value and min ID keying.
func (p *Pager) Prev(minID string) *Pager {
	if minID == "" {
		// no paging.
		return nil
	}

	if p == nil {
		// No previous page given.
		return &Pager{MinID: minID}
	}

	// Create new page.
	p2 := new(Pager)

	// Set original limit.
	p2.Limit = p.Limit

	// Set minID based on prev
	// which min type was used.
	if p.SinceID != "" {
		p2.SinceID = minID
	} else {
		p2.MinID = minID
	}

	return p2
}

// Page will page the given slice of GoToSocial IDs according
// to the receiving Pager's SinceID, MinID, MaxID and Limits.
// NOTE THE INPUT SLICE MUST BE SORTED IN ASCENDING ORDER
// (I.E. OLDEST ITEMS AT LOWEST INDICES, NEWER AT HIGHER).
func (p *Pager) PageAsc(ids []string) []string {
	if p == nil {
		// no paging.
		return ids
	}

	var asc bool

	if p.SinceID != "" {
		// If a sinceID is given, we
		// page down i.e. descending.
		asc = false

		for i := 0; i < len(ids); i++ {
			if ids[i] == p.SinceID {
				// Hit the boundary.
				// Reslice to be:
				// "from here"
				ids = ids[i+1:]
				break
			}
		}
	} else if p.MinID != "" {
		// We only support minID if
		// no sinceID is provided.
		//
		// If a minID is given, we
		// page up, i.e. ascending.
		asc = true

		for i := 0; i < len(ids); i++ {
			if ids[i] == p.MinID {
				// Hit the boundary.
				// Reslice to be:
				// "from here"
				ids = ids[i+1:]
				break
			}
		}
	}

	if p.MaxID != "" {
		for i := 0; i < len(ids); i++ {
			if ids[i] == p.MaxID {
				// Hit the boundary.
				// Reslice to be:
				// "up to here"
				ids = ids[:i]
				break
			}
		}
	}

	if !asc && len(ids) > 1 {
		var (
			// Start at front.
			i = 0

			// Start at back.
			j = len(ids) - 1
		)

		// Clone input IDs before
		// we perform modifications.
		ids = slices.Clone(ids)

		for i < j {
			// Swap i,j index values in slice.
			ids[i], ids[j] = ids[j], ids[i]

			// incr + decr,
			// looping until
			// they meet in
			// the middle.
			i++
			j--
		}
	}

	if p.Limit > 0 && p.Limit < len(ids) {
		// Reslice IDs to given limit.
		ids = ids[:p.Limit]
	}

	return ids
}

// Page will page the given slice of GoToSocial IDs according
// to the receiving Pager's SinceID, MinID, MaxID and Limits.
// NOTE THE INPUT SLICE MUST BE SORTED IN ASCENDING ORDER.
// (I.E. NEWEST ITEMS AT LOWEST INDICES, OLDER AT HIGHER).
func (p *Pager) PageDesc(ids []string) []string {
	if p == nil {
		// no paging.
		return ids
	}

	var asc bool

	if p.MaxID != "" {
		for i := 0; i < len(ids); i++ {
			if ids[i] == p.MaxID {
				// Hit the boundary.
				// Reslice to be:
				// "from here"
				ids = ids[i+1:]
				break
			}
		}
	}

	if p.SinceID != "" {
		// If a sinceID is given, we
		// page down i.e. descending.
		asc = false

		for i := 0; i < len(ids); i++ {
			if ids[i] == p.SinceID {
				// Hit the boundary.
				// Reslice to be:
				// "up to here"
				ids = ids[:i]
				break
			}
		}
	} else if p.MinID != "" {
		// We only support minID if
		// no sinceID is provided.
		//
		// If a minID is given, we
		// page up, i.e. ascending.
		asc = true

		for i := 0; i < len(ids); i++ {
			if ids[i] == p.MinID {
				// Hit the boundary.
				// Reslice to be:
				// "up to here"
				ids = ids[:i]
				break
			}
		}
	}

	if asc && len(ids) > 1 {
		var (
			// Start at front.
			i = 0

			// Start at back.
			j = len(ids) - 1
		)

		// Clone input IDs before
		// we perform modifications.
		ids = slices.Clone(ids)

		for i < j {
			// Swap i,j index values in slice.
			ids[i], ids[j] = ids[j], ids[i]

			// incr + decr,
			// looping until
			// they meet in
			// the middle.
			i++
			j--
		}
	}

	if p.Limit > 0 && p.Limit < len(ids) {
		// Reslice IDs to given limit.
		ids = ids[:p.Limit]
	}

	return ids
}

// NextLink will build a next page link string to use for given host configuration and extra query parameters.
func (p *Pager) NextLink(proto, host, path string, queryParams []string) string {
	if p == nil {
		return ""
	}

	var (
		maxIDKey = "max_id"
		maxIDVal = p.MaxID
	)

	return buildLink(
		proto,
		host,
		path,
		maxIDKey,
		maxIDVal,
		p.Limit,
		queryParams,
	)
}

// PrevLink will build a previous page link string to use for given host configuration and extra query parameters.
func (p *Pager) PrevLink(proto, host, path string, queryParams []string) string {
	if p == nil {
		return ""
	}

	var (
		minIDKey string
		minIDVal string
	)

	switch {
	// Use "min_id" minimum
	case p.MinID != "":
		minIDKey = "min_id"
		minIDVal = p.MinID

	// Use "since_id" minimum
	case p.SinceID != "":
		minIDKey = "since_id"
		minIDVal = p.SinceID
	}

	return buildLink(
		proto,
		host,
		path,
		minIDKey,
		minIDVal,
		p.Limit,
		queryParams,
	)
}

// buildLink will build a next / previous link for use in a pageable response.
func buildLink(proto, host, path, idKey, idVal string, limit int, queryParams []string) string {
	if idVal == "" {
		// No paging to do.
		return ""
	}

	// Append the `idKey=idValue` to query params.
	queryParams = append(queryParams, idKey+"="+idVal)

	if limit > 0 {
		// Build limit query parameter.
		param := "limit=" + strconv.Itoa(limit)

		// Append `limit=$value` query parameter.
		queryParams = append(queryParams, param)
	}

	// Build URL string.
	return (&url.URL{
		Scheme:   proto,
		Host:     host,
		Path:     path,
		RawQuery: strings.Join(queryParams, "&"),
	}).String()
}
