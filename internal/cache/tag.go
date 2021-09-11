package cache

import "github.com/superseriousbusiness/gotosocial/internal/gtsmodel"

type TagCache struct {
	cache Cache
}

func NewTag() *TagCache {
	c := TagCache{}
	c.cache.init()
	return &c
}

func (c *TagCache) Get(id string) (*gtsmodel.Tag, bool) {
	v, ok := c.cache.Get(id)
	if !ok {
		return nil, false
	}
	return v.(*gtsmodel.Tag), true
}

func (c *TagCache) Set(tag *gtsmodel.Tag) {
	// Check for valid supplied tag
	if tag == nil || tag.ID == "" {
		panic("invalid tag")
	}

	// Safely place tag in cache
	c.cache.Set(tag.ID, tag)
}
