package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Gmacem/wasmorph/internal/sql"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	queries  *sql.Queries
	compiler *Compiler
	cache    RuntimeCache
}

func NewService(pool *pgxpool.Pool, cache RuntimeCache) *Service {
	if cache == nil {
		cache = &NoOpCache{}
	}

	return &Service{
		queries:  sql.New(pool),
		compiler: NewCompiler("wasm-template", "/tmp"),
		cache:    cache,
	}
}

func (s *Service) SaveRule(ctx context.Context, userID, name, sourceCode string) (sql.WasmorphRule, error) {
	userIDInt, err := strconv.ParseInt(userID, 10, 32)
	if err != nil {
		return sql.WasmorphRule{}, fmt.Errorf("invalid user ID: %w", err)
	}

	wasmBytes, err := s.compiler.CompileGoToWasm(sourceCode, name)
	if err != nil {
		return sql.WasmorphRule{}, fmt.Errorf("compilation failed: %w", err)
	}

	rule, err := s.queries.CreateRule(ctx, sql.CreateRuleParams{
		Name:       name,
		UserID:     int32(userIDInt),
		SourceCode: sourceCode,
		WasmBinary: wasmBytes,
		IsActive:   pgtype.Bool{Bool: true, Valid: true},
	})

	if err != nil {
		return sql.WasmorphRule{}, fmt.Errorf("failed to save rule: %w", err)
	}

	return rule, nil
}

func (s *Service) ExecuteRule(ctx context.Context, userID, name string, input map[string]any) ([]byte, error) {
	userIDInt, err := strconv.ParseInt(userID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	cacheKey := fmt.Sprintf("%d:%s", userIDInt, name)

	if runtime, found := s.cache.Get(ctx, cacheKey); found && runtime != nil {
		return s.executeWithRuntime(runtime, input)
	}

	rule, err := s.queries.GetRuleByNameAndUser(ctx, sql.GetRuleByNameAndUserParams{
		Name:   name,
		UserID: int32(userIDInt),
	})
	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	runtime, err := NewRuntime(rule.WasmBinary)
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime: %w", err)
	}

	cost := int64(len(rule.WasmBinary))
	s.cache.Set(ctx, cacheKey, runtime, cost)

	return s.executeWithRuntime(runtime, input)
}

func (s *Service) executeWithRuntime(runtime *Runtime, input map[string]any) ([]byte, error) {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	result, err := runtime.ExecuteTransform(inputBytes)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	return result, nil
}

func (s *Service) ListRules(ctx context.Context, userID string) ([]sql.ListRulesByUserRow, error) {
	userIDInt, err := strconv.ParseInt(userID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	rules, err := s.queries.ListRulesByUser(ctx, int32(userIDInt))
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	return rules, nil
}

func (s *Service) GetRule(ctx context.Context, userID, name string) (sql.WasmorphRule, error) {
	userIDInt, err := strconv.ParseInt(userID, 10, 32)
	if err != nil {
		return sql.WasmorphRule{}, fmt.Errorf("invalid user ID: %w", err)
	}

	rule, err := s.queries.GetRuleByNameAndUser(ctx, sql.GetRuleByNameAndUserParams{
		Name:   name,
		UserID: int32(userIDInt),
	})
	if err != nil {
		return sql.WasmorphRule{}, fmt.Errorf("rule not found: %w", err)
	}

	return rule, nil
}

func (s *Service) DeleteRule(ctx context.Context, userID, name string) error {
	userIDInt, err := strconv.ParseInt(userID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	return s.queries.DeleteRule(ctx, sql.DeleteRuleParams{
		Name:   name,
		UserID: int32(userIDInt),
	})
}
