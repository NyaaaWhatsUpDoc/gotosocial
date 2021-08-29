package visibility_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/visibility"
	"github.com/superseriousbusiness/gotosocial/testrig"
)

type FilterTestSuite struct {
	db     db.DB // hold onto our own ptr so we can act on it
	filter visibility.Filter

	testAccounts map[string]*gtsmodel.Account
	testStatuses map[string]*gtsmodel.Status

	suite.Suite
}

func TestFilterTestSuite(t *testing.T) {
	suite.Run(t, new(FilterTestSuite))
}

func (suite *FilterTestSuite) SetupSuite() {
	suite.testAccounts = testrig.NewTestAccounts()
	suite.testStatuses = testrig.NewTestStatuses()
}

func (suite *FilterTestSuite) SetupTest() {
	suite.db = testrig.NewTestDB()

	suite.filter = visibility.NewFilter(
		suite.db,
		testrig.NewTestLog(),
	)

	testrig.StandardDBSetup(suite.db, suite.testAccounts)
}

func (suite *FilterTestSuite) TearDownTest() {
	testrig.StandardDBTeardown(suite.db)
}

func (suite *FilterTestSuite) runTests(test func(*gtsmodel.Account, *gtsmodel.Status)) {
	for _, status := range suite.testStatuses {
		for _, account := range suite.testAccounts {
			test(account, status)
		}
	}
}

func (suite *FilterTestSuite) TestStatusVisible() {
	suite.runTests(func(account *gtsmodel.Account, status *gtsmodel.Status) {
		_, err := suite.filter.StatusVisible(context.Background(), status, account)
		suite.NoError(err)
	})
}

func (suite *FilterTestSuite) TestStatusHomeTimelineable() {
	suite.runTests(func(account *gtsmodel.Account, status *gtsmodel.Status) {
		_, err := suite.filter.StatusHometimelineable(context.Background(), status, account)
		suite.NoError(err)
	})
}

func (suite *FilterTestSuite) TestStatusPublicTimelineable() {
	suite.runTests(func(account *gtsmodel.Account, status *gtsmodel.Status) {
		_, err := suite.filter.StatusPublictimelineable(context.Background(), status, account)
		suite.NoError(err)
	})
}
