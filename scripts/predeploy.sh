#!/bin/sh
set -e

echo "Running migrations..."
/migrate -path /migrations -database "$DB_ADDRESS" -verbose up

if [ "$SEED_ON_DEPLOY" = "true" ]; then
  echo "Seeding database..."
  /seed
fi

echo "Pre-deploy complete."
