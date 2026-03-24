#!/bin/bash
# Tool: opens cc-track dashboard in browser, waits, then collects viewport data + screenshot
# Usage: .claude/tools/browser-check.sh [port]

PORT=${1:-8099}
PROJECT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
SCREENSHOT="/tmp/cc-track-dashboard.png"

# Build
cd "$PROJECT_DIR"
go build -o cc-track . 2>&1

# Kill any existing instance on this port
lsof -ti:$PORT | xargs kill -9 2>/dev/null
sleep 0.5

# Start server in background
./cc-track serve -p $PORT &
SERVER_PID=$!
sleep 1

# Open browser
open "http://localhost:$PORT"

# Wait for page to load and report viewport
echo "Waiting for browser to report viewport..."
sleep 4

# Take screenshot
screencapture -l$(osascript -e 'tell app "Google Chrome" to id of window 1' 2>/dev/null || osascript -e 'tell app "Safari" to id of window 1' 2>/dev/null || echo "0") "$SCREENSHOT" 2>/dev/null
# Fallback: full screen capture if window capture fails
if [ ! -f "$SCREENSHOT" ] || [ ! -s "$SCREENSHOT" ]; then
  screencapture -x "$SCREENSHOT" 2>/dev/null
fi

# Fetch viewport data
echo "=== Viewport Data ==="
curl -s "http://localhost:$PORT/api/viewport" 2>/dev/null | python3 -m json.tool 2>/dev/null || curl -s "http://localhost:$PORT/api/viewport"
echo ""
echo "=== Screenshot ==="
echo "$SCREENSHOT"

# Stop server
kill $SERVER_PID 2>/dev/null
