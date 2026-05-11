# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

FIFA World Cup Prediction Platform backend — a REST API for passwordless auth (OTP + JWT), private prediction boards, match pick'ems, and group standings. Built with Go + PostgreSQL + Redis, runs entirely in Docker Compose for local development.

## Commands

All `make` commands that don't have an `-inner` suffix exec into the Docker `server` container. Run them from your host machine.

```bash
# Tests
make test-unit                        # Run all unit tests (via Docker)
make test-coverage                    # Run tests + generate coverage.html

# Run a single test or package directly inside Docker
docker compose exec server go test -v -race -count=1 ./internal/services/... -run TestAuthService_RequestOtp

# Migrations
make db-migrate-create <name>         # Create a new numbered migration
make db-migrate-up                    # Apply all pending migrations
make db-migrate-down                  # Roll back last migration
make db-test-migrate-up               # Apply migrations to test DB

# Seeding
make db-seed                          # Seed dev DB
make db-flush                         # Clear all data from dev DB

# Docs
make swagger                          # Regenerate OpenAPI spec in docs/
```

Lint/vet run only in CI (`.github/workflows/ci.yml`): `go mod verify`, `go vet`, `staticcheck`.

Local dev services:
- API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/swagger/index.html`
- Redis Commander: `http://localhost:8081`
- Dev DB: port 5432 | Test DB: port 5433

## Architecture

Clean Architecture with a strict dependency direction:

```
Handler → Service → Repository / Storage → DB / Redis
```

```
internal/
  app/             DI container (container.go) + router (router.go)
  domain/          Entities, interfaces, error types, scoring config
  dtos/            Request/response structs with validator tags
  handlers/        HTTP handlers, one file per resource domain
  services/        Business logic, implement domain interfaces
  repositories/    PostgreSQL queries, one repo per entity
  storage/         Redis access (OTP, user cache, OAuth state)
  middlewares/     Auth, rate limit, logging, CORS, board membership guard
  infrastructure/  Cross-cutting: config, DB pool, Redis client, JWT, OAuth, mailer, cron, validator
  jobs/            Background cron jobs (session cleanup, match result sync)
  test/            Shared mocks and test helpers
cmd/
  api/             main.go — loads config, builds container, starts server
  db/
    migrations/    Sequential SQL migration files
    seed/          Seeding utility (go run ./cmd/db/seed)
```

### Key patterns

**DI via container** — `internal/app/container.go` constructs every dependency once and wires them together. Tests inject mock implementations via constructors; no global state.

**Domain errors** — `internal/domain/errors.go` defines typed errors (e.g., `OtpCooldownError`, `ErrBoardNotFound`). `internal/handlers/errors.go` maps these to HTTP status codes. Do not use generic `errors.New` for business failures.

**Request context** — middleware enriches `context.Context` with user ID, session ID, request ID, and device info. Handlers retrieve these via `httpctx.GetRequestInfo(r)`.

**Validation** — DTOs carry `validate:` struct tags; handlers call the shared validator from `internal/infrastructure/validator/`. Validation errors surface as 422 with field-level detail.

**Swagger annotations** — handler methods carry `// @Summary`, `// @Param`, `// @Success` godoc comments. Run `make swagger` after changing handler signatures or DTOs.

**Migrations** — sequential numbered SQL files in `cmd/db/migrations/`. Always create with `make db-migrate-create <name>` to get the correct sequence number. Both dev and test DBs must be migrated separately.
