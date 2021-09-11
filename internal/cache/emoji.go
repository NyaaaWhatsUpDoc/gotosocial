package cache

import "github.com/superseriousbusiness/gotosocial/internal/gtsmodel"

// EmojiCache wraps Cache to provide gtsmodel.Emoji
type EmojiCache struct {
	cache Cache
}

// NewEmoji returns a new instantiated EmojiCache
func NewEmoji() *EmojiCache {
	c := EmojiCache{}
	c.cache.init()
	return &c
}

func (c *EmojiCache) SetEvictionCallback(hook EvictHook) {
	c.cache.SetEvictionCallback(hook)
}

func (c *EmojiCache) SetUpdateCallback(hook UpdateHook) {
	c.cache.SetUpdateCallback(hook)
}

// Get fetches an emoji with ID from the cache
func (c *EmojiCache) Get(id string) (*gtsmodel.Emoji, bool) {
	v, ok := c.cache.Get(id)
	if !ok {
		return nil, false
	}
	return v.(*gtsmodel.Emoji), true
}

// Set places an emoji in the cache
func (c *EmojiCache) Set(emoji *gtsmodel.Emoji) {
	// Check for valid supplied emoji
	if emoji == nil || emoji.ID == "" {
		panic("invalid emoji")
	}

	// Safely place emoji in cache
	c.cache.Set(emoji.ID, emoji)
}
