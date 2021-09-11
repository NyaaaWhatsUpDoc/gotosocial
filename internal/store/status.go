package store

import (
	"context"

	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

func (s *Store) GetBareStatus(ctx context.Context, id string) (*gtsmodel.Status, error) {
	// Check for status in cache
	status, ok := s.StatusCache().GetByID(id)
	if ok {
		return status, nil
	}
	return nil, nil
}

func (s *Store) GetStatus(ctx context.Context, id string) (*gtsmodel.Status, error) {
	// Check for status in cache
	status, ok := s.StatusCache().GetByID(id)
	if ok {
		return status, nil
	}

	// TODO: fetch from database

	// Finally, return a populated status
	return s.populateStatus(ctx, status)
}

func (s *Store) GetStatusByURI(ctx context.Context, uri string) (*gtsmodel.Status, error) {
	// Check for status in cache
	status, ok := s.StatusCache().GetByURI(uri)
	if ok {
		return status, nil
	}

	// TODO: fetch from database

	// Finally, return a populated status
	return s.populateStatus(ctx, status)
}

func (s *Store) GetStatusByURL(ctx context.Context, url string) (*gtsmodel.Status, error) {
	// Check for status in cache
	status, ok := s.StatusCache().GetByURL(url)
	if ok {
		return status, nil
	}

	// TODO: fetch from database

	// Finally, return a populated status
	return s.populateStatus(ctx, status)
}

// populateStatus will attempt to populate a status with top-level objects (i.e. only bare, no further) from the store
func (s *Store) populateStatus(ctx context.Context, status *gtsmodel.Status) (*gtsmodel.Status, error) {
	var err error

	// Attempt to fill status author
	status.Account, err = s.GetBareAccount(ctx, status.AccountID)
	if err != nil {
		return nil, err
	}

	if status.BoostOfID != "" {
		// Attempt to fill original boosted status
		status.BoostOf, err = s.GetBareStatus(ctx, status.BoostOfID)
		if err != nil {
			return nil, err
		}

		// Attempt to fill original boosted author
		status.BoostOfAccount, err = s.GetBareAccount(ctx, status.BoostOfAccountID)
		if err != nil {
			return nil, err
		}
	}

	if status.InReplyToID != "" {
		// Attempt to fetch in-reply-to status
		status.InReplyTo, err = s.GetBareStatus(ctx, status.InReplyToID)
		if err != nil {
			return nil, err
		}

		// Attempt to fetch in-reply-to author
		status.InReplyToAccount, err = s.GetBareAccount(ctx, status.InReplyToAccountID)
		if err != nil {
			return nil, err
		}
	}

	// Fill the status emoji slice
	for _, id := range status.EmojiIDs {
		emoji, err := s.GetEmoji(ctx, id)
		if err != nil {
			return nil, err
		}
		status.Emojis = append(status.Emojis, emoji)
	}

	// Fill the status mentions slice
	for _, id := range status.MentionIDs {
		mention, err := s.GetMention(ctx, id)
		if err != nil {
			return nil, err
		}
		status.Mentions = append(status.Mentions, mention)
	}

	// Fill the status media slice
	for _, id := range status.AttachmentIDs {
		attach, err := s.GetMediaAttachment(ctx, id)
		if err != nil {
			return nil, err
		}
		status.Attachments = append(status.Attachments, attach)
	}

	// Fill the status tags slice
	for _, id := range status.TagIDs {
		tag, err := s.GetTag(ctx, id)
		if err != nil {
			return nil, err
		}
		status.Tags = append(status.Tags, tag)
	}

	return status, nil
}

func (s *Store) PutStatus(ctx context.Context, status *gtsmodel.Status) error {
	return nil
}
