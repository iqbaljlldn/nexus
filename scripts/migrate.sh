#!/usr/bin/env bash
# ==============================================================================
# Nexus Database Migration Runner
# ==============================================================================
# Usage:
#   ./scripts/migrate.sh up           — Apply all pending migrations
#   ./scripts/migrate.sh down         — Rollback last migration
#   ./scripts/migrate.sh down-all     — Rollback ALL migrations
#   ./scripts/migrate.sh status       — Show current migration version
#   ./scripts/migrate.sh create NAME  — Create new migration files
#   ./scripts/migrate.sh force V      — Force set version (for fixing dirty state)
#
# Environment:
#   NEXUS_API_DB_URL  — PostgreSQL connection URL (default: localhost dev)
# ==============================================================================

set -euo pipefail

MIGRATIONS_DIR="$(cd "$(dirname "$0")/.." && pwd)/migrations"
DB_URL="${NEXUS_API_DB_URL:-postgres://postgres:postgres@localhost:5434/nexus?sslmode=disable}"

# Verify migrate CLI is installed
if ! command -v migrate &>/dev/null; then
    echo "Error: 'migrate' CLI not found."
    echo "Install: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    exit 1
fi

ACTION="${1:-}"

case "$ACTION" in
    up)
        echo "▶ Running migrations UP..."
        migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" up
        echo "✅ Migrations applied successfully."
        ;;
    down)
        echo "▶ Rolling back last migration..."
        migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" down 1
        echo "✅ Rolled back 1 migration."
        ;;
    down-all)
        echo "▶ Rolling back ALL migrations..."
        migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" down -all
        echo "✅ All migrations rolled back."
        ;;
    status)
        echo "▶ Current migration version:"
        migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" version
        ;;
    create)
        NAME="${2:-}"
        if [ -z "$NAME" ]; then
            echo "Error: Migration name required."
            echo "Usage: ./scripts/migrate.sh create <migration_name>"
            exit 1
        fi
        TIMESTAMP=$(date +%Y%m%d%H%M%S)
        UP_FILE="${MIGRATIONS_DIR}/${TIMESTAMP}_${NAME}.up.sql"
        DOWN_FILE="${MIGRATIONS_DIR}/${TIMESTAMP}_${NAME}.down.sql"
        touch "$UP_FILE" "$DOWN_FILE"
        echo "✅ Created migration files:"
        echo "   $UP_FILE"
        echo "   $DOWN_FILE"
        ;;
    force)
        VERSION="${2:-}"
        if [ -z "$VERSION" ]; then
            echo "Error: Version number required."
            echo "Usage: ./scripts/migrate.sh force <version>"
            exit 1
        fi
        echo "▶ Forcing migration version to $VERSION..."
        migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" force "$VERSION"
        echo "✅ Version forced to $VERSION."
        ;;
    *)
        echo "Nexus Database Migration Runner"
        echo ""
        echo "Usage: ./scripts/migrate.sh <command>"
        echo ""
        echo "Commands:"
        echo "  up            Apply all pending migrations"
        echo "  down          Rollback last migration"
        echo "  down-all      Rollback ALL migrations"
        echo "  status        Show current migration version"
        echo "  create NAME   Create new migration files"
        echo "  force V       Force set version (fix dirty state)"
        exit 1
        ;;
esac
