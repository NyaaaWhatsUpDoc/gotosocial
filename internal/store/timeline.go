package store

import (
	"context"

	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

func (s *Store) GetHomeTimeline(ctx context.Context, accountID string, pg Pagination) ([]*gtsmodel.Status, error) {
	return nil, nil
}

func (s *Store) GetPublicTimeline(ctx context.Context, accountID string, pg Pagination) ([]*gtsmodel.Status, error) {
	return nil, nil
}

func (s *Store) GetFaveTimeline(ctx context.Context, accountID string, pg Pagination) ([]*gtsmodel.Status, error) {
	return nil, nil
}
