include .env
MIGRATIONS_PATH = ./cmd/db/migrations

CYAN  := \033[0;36m
RESET := \033[0m

.DEFAULT_GOAL := help

# ==================== Testing ====================

.PHONY: test-unit
test-unit:
	@docker compose exec server make test-unit-inner

.PHONY: test-unit-inner
test-unit-inner:
	@echo "Running unit tests..."
	@go test -v -race -count=1 -timeout 60s ./internal/...
	@echo "Unit tests completed"
 
.PHONY: test-coverage
test-coverage:
	@docker compose exec server make test-coverage-inner

.PHONY: test-coverage-inner
test-coverage-inner:
	@echo "Running unit tests with coverage..."
	@go test -race -count=1 -timeout 60s \
		-coverprofile=coverage.out \
		-covermode=atomic \
		./internal/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: $(CYAN)coverage.html$(RESET)"

# ==================== Migrations ====================

.PHONY: db-migrate-create
db-migrate-create:
	@docker compose exec server make db-migrate-create-inner $(filter-out $@,$(MAKECMDGOALS))

.PHONY: db-migrate-create-inner
db-migrate-create-inner:
	@echo "Creating migration file..."
	@migrate create -seq -ext sql -dir $(MIGRATIONS_PATH) $(filter-out $@,$(MAKECMDGOALS))
	@echo "Migration file created successfully"

.PHONY: db-migrate-up
db-migrate-up:
	@docker compose exec server make db-migrate-up-inner

.PHONY: db-migrate-up-inner
db-migrate-up-inner:
	@echo "Applying migrations..."
	@migrate -path $(MIGRATIONS_PATH) -database $(DB_ADDRESS) up
	@echo "Migrations applied successfully"

.PHONY: db-migrate-down
db-migrate-down:
	@docker compose exec server make db-migrate-down-inner

.PHONY: db-migrate-down-inner
db-migrate-down-inner:
	@echo "Rolling back migrations..."
	@migrate -path $(MIGRATIONS_PATH) -database $(DB_ADDRESS) down
	@echo "Migrations rolled back successfully"

.PHONY: db-test-migrate-up
db-test-migrate-up:
	@docker compose exec server make db-test-migrate-up-inner

.PHONY: db-test-migrate-up-inner
db-test-migrate-up-inner:
	@echo "Applying migrations to test database..."
	@migrate -path $(MIGRATIONS_PATH) -database $(DB_TEST_ADDRESS) up
	@echo "Migrations applied successfully"

.PHONY: db-test-migrate-down
db-test-migrate-down:
	@docker compose exec server make db-test-migrate-down-inner

.PHONY: db-test-migrate-down-inner
db-test-migrate-down-inner:
	@echo "Rolling back migrations to test database..."
	@migrate -path $(MIGRATIONS_PATH) -database $(DB_TEST_ADDRESS) down
	@echo "Migrations rolled back successfully"

# ==================== Database Seeding ====================

.PHONY: db-seed
db-seed:
	@docker compose exec server make db-seed-inner

.PHONY: db-seed-inner
db-seed-inner:
	@go run ./cmd/db/seed

.PHONY: db-flush
db-flush:
	@docker compose exec server make db-flush-inner

.PHONY: db-flush-inner
db-flush-inner:
	@go run ./cmd/db/seed -flush
 
# ==================== Cache ====================

.PHONY: cache-flush
cache-flush:
	@make cache-flush-inner

.PHONY: cache-flush-inner
cache-flush-inner:
	@echo "Flushing Redis cache..."
	@docker exec -e REDISCLI_AUTH=$(REDIS_PASSWORD) redis redis-cli FLUSHDB
	@echo "Redis cache flushed successfully"


# ==================== Swagger ====================

.PHONY: swagger
swagger:
	@docker compose exec server make swagger-inner

.PHONY: swagger-inner
swagger-inner:
	@echo "Generating Swagger documentation..."
	@swag init -g main.go \
		-d ./cmd/api/,./internal/handlers,./internal/dtos,./internal/domain,./internal/httpx,./internal/infrastructure/validator \
		-o ./docs \
		&& swag fmt
	@echo "Swagger documentation generated successfully"

# ==================== Help ====================

.PHONY: help
help:
	@echo "═══════════════════════════════════════════════════════════════"
	@echo "  Social API - Makefile Commands"
	@echo "═══════════════════════════════════════════════════════════════"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  make test-unit               - Run unit tests (wrapper)"
	@echo "  make test-coverage           - Run tests with coverage (wrapper)"
	@echo ""
	@echo "💾 Database Seeding:"
	@echo "  make db-seed                - Seed database with test data (wrapper)"
	@echo "  make db-flush               - Remove all data from database (wrapper)"
	@echo ""
	@echo "🔄 Database Migrations:"
	@echo "  make db-migrate-create      - Create new migration file (wrapper)"
	@echo "  make db-migrate-up          - Apply all pending migrations (wrapper)"
	@echo "  make db-migrate-down        - Rollback last migration (wrapper)"
	@echo "  make db-test-migrate-up     - Apply test DB migrations (wrapper)"
	@echo "  make db-test-migrate-down   - Rollback test DB migrations (wrapper)"
	@echo ""
	@echo "🧹 Cache:"
	@echo "  make cache-flush            - Flush Redis cache (wrapper)"
	@echo ""
	@echo "📚 Documentation:"
	@echo "  make swagger                - Generate Swagger docs (wrapper)"
	@echo ""
	@echo "═══════════════════════════════════════════════════════════════"

# Catch-all target to prevent make from treating migration names as targets
%:
	@: