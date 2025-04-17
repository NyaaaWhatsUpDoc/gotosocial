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

package mutes_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/id"
	"github.com/superseriousbusiness/gotosocial/internal/util"
)

type StatusMuteTestSuite struct {
	FilterStandardTestSuite
}

func (suite *StatusMuteTestSuite) TestMutedStatusAuthor() {
	ctx, cncl := context.WithCancel(context.Background())
	defer cncl()

	status := suite.testStatuses["admin_account_status_1"]
	requester := suite.testAccounts["local_account_1"]

	// Initially check if status is muted to the requester.
	muted, err := suite.filter.StatusMuted(ctx, requester, status)
	suite.NoError(err)
	suite.False(muted)

	// Insert new user mute targetting status author.
	err = suite.state.DB.PutMute(ctx, &gtsmodel.UserMute{
		ID:              id.NewULID(),
		AccountID:       requester.ID,
		TargetAccountID: status.AccountID,
		Notifications:   util.Ptr(false),
	})
	suite.NoError(err)

	// Check again if status is muted to the requester.
	muted, err = suite.filter.StatusMuted(ctx, requester, status)
	suite.NoError(err)
	suite.True(muted)

	// Though notifications should still be enabled for status to requester.
	muted, err = suite.filter.StatusNotificationsMuted(ctx, requester, status)
	suite.NoError(err)
	suite.False(muted)
}

func (suite *StatusMuteTestSuite) TestMutedReply() {
	ctx, cncl := context.WithCancel(context.Background())
	defer cncl()

	_ = ctx
}

func (suite *StatusMuteTestSuite) TestMutedBoost() {
	ctx, cncl := context.WithCancel(context.Background())
	defer cncl()

	_ = ctx
}

func TestStatusVisibleTestSuite(t *testing.T) {
	suite.Run(t, new(StatusMuteTestSuite))
}
