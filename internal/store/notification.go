package store

import (
	"context"

	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

func (s *Store) GetNotification(ctx context.Context, id string) (*gtsmodel.Notification, error) {
	// Check for notification in cache
	notif, ok := s.NotificationCache().Get(id)
	if ok {
		return notif, nil
	}

	// TODO: fetch from DB

	return nil, nil
}

func (s *Store) PutNotification(ctx context.Context, notif *gtsmodel.Notification) error {
	return nil
}
