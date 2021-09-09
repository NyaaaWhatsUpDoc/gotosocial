/*
   GoToSocial
   Copyright (C) 2021 GoToSocial Authors admin@gotosocial.org

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

package visibility

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

func (f *filter) StatusPublictimelineable(ctx context.Context, targetStatus *gtsmodel.Status, timelineOwnerAccount *gtsmodel.Account) (bool, error) {
	l := f.log.WithFields(logrus.Fields{
		"func":     "StatusPublictimelineable",
		"statusID": targetStatus.ID,
	})

	// Boosts should not be visible on federated timeline
	if targetStatus.BoostOfID != "" {
		return false, nil
	}

	// If reply isn't part of a single-author thread, don't timeline
	if targetStatus.InReplyToID != "" && len(targetStatus.MentionIDs) > 0 {
		return false, nil
	}

	// status owner should always be able to see their own status in their timeline so we can return early if this is the case
	if timelineOwnerAccount != nil && targetStatus.AccountID == timelineOwnerAccount.ID {
		return true, nil
	}

	// Perform a regular visibility check for the status
	visible, err := f.StatusVisible(ctx, targetStatus, timelineOwnerAccount)
	if !visible && err == nil {
		l.Debug("status is not publicTimelinable because it's not visible to requester")
	}
	return visible, err
}
