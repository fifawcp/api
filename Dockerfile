# ---- Build Stage ----
# Use the official Go Alpine image to compile the binary.
# Alpine keeps the builder lean; it is NOT included in the final image.
FROM golang:1.26-alpine AS builder
WORKDIR /app

# Install CA certificates and git.
# - ca-certificates: required to copy TLS certs into the scratch image.
# - git: required by some Go module dependencies during go mod download.
RUN apk add --no-cache ca-certificates git

# Copy dependency files first to leverage Docker layer caching.
# go mod download only re-runs when go.mod or go.sum change.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Install swag to generate Swagger docs before building.
# Required because cmd/api imports the generated docs package.
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Install migrate for running database migrations at deploy time.
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Copy the rest of the source code.
COPY . .

# Generate Swagger documentation from handler, DTO, and domain annotations.
# Must run before go build since the docs package is imported by cmd/api.
RUN swag init -g main.go \
    -d ./cmd/api/,./internal/handlers,./internal/dtos,./internal/domain,./internal/packages/httputils \
    -o ./docs

# Build the binary.
# CGO_ENABLED=0 produces a fully static binary compatible with scratch.
# -ldflags="-s -w" strips debug symbols and DWARF info, reducing binary size.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o api \
    ./cmd/api/

# ---- Run Stage ----
# alpine is minimal but includes a shell, required for Railway's pre-deploy command.
FROM alpine:3.21
WORKDIR /app

# Install CA certificates for outbound HTTPS calls.
RUN apk add --no-cache ca-certificates

# Copy the compiled binary from the build stage.
COPY --from=builder /app/api .

# Copy the migrate binary and migrations for pre-deploy migrations.
COPY --from=builder /go/bin/migrate /migrate
COPY --from=builder /app/cmd/db/migrations /migrations

# Cloud Run injects the PORT env var and routes traffic to it.
# 8080 is the default expected port.
EXPOSE 8080

CMD ["./api"]
