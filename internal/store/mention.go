package store

import (
	"context"

	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

func (s *Store) GetBareMention(ctx context.Context, id string) (*gtsmodel.Mention, error) {
	return nil, nil
}

func (s *Store) GetMention(ctx context.Context, id string) (*gtsmodel.Mention, error) {
	// Check for mention in cache
	mention, ok := s.MentionCache().Get(id)
	if ok {
		return mention, nil
	}

	// TODO: fetch from database

	return nil, nil
}

func (s *Store) PutMention(ctx context.Context, mention *gtsmodel.Mention) error {
	return nil
}
