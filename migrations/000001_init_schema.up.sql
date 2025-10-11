-- Initialize wasmorph schema and tables
CREATE SCHEMA IF NOT EXISTS wasmorph;

-- Users table for authentication
CREATE TABLE IF NOT EXISTS wasmorph.users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE
);

CREATE INDEX idx_users_username ON wasmorph.users(username);

-- API keys for programmatic access
CREATE TABLE IF NOT EXISTS wasmorph.api_keys (
    id SERIAL PRIMARY KEY,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    user_id INTEGER NOT NULL REFERENCES wasmorph.users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE
);

CREATE INDEX idx_api_keys_api_key ON wasmorph.api_keys(api_key);

-- Rules table for storing WASM programs
CREATE TABLE IF NOT EXISTS wasmorph.rules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    user_id INTEGER NOT NULL REFERENCES wasmorph.users(id),
    source_code TEXT NOT NULL,
    wasm_binary BYTEA NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE,
    UNIQUE(name, user_id)
);

CREATE INDEX idx_rules_name ON wasmorph.rules(name);
CREATE INDEX idx_rules_user_id ON wasmorph.rules(user_id);
