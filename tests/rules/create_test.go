package rules

import (
	"encoding/json"
	"testing"

	"github.com/Gmacem/wasmorph/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CreateRulesTestSuite struct {
	suite.Suite
	dbClient   *helpers.DatabaseClient
	httpClient *helpers.HTTPClient
	apiKey     string
}

func (suite *CreateRulesTestSuite) SetupSuite() {
	var err error
	suite.dbClient, err = helpers.NewDatabaseClient()
	require.NoError(suite.T(), err)
	suite.httpClient = helpers.NewHTTPClient()
}

func (suite *CreateRulesTestSuite) TearDownSuite() {
	if suite.dbClient != nil {
		suite.dbClient.Close()
	}
}

func (suite *CreateRulesTestSuite) SetupTest() {
	suite.dbClient.CleanupAll()

	suite.apiKey = "test-api-key-create"
	userID := "testuser-create"

	err := suite.dbClient.AddUser(userID, "hashed-password")
	require.NoError(suite.T(), err)

	err = suite.dbClient.AddAPIKey(suite.apiKey, userID)
	require.NoError(suite.T(), err)
}

func (suite *CreateRulesTestSuite) TestCreateRule() {
	ruleName := "create-test-rule"
	sourceCode := `func Transform(in []byte) []byte {
		return []byte("processed: " + string(in))
	}`

	resp, err := suite.httpClient.CreateRule(suite.apiKey, ruleName, sourceCode)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), 201, resp.StatusCode)

	listResp, err := suite.httpClient.ListRules(suite.apiKey)
	require.NoError(suite.T(), err)
	defer listResp.Body.Close()

	assert.Equal(suite.T(), 200, listResp.StatusCode)

	var rules []map[string]any
	err = json.NewDecoder(listResp.Body).Decode(&rules)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), rules, 1)
	assert.Equal(suite.T(), ruleName, rules[0]["name"])
	assert.NotNil(suite.T(), rules[0]["user_id"])
}

func TestCreateRulesTestSuite(t *testing.T) {
	suite.Run(t, new(CreateRulesTestSuite))
}
