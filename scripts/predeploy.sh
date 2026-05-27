#!/bin/sh
set -e

echo "Running migrations..."
/migrate -path /migrations -database "$DB_ADDRESS" -verbose up

if [ -n "$SEED_SCENARIO" ]; then
  echo "Seeding database with scenario: $SEED_SCENARIO"
  /seed -scenario="$SEED_SCENARIO"
fi

echo "Pre-deploy complete."
