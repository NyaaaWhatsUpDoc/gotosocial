package store

import (
	"github.com/sirupsen/logrus"
	"github.com/superseriousbusiness/gotosocial/internal/cache"
)

type Store struct {
	accountCache *cache.AccountCache
	emojiCache   *cache.EmojiCache
	mentionCache *cache.MentionCache
	notifCache   *cache.NotificationCache
	tagCache     *cache.TagCache
	statusCache  *cache.StatusCache
	logger       *logrus.Logger
}

func Open(log *logrus.Logger) *Store {
	return &Store{
		accountCache: cache.NewAccount(),
		mentionCache: cache.NewMention(),
		notifCache:   cache.NewNotification(),
		tagCache:     cache.NewTag(),
		statusCache:  cache.NewStatus(),
		logger:       log,
	}
}

// AccountCache returns the store AccountCache
func (s *Store) AccountCache() *cache.AccountCache {
	return s.accountCache
}

// EmojiCache returns the store EmojiCache
func (s *Store) EmojiCache() *cache.EmojiCache {
	return s.emojiCache
}

// MentionCache returns the store MentionCache
func (s *Store) MentionCache() *cache.MentionCache {
	return s.mentionCache
}

// NotificationCache returns the store NotificationCache
func (s *Store) NotificationCache() *cache.NotificationCache {
	return s.notifCache
}

// TagCache returns the store TagCache
func (s *Store) TagCache() *cache.TagCache {
	return s.tagCache
}

// StatusCache returns the store StatusCache
func (s *Store) StatusCache() *cache.StatusCache {
	return s.statusCache
}
