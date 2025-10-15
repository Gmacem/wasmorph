-- name: ValidateAPIKey :one
SELECT user_id FROM wasmorph.api_keys 
WHERE api_key = $1 AND is_active = true;

-- name: GetUserByUsername :one
SELECT id, username, password_hash, email, created_at, updated_at, is_active 
FROM wasmorph.users 
WHERE username = $1 AND is_active = true;

-- name: GetUserByEmail :one
SELECT id, username, password_hash, email, created_at, updated_at, is_active 
FROM wasmorph.users 
WHERE email = $1 AND is_active = true;

-- name: GetUserByID :one
SELECT id, username, password_hash, email, created_at, updated_at, is_active 
FROM wasmorph.users 
WHERE id = $1 AND is_active = true;

-- name: CreateUser :one
INSERT INTO wasmorph.users (username, email, password_hash, is_active)
VALUES ($1, $2, $3, $4)
RETURNING id, username, email, password_hash, created_at, updated_at, is_active;

-- name: CreateAPIKey :one
INSERT INTO wasmorph.api_keys (api_key, user_id, is_active)
VALUES ($1, $2, $3)
RETURNING id, api_key, user_id, created_at, is_active;

-- name: CreateRule :one
INSERT INTO wasmorph.rules (name, user_id, source_code, wasm_binary, is_active)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (name, user_id) 
DO UPDATE SET 
    source_code = EXCLUDED.source_code,
    wasm_binary = EXCLUDED.wasm_binary,
    updated_at = NOW(),
    is_active = EXCLUDED.is_active
RETURNING id, name, user_id, source_code, wasm_binary, created_at, updated_at, is_active;

-- name: GetRuleByNameAndUser :one
SELECT id, name, user_id, source_code, wasm_binary, created_at, updated_at, is_active
FROM wasmorph.rules
WHERE name = $1 AND user_id = $2 AND is_active = true;

-- name: ListRulesByUser :many
SELECT id, name, user_id, created_at, updated_at, is_active
FROM wasmorph.rules
WHERE user_id = $1 AND is_active = true
ORDER BY created_at DESC;

-- name: UpdateRule :one
UPDATE wasmorph.rules
SET source_code = $3, wasm_binary = $4, updated_at = NOW()
WHERE name = $1 AND user_id = $2 AND is_active = true
RETURNING id, name, user_id, source_code, wasm_binary, created_at, updated_at, is_active;

-- name: DeleteRule :exec
UPDATE wasmorph.rules
SET is_active = false, updated_at = NOW()
WHERE name = $1 AND user_id = $2;
