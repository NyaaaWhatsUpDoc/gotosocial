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

// Page ...
type Page[T comparable] struct {
	// Min ...
	Min Boundary[T]

	// Max ...
	Max Boundary[T]

	// Limit will limit the returned
	// page of items to at most 'limit'.
	Limit int
}

// GetMin is a small helper function to return minimum boundary value or false (checking for nil page and zero minimum).
func (p *Page[T]) GetMin() (T, bool) {
	if p == nil || zero(p.Min.Value) {
		var z T // zero
		return z, false
	}
	return p.Min.Value, true
}

// GetMax is a small helper function to return maximum boundary value or false (checking for nil page and zero maximum).
func (p *Page[T]) GetMax() (T, bool) {
	if p == nil || zero(p.Max.Value) {
		var z T // zero
		return z, false
	}
	return p.Max.Value, true
}

// GetLimit is a small helper function to return limit or false (checking for nil page and zero limit).
func (p *Page[T]) GetLimit() (int, bool) {
	if p == nil || p.Limit <= 0 {
		return 0, false
	}
	return p.Limit, true
}

// GetOrder is a small helper function to return page ordering (checking for nil page).
func (p *Page[T]) GetOrder() (Order, bool) {
	if p == nil {
		return 0, false
	}
	return p.order(), true
}

// order returns this Page's ordering.
func (p *Page[T]) order() Order {
	switch {
	// we give preference to the
	// minimum boundary, as that's
	// usually where order comes
	// from, e.g. since_id vs min_id.
	case !p.Min.Order.None():
		return p.Min.Order
	case !p.Max.Order.None():
		return p.Max.Order
	default:
		return 0
	}
}

// PageAsc will page the given slice of input according
// to the receiving Page's minimum, maximum and limit.
// NOTE THE INPUT SLICE MUST BE SORTED IN ASCENDING ORDER
// (I.E. OLDEST ITEMS AT LOWEST INDICES, NEWER AT HIGHER).
func (p *Page[T]) PageAsc(in []T) []T {
	if p == nil {
		// no paging.
		return nil
	}

	// Look for min boundary in input, reslice
	// from (but not including) minimum value.
	if minIdx := p.Min.Find(in); minIdx != -1 {
		in = in[minIdx+1:]
	}

	// Look for max boundary in input, reslice
	// up-to (but not including) maximum value.
	if maxIdx := p.Max.Find(in); maxIdx != -1 {
		in = in[:maxIdx]
	}

	// Check if either require descending order,
	// bearing in mind that 'in' is ascending.
	needDesc := p.order() == OrderDescending

	if needDesc && len(in) > 1 {
		var (
			// Start at front.
			i = 0

			// Start at back.
			j = len(in) - 1
		)

		// Clone input before
		// any modifications.
		in = slices.Clone(in)

		for i < j {
			// Swap i,j index values in slice.
			in[i], in[j] = in[j], in[i]

			// incr + decr,
			// looping until
			// they meet in
			// the middle.
			i++
			j--
		}
	}

	if p.Limit > 0 && p.Limit < len(in) {
		// Reslice input to limit.
		in = in[:p.Limit]
	}

	return in
}

// PageDesc will page the given slice of input according
// to the receiving Page's minimum, maximum and limit.
// NOTE THE INPUT SLICE MUST BE SORTED IN ASCENDING ORDER.
// (I.E. NEWEST ITEMS AT LOWEST INDICES, OLDER AT HIGHER).
func (p *Page[T]) PageDesc(in []T) []T {
	if p == nil {
		// no paging.
		return nil
	}

	// Look for max boundary in input, reslice
	// from (but not including) maximum value.
	if maxIdx := p.Max.Find(in); maxIdx != -1 {
		in = in[maxIdx+1:]
	}

	// Look for min boundary in input, reslice
	// up-to (but not including) minimum value.
	if minIdx := p.Min.Find(in); minIdx != -1 {
		in = in[:minIdx]
	}

	// Check if either require ascending order,
	// bearing in mind that 'in' is descending.
	needAsc := p.order() == OrderAscending

	if needAsc && len(in) > 1 {
		var (
			// Start at front.
			i = 0

			// Start at back.
			j = len(in) - 1
		)

		// Clone input before
		// any modifications.
		in = slices.Clone(in)

		for i < j {
			// Swap i,j index values in slice.
			in[i], in[j] = in[j], in[i]

			// incr + decr,
			// looping until
			// they meet in
			// the middle.
			i++
			j--
		}
	}

	if p.Limit > 0 && p.Limit < len(in) {
		// Reslice input to limit.
		in = in[:p.Limit]
	}

	return in
}

// Next creates a new instance for the next returnable page, using
// given max value. This preserves original limit and max key name.
func (p *Page[T]) Next(max T) *Page[T] {
	if p == nil || zero(max) {
		// no paging.
		return nil
	}

	// Create new page.
	p2 := new(Page[T])

	// Set original limit.
	p2.Limit = p.Limit

	// Create new from old.
	p2.Max = p.Max.New(max)

	return p2
}

// Prev creates a new instance for the prev returnable page, using
// given min value. This preserves original limit and min key name.
func (p *Page[T]) Prev(min T) *Page[T] {
	if p == nil || zero(min) {
		// no paging.
		return nil
	}

	// Create new page.
	p2 := new(Page[T])

	// Set original limit.
	p2.Limit = p.Limit

	// Create new from old.
	p2.Min = p.Min.New(min)

	return p2
}

// ToLink builds a URL link for given endpoint information and extra query parameters,
// appending this Page's minimum / maximum boundaries and available limit (if any).
func (p *Page[T]) ToLink(proto, host, path string, queryParams []string) string {
	if p == nil {
		// no paging.
		return ""
	}

	// Check length before
	// adding boundary params.
	old := len(queryParams)

	if minParam := p.Min.Query(); minParam != "" {
		// A page-minimum query parameter is available.
		queryParams = append(queryParams, minParam)
	}

	if maxParam := p.Max.Query(); maxParam != "" {
		// A page-maximum query parameter is available.
		queryParams = append(queryParams, maxParam)
	}

	if len(queryParams) == old {
		// No page boundaries.
		return ""
	}

	if p.Limit > 0 {
		// Build limit key-value query parameter.
		param := "limit=" + strconv.Itoa(p.Limit)

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
