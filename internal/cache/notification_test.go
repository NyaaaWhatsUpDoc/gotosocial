package cache_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/superseriousbusiness/gotosocial/internal/cache"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/testrig"
)

type NotificationCacheTestSuite struct {
	suite.Suite
	data  map[string]*gtsmodel.Notification
	cache *cache.NotificationCache
}

func (suite *NotificationCacheTestSuite) SetupSuite() {
	suite.data = testrig.NewTestNotifications()
}

func (suite *NotificationCacheTestSuite) SetupTest() {
	suite.cache = cache.NewNotification()
}

func (suite *NotificationCacheTestSuite) TearDownTest() {
	suite.data = nil
	suite.cache = nil
}

func (suite *NotificationCacheTestSuite) TestStatusCache() {
	for _, notif := range suite.data {
		// Place in the cache
		suite.cache.Set(notif)
	}

	for _, notif := range suite.data {
		var ok bool
		var check *gtsmodel.Notification

		// Check we can retrieve
		check, ok = suite.cache.Get(notif.ID)
		if !ok && !notifIs(notif, check) {
			suite.Fail("Failed to fetch expected notification with ID")
		}
	}
}

func TestNotificationCache(t *testing.T) {
	suite.Run(t, &StatusCacheTestSuite{})
}

func notifIs(notif1, notif2 *gtsmodel.Notification) bool {
	return notif1.ID == notif2.ID
}
