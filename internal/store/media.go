package store

import (
	"context"

	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

func (s *Store) GetMediaAttachment(ctx context.Context, id string) (*gtsmodel.MediaAttachment, error) {
	return nil, nil
}

func (s *Store) PutMediaAttachment(ctx context.Context, attach *gtsmodel.MediaAttachment) error {
	return nil
}
