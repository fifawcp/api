#!/bin/sh
set -e

echo "Running migrations..."
/migrate -path /migrations -database "$DB_ADDRESS" -verbose up

# Seeding flushes data and is only for dev/qa. Omit it in production even if a
# seed var is set. The /seed binary independently refuses to run when
# ENV=production, so this is a graceful skip layered on top of that backstop.
# SEED_FROM_PROD (real prod data, read-only) takes precedence over SEED_SCENARIO.
seed_cmd=""
seed_desc=""
if [ "$SEED_FROM_PROD" = "true" ]; then
  seed_cmd="/seed -from-prod"
  seed_desc="live from prod snapshot (read-only)"
elif [ -n "$SEED_SCENARIO" ]; then
  seed_cmd="/seed -scenario=$SEED_SCENARIO"
  seed_desc="with scenario: $SEED_SCENARIO"
fi

if [ -n "$seed_cmd" ]; then
  # Match any casing, mirroring IsProd()'s EqualFold, so a prod deploy skips
  # gracefully instead of invoking /seed and hard-failing under set -e.
  case "$(printf '%s' "$ENV" | tr '[:upper:]' '[:lower:]')" in
    production|prod)
      echo "ENV=$ENV is production — skipping seed (dev only)"
      ;;
    *)
      echo "Seeding database $seed_desc"
      $seed_cmd
      ;;
  esac
fi

echo "Pre-deploy complete."
