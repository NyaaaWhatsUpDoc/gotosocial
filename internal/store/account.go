package store

import (
	"context"

	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

func (s *Store) GetBareAccount(ctx context.Context, id string) (*gtsmodel.Account, error) {
	// Check for account in cache
	account, ok := s.AccountCache().GetByID(id)
	if ok {
		return account, nil
	}

	// TODO: fetch from database
	return nil, nil
}

func (s *Store) GetAccount(ctx context.Context, id string) (*gtsmodel.Account, error) {
	// Check for account in cache
	account, ok := s.AccountCache().GetByID(id)
	if ok {
		return account, nil
	}

	// TODO: fetch from database

	// Finally, return a popuated account
	return s.populateAccount(ctx, account)
}

func (s *Store) GetAccountByURI(ctx context.Context, uri string) (*gtsmodel.Account, error) {
	// Check for account in cache
	account, ok := s.AccountCache().GetByURI(uri)
	if ok {
		return account, nil
	}

	// TODO: fetch from database

	// Finally, return a popuated account
	return s.populateAccount(ctx, account)
}

func (s *Store) GetAccountByURL(ctx context.Context, url string) (*gtsmodel.Account, error) {
	// Check for account in cache
	account, ok := s.AccountCache().GetByURL(url)
	if ok {
		return account, nil
	}

	// TODO: fetch from database

	// Finally, return a popuated account
	return s.populateAccount(ctx, account)
}

// populateAccount will attempt to populate an account with top-level objects (i.e. only bare, no further) from the store
func (s *Store) populateAccount(ctx context.Context, account *gtsmodel.Account) (*gtsmodel.Account, error) {
	var err error

	if account.AvatarMediaAttachmentID != "" {
		// Attempt to fill account avatar
		account.AvatarMediaAttachment, err = s.GetMediaAttachment(ctx, account.AvatarMediaAttachment.AccountID)
		if err != nil {
			return nil, err
		}
	}

	if account.HeaderMediaAttachmentID != "" {
		// Attempt to fill account header
		account.HeaderMediaAttachment, err = s.GetMediaAttachment(ctx, account.HeaderMediaAttachmentID)
		if err != nil {
			return nil, err
		}
	}

	return account, nil
}

func (s *Store) UpdateAccount(ctx context.Context, account *gtsmodel.Account) error {
	return nil
}

func (s *Store) PutAccount(ctx context.Context, account *gtsmodel.Account) error {
	return nil
}
