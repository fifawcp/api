# FIFA World Cup Prediction Platform - API

Backend REST API for the FIFA World Cup prediction platform. Provides passwordless authentication, user management, and board functionality for competing with friends.

## Overview

This API handles authentication, user management, and private board creation for the FIFA World Cup prediction platform. It uses a clean architecture pattern with Go, PostgreSQL, and Redis.

## Tech Stack

- Go
- PostgreSQL
- Redis

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

- [Go 1.26.2 or higher](https://go.dev/dl/)
- [Docker and Docker Compose](https://docs.docker.com/get-docker/)
- Make (usually pre-installed on macOS/Linux)

### Setup

1. **Clone the repository**

      ```bash
      git clone https://github.com/fifawcp/api.git
      cd api
      ```

2. **Start Docker services**

      ```bash
      make docker-up
      ```

      This starts:
      - PostgreSQL (dev): `localhost:5432`
      - PostgreSQL (test): `localhost:5433`
      - Redis: `localhost:6379`
      - Redis Commander: `http://localhost:8081`

3. **Configure environment variables**

      The application loads configuration from environment variables. See `internal/infrastructure/config/config.go` for all available options.

      Create a `.env` file in the project root with the following required variables:

      ```env
      # Database
      DB_ADDRESS=postgres://postgres:password@localhost:5432/pickems?sslmode=disable
      DB_TEST_ADDRESS=postgres://postgres:password@localhost:5433/pickems_test?sslmode=disable

      # Redis
      REDIS_ADDRESS=localhost:6379
      REDIS_PASSWORD=your_redis_password

      # JWT
      JWT_SECRET=your_jwt_secret_key
      JWT_ACCESS_TOKEN_EXPIRY=15m

      # CORS
      CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

      # Email (Resend)
      MAILER_API_KEY=your_resend_api_key
      MAILER_FROM_ADDRESS=noreply@yourdomain.com
      ```

      _Note: Replace all placeholder values with your actual configuration._

4. **Run database migrations**

      ```bash
      make migrate-up
      ```

## Running the Project

### Development Mode (with hot reload)

```bash
make dev
```

The API will be available at `http://localhost:8080`

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
