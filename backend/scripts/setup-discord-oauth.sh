#!/bin/bash

# Script to configure Discord OAuth via PocketBase API
# This does exactly what the admin UI does

set -e

# Load environment variables from .env if present
if [ -f .env ]; then
    set -o allexport
    source .env
    set +o allexport
fi

# Check required environment variables
if [ -z "$DISCORD_CLIENT_ID" ] || [ -z "$DISCORD_CLIENT_SECRET" ]; then
    echo "❌ Error: DISCORD_CLIENT_ID and DISCORD_CLIENT_SECRET must be set in .env"
    exit 1
fi

if [ -z "$SUPERUSER_EMAIL" ] || [ -z "$SUPERUSER_PASSWORD" ]; then
    echo "❌ Error: SUPERUSER_EMAIL and SUPERUSER_PASSWORD must be set in .env"
    exit 1
fi

echo "🔧 Setting up Discord OAuth..."

# Login as admin to get auth token
echo "🔑 Authenticating as admin..."
AUTH_RESPONSE=$(curl -s -X POST 'http://127.0.0.1:8090/api/admins/auth-with-password' \
  -H 'Content-Type: application/json' \
  -d "{\"identity\":\"$SUPERUSER_EMAIL\",\"password\":\"$SUPERUSER_PASSWORD\"}")

# Extract token from response
TOKEN=$(echo "$AUTH_RESPONSE" | python3 -c "import sys, json; print(json.loads(sys.stdin.read())['token'])" 2>/dev/null)

if [ -z "$TOKEN" ]; then
    echo "❌ Failed to authenticate. Make sure the server is running and superuser exists."
    echo "Response: $AUTH_RESPONSE"
    exit 1
fi

echo "✅ Authenticated successfully"

# Configure Discord OAuth via settings API
echo "🔧 Configuring Discord OAuth..."
OAUTH_RESPONSE=$(curl -s -X PATCH 'http://127.0.0.1:8090/api/settings' \
  -H "Authorization: $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"discordAuth\":{\"enabled\":true,\"clientId\":\"$DISCORD_CLIENT_ID\",\"clientSecret\":\"$DISCORD_CLIENT_SECRET\",\"authUrl\":\"\",\"tokenUrl\":\"\",\"userApiUrl\":\"\",\"displayName\":\"\",\"pkce\":null}}")

# Check if successful
if echo "$OAUTH_RESPONSE" | grep -q "discordAuth"; then
    echo "✅ Discord OAuth configured successfully!"
    echo "🎮 Discord authentication is now enabled"
else
    echo "❌ Failed to configure Discord OAuth"
    echo "Response: $OAUTH_RESPONSE"
    exit 1
fi