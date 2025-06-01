#!/bin/bash

# Script to safely restart the Xandaris backend server

set -e

cd "$(dirname "$0")/.."

echo "Building Xandaris backend..."
go build -o xandaris cmd/main.go

echo "Stopping existing server..."
pkill -f "xandaris serve" || true
sleep 2

echo "Starting server in background..."
nohup ./xandaris serve --dev --dir=pb_data > server.log 2>&1 &
SERVER_PID=$!

echo "Server started with PID: $SERVER_PID"
echo $SERVER_PID > server.pid

# Wait a moment and check if server is running
sleep 3
if kill -0 $SERVER_PID 2>/dev/null; then
    echo "✅ Server is running successfully!"
    echo "📋 Logs: tail -f backend/server.log"
    echo "🛑 Stop: kill \$(cat backend/server.pid)"
else
    echo "❌ Server failed to start. Check backend/server.log for errors."
    exit 1
fi