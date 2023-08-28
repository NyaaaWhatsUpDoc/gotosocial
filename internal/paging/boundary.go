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

import "fmt"

// MinID ...
func MinID(minID, sinceID string) Boundary[string] {
	switch {
	case sinceID != "":
		return Boundary[string]{
			Name:  "since_id",
			Value: sinceID,

			// with "since_id", return
			// as descending order items.
			Order: OrderDescending,
		}
	default:
		// minID is default min type.
		return Boundary[string]{
			Name:  "min_id",
			Value: minID,

			// with "min_id", return
			// as ascending order items.
			Order: OrderAscending,
		}
	}
}

// MaxID returns a boundary with given maximum
// ID value, and the "max_id" query key set.
func MaxID(maxID string) Boundary[string] {
	return Boundary[string]{
		Name:  "max_id",
		Value: maxID,

		// no order specified
		Order: 0,
	}
}

// MinShortcodeDomain returns a boundary with the given minimum emoji
// shortcode@domain, and the "min_shortcode_domain" query key set.
func MinShortcodeDomain(min string) Boundary[string] {
	return Boundary[string]{
		Name:  "min_shortcode_domain",
		Value: min,
		Order: OrderDescending,
	}
}

// MaxShortcodeDomain returns a boundary with the given maximum emoji
// shortcode@domain, and the "max_shortcode_domain" query key set.
func MaxShortcodeDomain(max string) Boundary[string] {
	return Boundary[string]{
		Name:  "max_shortcode_domain",
		Value: max,
	}
}

// Boundary ...
type Boundary[T comparable] struct {
	Name  string
	Value T
	Order Order
}

// New ...
func (b Boundary[T]) New(value T) Boundary[T] {
	return Boundary[T]{
		Name:  b.Name,
		Value: value,
		Order: b.Order,
	}
}

// Find ...
func (b Boundary[T]) Find(in []T) int {
	if zero(b.Value) {
		return -1
	}
	for i := range in {
		if in[i] == b.Value {
			return i
		}
	}
	return -1
}

// Query ...
func (b Boundary[T]) Query() string {
	switch {
	case !zero(b.Value):
		return fmt.Sprintf("%s=%v", b.Name, b.Value)
	default:
		return ""
	}
}

// Update ...
func (b *Boundary[T]) Update(value T) {
	b.Value = value
}
