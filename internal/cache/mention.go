package cache

import (
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// MentionCache is a wrapper around Cache to provide gtsmodel.Mention
type MentionCache struct {
	cache Cache
}

// NewStatus returns a new instantiated MentionCache object
func NewMention() *MentionCache {
	c := MentionCache{}
	c.cache.init()
	return &c
}

func (c *MentionCache) SetEvictionCallback(hook EvictHook) {
	c.cache.SetEvictionCallback(hook)
}

func (c *MentionCache) SetUpdateCallback(hook UpdateHook) {
	c.cache.SetUpdateCallback(hook)
}

// GetByID attempts to fetch a mention from the cache by its ID, you will receive a copy for thread-safety
func (c *MentionCache) Get(id string) (*gtsmodel.Mention, bool) {
	v, ok := c.cache.Get(id)
	if !ok {
		return nil, false
	}
	return copyMention(v.(*gtsmodel.Mention)), true
}

// Set places a mention in the cache, ensuring that the object placed is a copy for thread-safety
func (c *MentionCache) Set(mention *gtsmodel.Mention) {
	// Check for valid input status
	if mention == nil || mention.ID == "" {
		panic("invalid mention")
	}

	// Safely set item in cache
	c.cache.Set(mention.ID, copyMention(mention))
}

// copyMention performs a surface-level copy of mention, only keeping attached IDs intact, not the objects.
// due to all the data being copied being 99% primitive types or strings (which are immutable and passed by ptr)
// this should be a relatively cheap process
func copyMention(mention *gtsmodel.Mention) *gtsmodel.Mention {
	m := *mention
	m.Status = nil
	m.OriginAccount = nil
	m.TargetAccount = nil
	return &m
}
