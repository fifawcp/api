include .env
MIGRATIONS_PATH = ./cmd/db/migrations

.DEFAULT_GOAL := help

# ==================== Development ====================

.PHONY: dev
dev:
	@echo "Starting API with hot reload (air)..."
	@air

# ==================== Migrations ====================

.PHONY: migration
migration:
	@echo "Creating migration file..."
	@migrate create -seq -ext sql -dir $(MIGRATIONS_PATH) $(filter-out $@,$(MAKECMDGOALS))
	@echo "Migration file created successfully"

.PHONY: migrate-up
migrate-up:
	@echo "Applying migrations..."
	@migrate -path $(MIGRATIONS_PATH) -database $(DB_ADDRESS) up
	@echo "Migrations applied successfully"

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back migrations..."
	@migrate -path $(MIGRATIONS_PATH) -database $(DB_ADDRESS) down
	@echo "Migrations rolled back successfully"

.PHONY: migrate-test-up
migrate-test-up:
	@echo "Applying migrations to test database..."
	@migrate -path $(MIGRATIONS_PATH) -database $(DB_TEST_ADDRESS) up
	@echo "Migrations applied successfully"

.PHONY: migrate-test-down
migrate-test-down:
	@echo "Rolling back migrations to test database..."
	@migrate -path $(MIGRATIONS_PATH) -database $(DB_TEST_ADDRESS) down
	@echo "Migrations rolled back successfully"

# ==================== Database ====================

.PHONY: db-seed
db-seed:
	@go run ./cmd/db/seed

.PHONY: db-flush
db-flush:
	@go run ./cmd/db/seed -flush
 
# ==================== Cache ====================

.PHONY: cache-flush
cache-flush:
	@echo "Flushing Redis cache..."
	@docker exec -e REDISCLI_AUTH=$(REDIS_PASSWORD) redis redis-cli FLUSHDB
	@echo "Redis cache flushed successfully"

# ==================== Docker & Project Setup ====================

.PHONY: docker-up
docker-up:
	@echo "Starting all Docker services..."
	@docker compose up -d
	@echo "Services started:"
	@echo "	- PostgreSQL (dev):  localhost:5432"
	@echo "	- PostgreSQL (test): localhost:5433"
	@echo "	- Redis:             localhost:6379"
	@echo "	- Redis Commander:   \033[36mhttp://localhost:8081\033[0m"

.PHONY: docker-down
docker-down:
	@echo "Stopping all Docker services..."
	@docker compose down
	@echo "All services stopped"

.PHONY: docker-restart
docker-restart:
	@echo "Restarting all Docker services..."
	@docker compose restart
	@echo "Services restarted"

# ==================== Swagger ====================
 
.PHONY: swagger
swagger:
	@echo "Generating Swagger documentation..."
	@swag init -g main.go \
		-d ./cmd/api/,./internal/handlers,./internal/dtos,./internal/domain,./internal/packages/httputils \
		-o ./docs/swagger \
		&& swag fmt
	@echo "Swagger documentation generated successfully"

# ==================== Help ====================

.PHONY: help
help:
	@echo "═══════════════════════════════════════════════════════════════"
	@echo "  Social API - Makefile Commands"
	@echo "═══════════════════════════════════════════════════════════════"
	@echo ""
	@echo "💻 Development:"
	@echo "  make dev               - Run API with hot reload"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  make docker-up         - Start all services (DB, Redis)"
	@echo "  make docker-down       - Stop all services"
	@echo "  make docker-restart    - Restart all services"
	@echo ""
	@echo "💾 Database:"
	@echo "  make db-seed           - Seed database with test data"
	@echo "  make db-flush          - Remove all data from database"
	@echo ""
	@echo "🔄 Migrations:"
	@echo "  make migration         - Create new migration file"
	@echo "  make migrate-up        - Apply all pending migrations"
	@echo "  make migrate-down      - Rollback last migration"
	@echo "  make migrate-test-up   - Apply all pending migrations in test db"
	@echo "  make migrate-test-down - Rollback last migration in test db"
	@echo ""
	@echo "📚 Documentation:"
	@echo "  make swagger           - Generate Swagger docs"
	@echo ""
	@echo "═══════════════════════════════════════════════════════════════"

# Catch-all target to prevent make from treating migration names as targets
%:
	@: 