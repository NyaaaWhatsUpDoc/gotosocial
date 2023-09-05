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

package timelines

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apiutil "github.com/superseriousbusiness/gotosocial/internal/api/util"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/oauth"
	"github.com/superseriousbusiness/gotosocial/internal/paging"
)

// HomeTimelineGETHandler swagger:operation GET /api/v1/timelines/tag/{tag_name} tagTimeline
//
// See public statuses that use the given hashtag (case insensitive).
//
// The statuses will be returned in descending chronological order (newest first), with sequential IDs (bigger = newer).
//
// The returned Link header can be used to generate the previous and next queries when scrolling up or down a timeline.
//
// Example:
//
// ```
// <https://example.org/api/v1/timelines/tag/example?limit=20&max_id=01FC3GSQ8A3MMJ43BPZSGEG29M>; rel="next", <https://example.org/api/v1/timelines/tag/example?limit=20&min_id=01FC3KJW2GYXSDDRA6RWNDM46M>; rel="prev"
// ````
//
//	---
//	tags:
//	- timelines
//
//	produces:
//	- application/json
//
//	parameters:
//	-
//		name: max_id
//		type: string
//		description: >-
//			Return only statuses *OLDER* than the given max status ID.
//			The status with the specified ID will not be included in the response.
//		in: query
//		required: false
//	-
//		name: since_id
//		type: string
//		description: >-
//			Return only statuses *newer* than the given since status ID.
//			The status with the specified ID will not be included in the response.
//		in: query
//	-
//		name: min_id
//		type: string
//		description: >-
//			Return only statuses *immediately newer* than the given since status ID.
//			The status with the specified ID will not be included in the response.
//		in: query
//		required: false
//	-
//		name: limit
//		type: integer
//		description: Number of statuses to return.
//		default: 20
//		minimum: 1
//		maximum: 40
//		in: query
//		required: false
//
//	security:
//	- OAuth2 Bearer:
//		- read:statuses
//
//	responses:
//		'200':
//			name: statuses
//			description: Array of statuses.
//			schema:
//				type: array
//				items:
//					"$ref": "#/definitions/status"
//			headers:
//				Link:
//					type: string
//					description: Links to the next and previous queries.
//		'401':
//			description: unauthorized
//		'400':
//			description: bad request
func (m *Module) TagTimelineGETHandler(c *gin.Context) {
	authed, err := oauth.Authed(c, true, true, true, true)
	if err != nil {
		apiutil.ErrorHandler(c, gtserror.NewErrorUnauthorized(err, err.Error()), m.processor.InstanceGetV1)
		return
	}

	if _, err := apiutil.NegotiateAccept(c, apiutil.JSONAcceptHeaders...); err != nil {
		apiutil.ErrorHandler(c, gtserror.NewErrorNotAcceptable(err, err.Error()), m.processor.InstanceGetV1)
		return
	}

	limit, errWithCode := apiutil.ParseLimit(c.Query(apiutil.LimitKey), 20, 40, 1)
	if errWithCode != nil {
		apiutil.ErrorHandler(c, errWithCode, m.processor.InstanceGetV1)
		return
	}

	tagName, errWithCode := apiutil.ParseTagName(c.Param(apiutil.TagNameKey))
	if errWithCode != nil {
		apiutil.ErrorHandler(c, errWithCode, m.processor.InstanceGetV1)
		return
	}

	resp, errWithCode := m.processor.Timeline().TagTimelineGet(
		c.Request.Context(),
		authed.Account,
		tagName,
		&paging.Page[string]{
			// Query provided paging values (note they may be empty).
			Min:   paging.MinID(c.Query("min_id"), c.Query("since_id")),
			Max:   paging.MaxID(c.Query("max_id")),
			Limit: limit,
		},
	)
	if errWithCode != nil {
		apiutil.ErrorHandler(c, errWithCode, m.processor.InstanceGetV1)
		return
	}

	if resp.LinkHeader != "" {
		// Add paging link header if found.
		c.Header("Link", resp.LinkHeader)
	}

	c.JSON(http.StatusOK, resp.Items)
}
