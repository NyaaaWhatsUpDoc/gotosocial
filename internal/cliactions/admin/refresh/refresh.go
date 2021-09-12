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

package refresh

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"git.iim.gay/grufwub/go-store/kv"
	"github.com/sirupsen/logrus"
	"github.com/superseriousbusiness/gotosocial/internal/cliactions"
	"github.com/superseriousbusiness/gotosocial/internal/config"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/db/bundb"
	"github.com/superseriousbusiness/gotosocial/internal/federation"
	"github.com/superseriousbusiness/gotosocial/internal/federation/federatingdb"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/media"
	"github.com/superseriousbusiness/gotosocial/internal/oauth"
	"github.com/superseriousbusiness/gotosocial/internal/processing"
	"github.com/superseriousbusiness/gotosocial/internal/timeline"
	"github.com/superseriousbusiness/gotosocial/internal/transport"
	"github.com/superseriousbusiness/gotosocial/internal/typeutils"
)

// ensure we conform to GTSAction func signature
var _ cliactions.GTSAction = ForceRefresh

func ForceRefresh(ctx context.Context, cfg *config.Config, log *logrus.Logger) error {
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

	// Build converters and utils
	typeConv := typeutils.NewConverter(cfg, dbConn, log)
	timelineMgr := timeline.NewManager(dbConn, typeConv, cfg, log)

	// Build backend handlers
	media := media.New(cfg, dbConn, storage, log)
	oauth := oauth.New(dbConn, log)
	transportCtrl := transport.NewController(cfg, dbConn, &federation.Clock{}, http.DefaultClient, log)
	fedDB := federatingdb.New(dbConn, cfg, log)
	federator := federation.NewFederator(dbConn, fedDB, transportCtrl, cfg, log, typeConv, media)
	processor := processing.NewProcessor(cfg, typeConv, federator, oauth, media, storage, timelineMgr, dbConn, log)
	_ = processor

	// Fetch all remote accounts from DB
	accounts := []*gtsmodel.Account{}
	err = dbConn.GetWhere(ctx, []db.Where{{Key: "domain", Value: nil}}, accounts)
	if err != nil {
		return fmt.Errorf("error fetch accounts from DB: %s", err)
	}

	// Iterate accounts and fetch remote
	for _, acc := range accounts {
		// Parse the remote account URI
		accURI, err := url.Parse(acc.URI)
		if err != nil {
			return fmt.Errorf("account with invalid URI in DB: %s", err)
		}

		// Perform a force refresh of the remote account
		_, _, err = federator.GetRemoteAccount(ctx, "gotosocial_admin_cli", accURI, true)
		if err != nil {
			return fmt.Errorf("error refreshing remote account: %s", err)
		}
	}

	// Fetch all remote statuses from DB
	statuses := []*gtsmodel.Status{}
	err = dbConn.GetWhere(ctx, []db.Where{{Key: "local", Value: false}}, statuses)
	if err != nil {
		return fmt.Errorf("error fetching statuses from DB: %s", err)
	}

	// Iterate statuses and fetch remote
	for _, status := range statuses {
		// Parse the remote status URI
		statusURI, err := url.Parse(status.URI)
		if err != nil {
			return fmt.Errorf("status with invalid URI in DB: %s", err)
		}

		// Perform a force refresh of the remote status
		_, _, _, err = federator.GetRemoteStatus(ctx, "gotosocial_admin_cli", statusURI, true, false, false)
		if err != nil {
			return fmt.Errorf("error refreshing remote status: %s", err)
		}
	}

	// Fetch all remote media from DB
	attachments := []*gtsmodel.MediaAttachment{}
	err = dbConn.GetWhere(ctx, []db.Where{{Key: "remote_url", Value: nil, Not: true}}, attachments)
	if err != nil {
		return fmt.Errorf("error fetching media attachments from DB: %s", err)
	}

	// Iterate attachments and fetch remote
	for _, attach := range attachments {
		// Perform a force dereference of the remote attachment
		attach, err = federator.RefreshAttachment(ctx, "gotosocial_admin_cli", attach)
		if err != nil {
			return fmt.Errorf("error refreshing remote media attachment: %s", err)
		}

		// Update the media attachment in DB
		err = dbConn.UpdateByPrimaryKey(ctx, attach)
		if err != nil {
			return fmt.Errorf("error updating refreshed media attachment in DB: %s", err)
		}
	}

	return nil
}
