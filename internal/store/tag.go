package store

import (
	"context"

	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

func (s *Store) GetTag(ctx context.Context, id string) (*gtsmodel.Tag, error) {
	return nil, nil
}

func (s *Store) PutTag(ctx context.Context, tag *gtsmodel.Tag) error {
	return nil
}
