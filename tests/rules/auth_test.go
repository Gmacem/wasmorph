package rules

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/Gmacem/wasmorph/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AuthTestSuite struct {
	suite.Suite
	dbClient   *helpers.DatabaseClient
	httpClient *helpers.HTTPClient
	testAPIKey string
	username   string
}

func (suite *AuthTestSuite) SetupSuite() {
	var err error
	suite.dbClient, err = helpers.NewDatabaseClient()
	require.NoError(suite.T(), err)
	suite.httpClient = helpers.NewHTTPClient()
}

func (suite *AuthTestSuite) TearDownSuite() {
	if suite.dbClient != nil {
		suite.dbClient.Close()
	}
}

func (suite *AuthTestSuite) SetupTest() {
	// Clean everything and setup fresh for each test
	suite.dbClient.CleanupAll()

	suite.username = "testuser"
	suite.testAPIKey = "test-api-key-12345"

	err := suite.dbClient.AddUser(suite.username, "hashed-password")
	require.NoError(suite.T(), err)

	err = suite.dbClient.AddAPIKey(suite.testAPIKey, suite.username)
	require.NoError(suite.T(), err)
}

func (suite *AuthTestSuite) TestAPIKeyAuthentication() {
	resp, err := suite.httpClient.CreateRule(suite.testAPIKey, "test-rule", transformProgram)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// Debug: print response body
	bodyBytes, _ := io.ReadAll(resp.Body)
	suite.T().Logf("Response status: %d", resp.StatusCode)
	suite.T().Logf("Response body: %s", string(bodyBytes))

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var response map[string]string
	err = json.Unmarshal(bodyBytes, &response)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Rule created", response["message"])
}

func (suite *AuthTestSuite) TestAPIKeyAuthenticationInvalid() {
	resp, err := suite.httpClient.CreateRule("invalid-key", "test-rule", transformProgram)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}
