package rules

import (
	"context"
	"testing"

	"github.com/Gmacem/wasmorph/internal/sql"
	"github.com/Gmacem/wasmorph/tests/helpers"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const transformProgram = `func Transform(input []byte) []byte {
	return make([]byte, 0)
}`

type ListRulesTestSuite struct {
	suite.Suite
	dbClient *helpers.DatabaseClient
	queries  *sql.Queries
	conn     *pgx.Conn
}

func (suite *ListRulesTestSuite) SetupSuite() {
	// Initialize clients only
	var err error
	suite.dbClient, err = helpers.NewDatabaseClient()
	require.NoError(suite.T(), err)

	suite.conn, err = pgx.Connect(context.Background(), suite.dbClient.GetDatabaseURL())
	require.NoError(suite.T(), err)

	suite.queries = sql.New(suite.conn)
}

func (suite *ListRulesTestSuite) TearDownSuite() {
	if suite.conn != nil {
		suite.conn.Close(context.Background())
	}
	if suite.dbClient != nil {
		suite.dbClient.Close()
	}
}

func (suite *ListRulesTestSuite) SetupTest() {
	suite.dbClient.CleanupAll()

	err := suite.dbClient.AddUser("user1", "password1")
	require.NoError(suite.T(), err)
	err = suite.dbClient.AddUser("user2", "password2")
	require.NoError(suite.T(), err)
}

func (suite *ListRulesTestSuite) TestListRulesEmpty() {
	userID, err := suite.dbClient.GetUserID("user1")
	require.NoError(suite.T(), err)

	rules, err := suite.queries.ListRulesByUser(context.Background(), userID)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), rules, 0)
}

func (suite *ListRulesTestSuite) TestListRulesSingle() {
	userID, err := suite.dbClient.GetUserID("user1")
	require.NoError(suite.T(), err)
	ruleName := "single-rule"

	_, err = suite.queries.CreateRule(context.Background(), sql.CreateRuleParams{
		Name:       ruleName,
		UserID:     userID,
		SourceCode: transformProgram,
		WasmBinary: []byte{0x01, 0x02, 0x03},
		IsActive:   pgtype.Bool{Bool: true, Valid: true},
	})
	require.NoError(suite.T(), err)

	rules, err := suite.queries.ListRulesByUser(context.Background(), userID)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), rules, 1)
	assert.Equal(suite.T(), ruleName, rules[0].Name)
	assert.Equal(suite.T(), userID, rules[0].UserID)
	assert.True(suite.T(), rules[0].IsActive.Bool)
}

func (suite *ListRulesTestSuite) TestListRulesMultiple() {
	userID, err := suite.dbClient.GetUserID("user1")
	require.NoError(suite.T(), err)
	ruleNames := []string{"rule1", "rule2", "rule3"}

	for i, ruleName := range ruleNames {
		_, err := suite.queries.CreateRule(context.Background(), sql.CreateRuleParams{
			Name:       ruleName,
			UserID:     userID,
			SourceCode: transformProgram,
			WasmBinary: []byte{byte(i + 1), byte(i + 2), byte(i + 3)},
			IsActive:   pgtype.Bool{Bool: true, Valid: true},
		})
		require.NoError(suite.T(), err)
	}

	rules, err := suite.queries.ListRulesByUser(context.Background(), userID)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), rules, 3)

	actualNames := make([]string, len(rules))
	for i, rule := range rules {
		actualNames[i] = rule.Name
		assert.Equal(suite.T(), userID, rule.UserID)
		assert.True(suite.T(), rule.IsActive.Bool)
	}

	for _, expectedName := range ruleNames {
		assert.Contains(suite.T(), actualNames, expectedName)
	}
}

func (suite *ListRulesTestSuite) TestListRulesUserIsolation() {
	user1ID, err := suite.dbClient.GetUserID("user1")
	require.NoError(suite.T(), err)
	user2ID, err := suite.dbClient.GetUserID("user2")
	require.NoError(suite.T(), err)

	_, err = suite.queries.CreateRule(context.Background(), sql.CreateRuleParams{
		Name:       "user1-rule1",
		UserID:     user1ID,
		SourceCode: transformProgram,
		WasmBinary: []byte{0x01, 0x02, 0x03},
		IsActive:   pgtype.Bool{Bool: true, Valid: true},
	})
	require.NoError(suite.T(), err)

	_, err = suite.queries.CreateRule(context.Background(), sql.CreateRuleParams{
		Name:       "user1-rule2",
		UserID:     user1ID,
		SourceCode: transformProgram,
		WasmBinary: []byte{0x04, 0x05, 0x06},
		IsActive:   pgtype.Bool{Bool: true, Valid: true},
	})
	require.NoError(suite.T(), err)

	_, err = suite.queries.CreateRule(context.Background(), sql.CreateRuleParams{
		Name:       "user2-rule1",
		UserID:     user2ID,
		SourceCode: transformProgram,
		WasmBinary: []byte{0x07, 0x08, 0x09},
		IsActive:   pgtype.Bool{Bool: true, Valid: true},
	})
	require.NoError(suite.T(), err)

	user1Rules, err := suite.queries.ListRulesByUser(context.Background(), user1ID)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), user1Rules, 2)

	user2Rules, err := suite.queries.ListRulesByUser(context.Background(), user2ID)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), user2Rules, 1)

	user1Names := make([]string, len(user1Rules))
	for i, rule := range user1Rules {
		user1Names[i] = rule.Name
	}
	assert.Contains(suite.T(), user1Names, "user1-rule1")
	assert.Contains(suite.T(), user1Names, "user1-rule2")
	assert.NotContains(suite.T(), user1Names, "user2-rule1")

	user2Names := make([]string, len(user2Rules))
	for i, rule := range user2Rules {
		user2Names[i] = rule.Name
	}
	assert.Contains(suite.T(), user2Names, "user2-rule1")
	assert.NotContains(suite.T(), user2Names, "user1-rule1")
	assert.NotContains(suite.T(), user2Names, "user1-rule2")
}

func (suite *ListRulesTestSuite) TestListRulesOrderedByCreatedAt() {
	userID, err := suite.dbClient.GetUserID("user1")
	require.NoError(suite.T(), err)
	ruleNames := []string{"first-rule", "second-rule", "third-rule"}

	for i, ruleName := range ruleNames {
		_, err := suite.queries.CreateRule(context.Background(), sql.CreateRuleParams{
			Name:       ruleName,
			UserID:     userID,
			SourceCode: transformProgram,
			WasmBinary: []byte{byte(i + 1), byte(i + 2), byte(i + 3)},
			IsActive:   pgtype.Bool{Bool: true, Valid: true},
		})
		require.NoError(suite.T(), err)
	}

	rules, err := suite.queries.ListRulesByUser(context.Background(), userID)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), rules, 3)

	expectedOrder := []string{"third-rule", "second-rule", "first-rule"}
	actualNames := make([]string, len(rules))
	for i, rule := range rules {
		actualNames[i] = rule.Name
	}

	assert.Equal(suite.T(), expectedOrder, actualNames)
}

func (suite *ListRulesTestSuite) TestListRulesOnlyActive() {
	userID, err := suite.dbClient.GetUserID("user1")
	require.NoError(suite.T(), err)

	_, err = suite.queries.CreateRule(context.Background(), sql.CreateRuleParams{
		Name:       "active-rule",
		UserID:     userID,
		SourceCode: transformProgram,
		WasmBinary: []byte{0x01, 0x02, 0x03},
		IsActive:   pgtype.Bool{Bool: true, Valid: true},
	})
	require.NoError(suite.T(), err)

	rules, err := suite.queries.ListRulesByUser(context.Background(), userID)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), rules, 1)
	assert.Equal(suite.T(), "active-rule", rules[0].Name)
	assert.True(suite.T(), rules[0].IsActive.Bool)
}

func TestListRulesTestSuite(t *testing.T) {
	suite.Run(t, new(ListRulesTestSuite))
}
