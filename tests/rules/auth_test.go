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

func (suite *AuthTestSuite) TestUserRegistrationAndLogin() {
	suite.dbClient.CleanupAll()

	newUsername := "newuser"
	newEmail := "newuser@example.com"
	newPassword := "password123"

	resp, err := suite.httpClient.Register(newUsername, newEmail, newPassword)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var registerResponse map[string]string
	bodyBytes, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(bodyBytes, &registerResponse)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), registerResponse["access_token"])

	user, err := suite.dbClient.GetUserByUsername(newUsername)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), newUsername, user.Username)
	assert.Equal(suite.T(), newEmail, user.Email)

	loginResp, err := suite.httpClient.Login(newUsername, newPassword)
	require.NoError(suite.T(), err)
	defer loginResp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, loginResp.StatusCode)
}

func (suite *AuthTestSuite) TestLoginWithInvalidCredentials() {
	suite.dbClient.CleanupAll()

	username := "testuser"
	email := "test@example.com"
	password := "correct_password"

	resp, err := suite.httpClient.Register(username, email, password)
	require.NoError(suite.T(), err)
	resp.Body.Close()
	require.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	wrongPasswordResp, err := suite.httpClient.Login(username, "wrong_password")
	require.NoError(suite.T(), err)
	defer wrongPasswordResp.Body.Close()
	assert.Equal(suite.T(), http.StatusUnauthorized, wrongPasswordResp.StatusCode)

	wrongUsernameResp, err := suite.httpClient.Login("wrong_username", password)
	require.NoError(suite.T(), err)
	defer wrongUsernameResp.Body.Close()
	assert.Equal(suite.T(), http.StatusUnauthorized, wrongUsernameResp.StatusCode)
}

func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}
