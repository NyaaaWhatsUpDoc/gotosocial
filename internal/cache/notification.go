package cache

import (
	"time"

	"codeberg.org/gruf/go-cache"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// NotificationCache is a wrapper around cache.LookupCache to provide URL and URI lookups for gtsmodel.Notification
type NotificationCache struct {
	cache cache.Cache
}

// NewNotificationCache returns a new instantiated NotificationCache object
func NewNotificationCache() *NotificationCache {
	c := cache.New()
	if !c.Stop() || !c.Start(time.Second*30) {
		panic("failed starting cache")
	}
	return &NotificationCache{cache: c}
}

// GetByID attempts to fetch a notification from the cache by its ID, you will receive a copy for thread-safety
func (c *NotificationCache) GetByID(id string) (*gtsmodel.Notification, bool) {
	v, ok := c.cache.Get(id)
	if !ok {
		return nil, false
	}
	return copyNotification(v.(*gtsmodel.Notification)), true
}

// Put places a notification in the cache, ensuring that the object place is a copy for thread-safety
func (c *NotificationCache) Put(notif *gtsmodel.Notification) {
	if notif == nil || notif.ID == "" {
		panic("invalid notification")
	}
	c.cache.Set(notif.ID, copyNotification(notif))
}

// copyNotification performs a surface-level copy of notification, only keeping attached IDs intact, not the objects.
// due to all the data being copied being 99% primitive types or strings (which are immutable and passed by ptr)
// this should be a relatively cheap process
func copyNotification(mention *gtsmodel.Notification) *gtsmodel.Notification {
	new := *mention
	new.Status = nil
	new.OriginAccount = nil
	new.TargetAccount = nil
	return &new
}
