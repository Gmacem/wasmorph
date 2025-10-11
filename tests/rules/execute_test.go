package rules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/Gmacem/wasmorph/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExecuteRulesTestSuite struct {
	suite.Suite
	dbClient     *helpers.DatabaseClient
	httpClient   *helpers.HTTPClient
	apiKey       string
	testRuleName string
	testUserID   string
}

func (suite *ExecuteRulesTestSuite) SetupSuite() {
	// Initialize clients only
	var err error
	suite.dbClient, err = helpers.NewDatabaseClient()
	require.NoError(suite.T(), err)
	suite.httpClient = helpers.NewHTTPClient()
}

func (suite *ExecuteRulesTestSuite) TearDownSuite() {
	if suite.dbClient != nil {
		suite.dbClient.Close()
	}
}

func (suite *ExecuteRulesTestSuite) SetupTest() {
	// Clean everything and setup fresh for each test
	suite.dbClient.CleanupAll()

	suite.testUserID = "testuser-execute"
	suite.apiKey = "test-api-key-execute"
	suite.testRuleName = "execute-test-rule"

	err := suite.dbClient.AddUser(suite.testUserID, "hashed-password")
	require.NoError(suite.T(), err)

	err = suite.dbClient.AddAPIKey(suite.apiKey, suite.testUserID)
	require.NoError(suite.T(), err)

	// Verify that user and API key were created
	err = suite.dbClient.VerifyUserAndAPIKey(suite.testUserID, suite.apiKey)
	require.NoError(suite.T(), err, "User and API key should exist in database")
}

func (suite *ExecuteRulesTestSuite) createTestRule() error {
	sourceCode := `func Transform(in []byte) []byte {
	return []byte("Hello from WASM! Input: " + string(in))
}`

	resp, err := suite.httpClient.CreateRule(suite.apiKey, suite.testRuleName, sourceCode)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create rule: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (suite *ExecuteRulesTestSuite) TestExecuteRule() {
	err := suite.createTestRule()
	require.NoError(suite.T(), err)

	input := map[string]any{
		"message": "Hello World",
		"number":  42,
	}

	resp, err := suite.httpClient.ExecuteRule(suite.apiKey, suite.testRuleName, input)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// Debug: print response body if status is not OK
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		suite.T().Logf("Response status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(suite.T(), err)

	var response map[string]any
	err = json.Unmarshal(bodyBytes, &response)
	require.NoError(suite.T(), err)

	expectedResult := "Hello from WASM! Input: {\"message\":\"Hello World\",\"number\":42}"
	assert.Equal(suite.T(), expectedResult, response["result"])
}

func (suite *ExecuteRulesTestSuite) TestExecuteRuleWithComplexInput() {
	err := suite.createTestRule()
	require.NoError(suite.T(), err)

	input := map[string]any{
		"user": map[string]any{
			"name": "John Doe",
			"age":  30,
		},
		"items": []any{"apple", "banana", "cherry"},
		"count": 3,
	}

	resp, err := suite.httpClient.ExecuteRule(suite.apiKey, suite.testRuleName, input)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(suite.T(), err)

	var response map[string]any
	err = json.Unmarshal(bodyBytes, &response)
	require.NoError(suite.T(), err)

	expectedResult := "Hello from WASM! Input: {\"count\":3,\"items\":[\"apple\",\"banana\",\"cherry\"],\"user\":{\"age\":30,\"name\":\"John Doe\"}}"
	assert.Equal(suite.T(), expectedResult, response["result"])
}

func (suite *ExecuteRulesTestSuite) TestExecuteRuleNotFound() {
	input := map[string]any{"test": "value"}
	resp, err := suite.httpClient.ExecuteRule(suite.apiKey, "non-existent-rule", input)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
}

func TestExecuteRulesTestSuite(t *testing.T) {
	suite.Run(t, new(ExecuteRulesTestSuite))
}
