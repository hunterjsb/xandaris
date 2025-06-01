#!/bin/bash

# Reset script for Xandaris - cleans all game data and reseeds
# Preserves user accounts and authentication

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BACKEND_DIR="$SCRIPT_DIR/.."

echo "🔄 Xandaris Reset Script"
echo "========================"
echo "This will:"
echo "  ✅ Preserve all user accounts and authentication"
echo "  🗑️  Delete all game data (systems, planets, etc.)"
echo "  🌱 Regenerate the universe with fresh data"
echo ""

# Build utilities if needed
cd "$BACKEND_DIR"

echo "📦 Building utilities..."
go build -o util cmd/util/main.go
go build -o seed cmd/seed/main.go

echo ""
echo "🧹 Cleaning game data..."
./util clean

echo ""
echo "🌱 Seeding new universe..."
./seed

echo ""
echo "✅ Reset complete!"
echo ""
echo "📊 Final status:"
./util check