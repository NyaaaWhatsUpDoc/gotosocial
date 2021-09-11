package store

import (
	"context"

	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

func (s *Store) GetRouterSession(ctx context.Context) (*gtsmodel.RouterSession, error) {
	return nil, nil
}

func (s *Store) PutRouterSession(ctx context.Context, session *gtsmodel.RouterSession) error {
	return nil
}
