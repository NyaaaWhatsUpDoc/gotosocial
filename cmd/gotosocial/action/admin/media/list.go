// GoToSocial
// Copyright (C) GoToSocial Authors admin@gotosocial.org
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package media

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/superseriousbusiness/gotosocial/cmd/gotosocial/action"
	"github.com/superseriousbusiness/gotosocial/internal/config"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/db/bundb"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/log"
	"github.com/superseriousbusiness/gotosocial/internal/paging"
	"github.com/superseriousbusiness/gotosocial/internal/state"
)

type list struct {
	dbService  db.DB
	state      *state.State
	localOnly  bool
	remoteOnly bool
	out        *bufio.Writer
}

// Get a list of attachment using a custom filter
func (l *list) GetAllAttachmentPaths(ctx context.Context, filter func(*gtsmodel.MediaAttachment) string) ([]string, error) {
	// page for iterative media fetching
	// from a previous maximum media ID.
	page := paging.Page[string]{
		Limit: 200,
	}

	res := make([]string, 0, 100)

	for {
		attachments, err := l.dbService.GetAttachments(ctx, &page)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve media metadata from database: %w", err)
		}

		for _, a := range attachments {
			v := filter(a)
			if v != "" {
				res = append(res, v)
			}
		}

		// If we got less results than our limit, we've reached the
		// last page to retrieve and we can break the loop. If the
		// last batch happens to contain exactly the same amount of
		// items as the limit we'll end up doing one extra query.
		if len(attachments) < page.Limit {
			break
		}

		// Use last attachment as next page maxID value.
		page.Max.Value = attachments[len(attachments)-1].ID
	}

	return res, nil
}

// Get a list of emojis using a custom filter
func (l *list) GetAllEmojisPaths(ctx context.Context, filter func(*gtsmodel.Emoji) string) ([]string, error) {
	// page for iterative media fetching
	// from a previous maximum media ID.
	page := paging.Page[string]{
		Limit: 200,
	}

	res := make([]string, 0, 100)

	for {
		emojis, err := l.dbService.GetEmojis(ctx, &page)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve media metadata from database: %w", err)
		}

		for _, a := range emojis {
			v := filter(a)
			if v != "" {
				res = append(res, v)
			}
		}

		// If we got less results than our limit, we've reached the
		// last page to retrieve and we can break the loop. If the
		// last batch happens to contain exactly the same amount of
		// items as the limit we'll end up doing one extra query.
		if len(emojis) < page.Limit {
			break
		}

		// Use last emoji as next page maxID value.
		page.Max.Value = emojis[len(emojis)-1].ID
	}
	return res, nil
}

func setupList(ctx context.Context) (*list, error) {
	var (
		localOnly  = config.GetAdminMediaListLocalOnly()
		remoteOnly = config.GetAdminMediaListRemoteOnly()
		state      state.State
	)

	// Validate flags.
	if localOnly && remoteOnly {
		return nil, errors.New(
			"local-only and remote-only flags cannot be true at the same time; " +
				"choose one or the other, or set neither to list all media",
		)
	}

	state.Caches.Init()
	state.Caches.Start()

	state.Workers.Start()

	dbService, err := bundb.NewBunDBService(ctx, &state)
	if err != nil {
		return nil, fmt.Errorf("error creating dbservice: %w", err)
	}
	state.DB = dbService

	return &list{
		dbService:  dbService,
		state:      &state,
		localOnly:  localOnly,
		remoteOnly: remoteOnly,
		out:        bufio.NewWriter(os.Stdout),
	}, nil
}

func (l *list) shutdown() error {
	l.out.Flush()
	err := l.dbService.Close()
	l.state.Workers.Stop()
	l.state.Caches.Stop()
	return err
}

// ListAttachments lists local, remote, or all attachment paths.
var ListAttachments action.GTSAction = func(ctx context.Context) error {
	list, err := setupList(ctx)
	if err != nil {
		return err
	}

	defer func() {
		// Ensure lister gets shutdown on exit.
		if err := list.shutdown(); err != nil {
			log.Error(ctx, err)
		}
	}()

	var (
		mediaPath = config.GetStorageLocalBasePath()
		filter    func(*gtsmodel.MediaAttachment) string
	)

	switch {
	case list.localOnly:
		filter = func(m *gtsmodel.MediaAttachment) string {
			if m.RemoteURL != "" {
				// Remote, not
				// interested.
				return ""
			}

			return path.Join(mediaPath, m.File.Path)
		}

	case list.remoteOnly:
		filter = func(m *gtsmodel.MediaAttachment) string {
			if m.RemoteURL == "" {
				// Local, not
				// interested.
				return ""
			}

			return path.Join(mediaPath, m.File.Path)
		}

	default:
		filter = func(m *gtsmodel.MediaAttachment) string {
			return path.Join(mediaPath, m.File.Path)
		}
	}

	attachments, err := list.GetAllAttachmentPaths(ctx, filter)
	if err != nil {
		return err
	}

	for _, a := range attachments {
		_, _ = list.out.WriteString(a + "\n")
	}
	return nil
}

// ListEmojis lists local, remote, or all emoji filepaths.
var ListEmojis action.GTSAction = func(ctx context.Context) error {
	list, err := setupList(ctx)
	if err != nil {
		return err
	}

	defer func() {
		// Ensure lister gets shutdown on exit.
		if err := list.shutdown(); err != nil {
			log.Error(ctx, err)
		}
	}()

	var (
		mediaPath = config.GetStorageLocalBasePath()
		filter    func(*gtsmodel.Emoji) string
	)

	switch {
	case list.localOnly:
		filter = func(e *gtsmodel.Emoji) string {
			if e.ImageRemoteURL != "" {
				// Remote, not
				// interested.
				return ""
			}

			return path.Join(mediaPath, e.ImagePath)
		}

	case list.remoteOnly:
		filter = func(e *gtsmodel.Emoji) string {
			if e.ImageRemoteURL == "" {
				// Local, not
				// interested.
				return ""
			}

			return path.Join(mediaPath, e.ImagePath)
		}

	default:
		filter = func(e *gtsmodel.Emoji) string {
			return path.Join(mediaPath, e.ImagePath)
		}
	}

	emojis, err := list.GetAllEmojisPaths(ctx, filter)
	if err != nil {
		return err
	}

	for _, e := range emojis {
		_, _ = list.out.WriteString(e + "\n")
	}
	return nil
}
