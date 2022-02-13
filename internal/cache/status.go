package cache

import (
	"time"

	"codeberg.org/gruf/go-cache"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// StatusCache is a wrapper around cache.LookupCache to provide URL and URI lookups for gtsmodel.Status
type StatusCache struct {
	cache cache.LookupCache
}

// NewStatusCache returns a new instantiated statusCache object
func NewStatusCache() *StatusCache {
	c := cache.NewLookup(cache.LookupCfg{
		RegisterLookups: func(lm *cache.LookupMap) {
			lm.RegisterLookup("uri")
			lm.RegisterLookup("url")
		},
		AddLookups: func(lm *cache.LookupMap, i interface{}) {
			status := i.(*gtsmodel.Status)
			if status.URI != "" {
				lm.Set("uri", status.URI, status.ID)
			}
			if status.URL != "" {
				lm.Set("url", status.URL, status.ID)
			}
		},
		DeleteLookups: func(lm *cache.LookupMap, i interface{}) {
			status := i.(*gtsmodel.Status)
			if status.URI != "" {
				lm.Delete("uri", status.URI)
			}
			if status.URL != "" {
				lm.Delete("url", status.URL)
			}
		},
	})
	if !c.Stop() || !c.Start(time.Second*30) {
		panic("failed starting cache")
	}
	return &StatusCache{cache: c}
}

// GetByID attempts to fetch a status from the cache by its ID, you will receive a copy for thread-safety
func (c *StatusCache) GetByID(id string) (*gtsmodel.Status, bool) {
	v, ok := c.cache.Get(id)
	if !ok {
		return nil, false
	}
	return copyStatus(v.(*gtsmodel.Status)), true
}

// GetByURL attempts to fetch a status from the cache by its URL, you will receive a copy for thread-safety
func (c *StatusCache) GetByURL(url string) (*gtsmodel.Status, bool) {
	v, ok := c.cache.GetBy("url", url)
	if !ok {
		return nil, false
	}
	return copyStatus(v.(*gtsmodel.Status)), true
}

// GetByURI attempts to fetch a status from the cache by its URI, you will receive a copy for thread-safety
func (c *StatusCache) GetByURI(uri string) (*gtsmodel.Status, bool) {
	v, ok := c.cache.GetBy("uri", uri)
	if !ok {
		return nil, false
	}
	return copyStatus(v.(*gtsmodel.Status)), true
}

// Put places a status in the cache, ensuring that the object place is a copy for thread-safety
func (c *StatusCache) Put(status *gtsmodel.Status) {
	if status == nil || status.ID == "" {
		panic("invalid status")
	}
	c.cache.Set(status.ID, copyStatus(status))
}

// copyStatus performs a surface-level copy of status, only keeping attached IDs intact, not the objects.
// due to all the data being copied being 99% primitive types or strings (which are immutable and passed by ptr)
// this should be a relatively cheap process
func copyStatus(status *gtsmodel.Status) *gtsmodel.Status {
	new := *status
	new.Attachments = nil
	new.Tags = nil
	new.Mentions = nil
	new.Emojis = nil
	new.Account = nil
	new.InReplyTo = nil
	new.InReplyToAccount = nil
	new.BoostOf = nil
	new.BoostOfAccount = nil
	return &new
}
