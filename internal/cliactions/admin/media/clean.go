/*
   GoToSocial
   Copyright (C) 2021 GoToSocial Authors admin@gotosocial.org

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package media

import (
	"context"
	"fmt"

	"git.iim.gay/grufwub/go-store/kv"
	"github.com/sirupsen/logrus"
	"github.com/superseriousbusiness/gotosocial/internal/cliactions"
	"github.com/superseriousbusiness/gotosocial/internal/config"
	"github.com/superseriousbusiness/gotosocial/internal/db/bundb"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// ensure we conform to GTSAction func signature
var _ cliactions.GTSAction = Clean

func Clean(ctx context.Context, cfg *config.Config, log *logrus.Logger) error {
	// Open connection to database
	dbConn, err := bundb.NewBunDBService(ctx, cfg, log)
	if err != nil {
		return fmt.Errorf("error creating dbservice: %s", err)
	}

	// Open the storage backend
	storage, err := kv.OpenFile(cfg.StorageConfig.BasePath, nil)
	if err != nil {
		return fmt.Errorf("error creating storage backend: %s", err)
	}

	// Fetch media attachments in DB
	attachments := []*gtsmodel.MediaAttachment{}
	err = dbConn.GetAll(ctx, attachments)
	if err != nil {
		return fmt.Errorf("error fetching media attachments from DB: %s", err)
	}

	// Fetch a list of all statuses and account
	statuses := []*gtsmodel.Status{}
	accounts := []*gtsmodel.Account{}
	err = dbConn.GetAll(ctx, statuses)
	if err != nil {
		return fmt.Errorf("error fetching statuses from DB: %s", err)
	}
	err = dbConn.GetAll(ctx, accounts)
	if err != nil {
		return fmt.Errorf("error fetching accounts from DB: %s", err)
	}

	for _, attachment := range attachments {
		filepath := attachment.File.Path

		// Attempt to retrieve file at path
		_, err := storage.Get(filepath)
		if err == nil {
			// no issue!
			continue
		}

		log.Infof("Found missing media for %s: %s", attachment.ID, filepath)

		// Remove the DB entry for this broke attachment
		err = dbConn.DeleteByID(ctx, attachment.ID, nil)
		if err != nil {
			log.Errorf("error removing media attachment from DB: %s", err)
			continue
		}

		var updated bool

		// Find and update any statuses referencing this in DB
		for _, status := range statuses {
			status.AttachmentIDs, updated = removeFrom(status.AttachmentIDs, attachment.ID)
			if updated {
				err = dbConn.UpdateByPrimaryKey(ctx, status)
				if err != nil {
					return fmt.Errorf("error updating status in DB: %s", err)
				}
			}
		}

		// Find and update any accounts referencing this in DB
		for _, account := range accounts {
			updated = false

			if account.AvatarMediaAttachmentID == attachment.ID {
				account.AvatarMediaAttachmentID = ""
				updated = true
			}

			if account.HeaderMediaAttachmentID == attachment.ID {
				account.HeaderMediaAttachmentID = ""
				updated = true
			}

			if updated {
				err = dbConn.UpdateByPrimaryKey(ctx, account)
				if err != nil {
					return fmt.Errorf("error updating account in DB: %s", err)
				}
			}
		}
	}

	return nil
}

func removeFrom(from []string, remove string) ([]string, bool) {
	removed := false

	i := 0
	for i < len(from) {
		if from[i] == remove {
			from = append(from[:i], from[i+1:]...)
			removed = true
		} else {
			i++
		}
	}

	return from, removed
}
