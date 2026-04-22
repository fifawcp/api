# FIFA World Cup Prediction Platform - API

Backend REST API for the FIFA World Cup prediction platform. Provides passwordless authentication, user management, and board functionality for competing with friends.

## Overview

This API handles authentication, user management, and private board creation for the FIFA World Cup prediction platform. It uses a clean architecture pattern with Go, PostgreSQL, and Redis.

## Tech Stack

- Go
- PostgreSQL
- Redis
- Docker

## Architecture

Clean architecture with clear separation of concerns:

```md
cmd/
├── api/            # Application entry point
└── db/             # Database migrations
internal/
├── app/            # Dependency injection container
├── domain/         # Business entities and errors
├── dtos/           # Request/response DTOs with validation
├── handlers/       # HTTP handlers
├── infrastructure/ # Infrastructure layer
├── jobs/           # Background jobs
├── packages/       # Reusable utilities
├── repositories/   # Database access layer
├── services/       # Business logic layer
└── storage/        # Cache access layer
```

## Installation

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- GNU Make (optional, for command shortcuts)

### Setup

1. **Clone the repository**

      ```bash
      git clone https://github.com/fifawcp/api.git
      cd api
      ```

2. **Configure environment variables**

      The application loads configuration from environment variables. See `internal/infrastructure/config/config.go` for all available options.

      Create a `.env` file in the project root with the following required variables:

      ```env
      PORT=8080

      # Database
      DB_ADDRESS=postgres://postgres:password@db:5432/fifawcp?sslmode=disable
      DB_TEST_ADDRESS=postgres://postgres:password@test-db:5432/fifawcp_test?sslmode=disable

      # Redis
      REDIS_PASSWORD=password
      REDIS_ADDRESS=redis:6379

      # CORS
      CORS_ALLOWED_ORIGINS=http://localhost:3000

      # Email (Resend)
      MAILER_API_KEY=your_resend_api_key
      MAILER_FROM_ADDRESS=noreply@yourdomain.com
      ```

      _Note: Replace all placeholder values with your actual configuration._
      
      _Docker note: `db`, `test-db`, and `redis` are Docker Compose service names. Inside containers, use these hostnames instead of `localhost`._

3. **Start Docker services**

      ```bash
      docker compose up --build -d
      ```

      This starts:
      - API Server: `http://localhost:8080`
      - PostgreSQL (dev): `localhost:5432`
      - PostgreSQL (test): `localhost:5433`
      - Redis: `localhost:6379`
      - Redis Commander: `http://localhost:8081`

4. **Run database migrations**

      ```bash
      make db-migrate-up
      ```

      _Windows users: see `Running Commands` -> `Windows` for the direct `docker compose exec ...` equivalent._

### Shutdown

Stop all services:

```bash
docker compose down
```

### Clean setup (reset everything)

If your local state is broken and you want a fully clean environment:

```bash
docker compose down -v
docker compose up --build -d
make db-migrate-up
```

## Running the Project

The API will be available at `http://localhost:8080`

_Note: Development is Docker-only. The API runs inside the `server` compose service using `Dockerfile.dev` and `.air.docker.toml`._

## Running Commands

### macOS/Linux

Use `make` shortcuts from the project root:

```bash
make help
make test-unit
make swagger
...
```

### Windows

If you do not use `make` on Windows, run the equivalent command directly from `Makefile`.

`Makefile` target:

```make
db-migrate-up:
	@docker compose exec server make db-migrate-up-inner
```

Equivalent command you can run directly:

```bash
docker compose exec server make db-migrate-up-inner
```

More examples:

```bash
# make test-unit
docker compose exec server make test-unit-inner

# make swagger
docker compose exec server make swagger-inner
```

## Testing

```bash
# Run all unit tests
make test-unit

# Run tests with coverage report
make test-coverage
```

## API Documentation

Once the server is running, access the Swagger documentation at:

[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

## Useful Commands

For a complete list of available commands, run:

```bash
make help
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b 'API-<<issue-number>>'`)

   _Note: Replace `<issue-number>` with the actual GitHub issue number._
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)

   _Note: Use conventional commits: `feat:`, `fix:`, `chore:`, `docs:`, etc._
4. Push to the branch (`git push origin API-<<issue-number>>`)
5. Open a Pull Request to `develop`
