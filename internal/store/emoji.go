package store

import (
	"context"

	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

func (s *Store) GetEmoji(ctx context.Context, id string) (*gtsmodel.Emoji, error) {
	// Check for emoji in the cache
	emoji, ok := s.EmojiCache().Get(id)
	if ok {
		return emoji, nil
	}

	// TODO: fetch from database
	return nil, nil
}

func (s *Store) PutEmoji(ctx context.Context, emoji *gtsmodel.Emoji) error {
	return nil
}
