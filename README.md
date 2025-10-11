# Wasmorph

A platform for integrating runtime rules written in Go and compiled to WebAssembly. Define custom logic, store it, and execute it dynamically with JSON input/output.

## What it does

- Write rules as Go functions that take JSON and return JSON
- Store and manage rule functions
- Execute rules via WebAssembly runtime
- Web interface for managing and testing rules

## Local Setup

### 1. Start PostgreSQL

```bash
docker run -d --name wasmorph-db \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=wasmorph \
  -p 6432:5432 \
  postgres:15
```

### 2. Apply Migrations

Install [golang-migrate](https://github.com/golang-migrate/migrate):

```bash
brew install golang-migrate  # macOS
```

Run migrations:

```bash
export DATABASE_URL="postgresql://postgres:postgres@localhost:6432/wasmorph?sslmode=disable"
migrate -path migrations -database "$DATABASE_URL" up
```

### 3. Create User

```bash
docker exec wasmorph-db psql -U postgres -d wasmorph -c \
  "INSERT INTO wasmorph.users (username, password_hash, is_active) 
   VALUES ('admin', 'pass', true);"
```

### 4. Start Server

```bash
export DATABASE_URL="postgresql://postgres:postgres@localhost:6432/wasmorph?sslmode=disable"
export JWT_SECRET="your-secret-key"
export PORT=8080

go run cmd/server/main.go
```

### 5. Access Web UI

Open http://localhost:8080 and login with:
- **Username**: `admin`
- **Password**: `pass`
