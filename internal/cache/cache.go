/*
   GoToSocial
   Copyright (C) 2021-2022 GoToSocial Authors admin@gotosocial.org

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package cache

import (
	"time"

	"codeberg.org/gruf/go-cache"
)

// Cache defines an in-memory cache that is safe to be wiped when the application is restarted
type Cache cache.Cache

// New returns a new in-memory cache.
func New() Cache {
	c := cache.New()
	if !c.Stop() || !c.Start(time.Second*30) {
		panic("failed starting cache")
	}
	return c
}
