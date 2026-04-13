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

# Catch-all target to prevent make from treating migration names as targets
%:
	@: 