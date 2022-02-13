package cache

import (
	"time"

	"codeberg.org/gruf/go-cache"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// MentionCache is a wrapper around cache.LookupCache to provide URL and URI lookups for gtsmodel.Mention
type MentionCache struct {
	cache cache.Cache
}

// NewMentionCache returns a new instantiated MentionCache object
func NewMentionCache() *MentionCache {
	c := cache.New()
	if !c.Stop() || !c.Start(time.Second*30) {
		panic("failed starting cache")
	}
	return &MentionCache{cache: c}
}

// GetByID attempts to fetch a mention from the cache by its ID, you will receive a copy for thread-safety
func (c *MentionCache) GetByID(id string) (*gtsmodel.Mention, bool) {
	v, ok := c.cache.Get(id)
	if !ok {
		return nil, false
	}
	return copyMention(v.(*gtsmodel.Mention)), true
}

// Put places a mention in the cache, ensuring that the object place is a copy for thread-safety
func (c *MentionCache) Put(mention *gtsmodel.Mention) {
	if mention == nil || mention.ID == "" {
		panic("invalid mention")
	}
	c.cache.Set(mention.ID, copyMention(mention))
}

// copyMention performs a surface-level copy of mention, only keeping attached IDs intact, not the objects.
// due to all the data being copied being 99% primitive types or strings (which are immutable and passed by ptr)
// this should be a relatively cheap process
func copyMention(mention *gtsmodel.Mention) *gtsmodel.Mention {
	new := *mention
	new.Status = nil
	new.OriginAccount = nil
	new.TargetAccount = nil
	return &new
}
