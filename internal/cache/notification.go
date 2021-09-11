package cache

import (
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// NotificationCache is a wrapper around Cache for gtsmodel.Notification specific needs
type NotificationCache struct {
	cache Cache
}

// NewStatus returns a new instantiated NotificationCache object
func NewNotification() *NotificationCache {
	c := NotificationCache{}
	c.cache.init()
	return &c
}

func (c *NotificationCache) SetEvictionCallback(hook EvictHook) {
	c.cache.SetEvictionCallback(hook)
}

func (c *NotificationCache) SetUpdateCallback(hook UpdateHook) {
	c.cache.SetUpdateCallback(hook)
}

// Get attempts to fetch a notification from the cache by its ID, you will receive a copy for thread-safety
func (c *NotificationCache) Get(id string) (*gtsmodel.Notification, bool) {
	v, ok := c.cache.Get(id)
	if !ok {
		return nil, false
	}
	return copyNotification(v.(*gtsmodel.Notification)), ok
}

// Set places a notification in the cache, ensuring that the object placed is a copy for thread-safety
func (c *NotificationCache) Set(notif *gtsmodel.Notification) {
	// Check for valid input status
	if notif == nil || notif.ID == "" {
		panic("invalid notification")
	}

	// Safely set item in cache
	c.cache.Set(notif.ID, copyNotification(notif))
}

// copyNotification performs a surface-level copy of status, only keeping attached IDs intact, not the objects.
// due to all the data being copied being 99% primitive types or strings (which are immutable and passed by ptr)
// this should be a relatively cheap process
func copyNotification(notif *gtsmodel.Notification) *gtsmodel.Notification {
	n := *notif
	n.OriginAccount = nil
	n.Status = nil
	n.TargetAccount = nil
	return &n
}
