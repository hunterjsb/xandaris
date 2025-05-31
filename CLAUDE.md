# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview
XANDARIS is a numbers-only 4X-lite space strategy game that runs on a single AWS t3.micro. The project consists of:
- **Frontend**: Vanilla JS + Canvas + Tailwind CSS (port 5173)
- **Backend**: Go + PocketBase + SQLite (port 8090)

## Development Workflow
- Create branches using the pattern `<issue-number>-<short-description>`
- Example: `5-combat-system-improvements`
- PRs should include a summary of changes and a test plan
- When done, merge to master, pull changes, then delete the feature branch

## Build/Run Commands

### Backend (Go + PocketBase)
- Navigate to backend: `cd backend`
- Install dependencies: `go mod tidy`
- Build: `go build -o xandaris cmd/main.go`
- Run server: `./xandaris serve --dev --dir=pb_data`
- Run migrations: `./xandaris migrate up`
- Seed initial data: `go run cmd/seed/main.go`
- Test: `go test ./...`

### Frontend (Vanilla JS + Vite)
- Install: `npm install`
- Dev server: `npm run dev`
- Build: `npm run build`
- Preview: `npm run preview`

## Key Components

### Backend Architecture
- **PocketBase**: Authentication, database, and REST API
- **Game Tick System**: Hourly cron job for economy/fleet processing
- **Collections**: `systems`, `fleets`, `trade_routes`, `treaties`, `users`
- **Custom API**: `/api/map`, `/api/orders/*`, `/api/diplomacy`

### Frontend Architecture
- **Canvas Renderer**: `src/components/mapRenderer.js` - Main game view
- **PocketBase Client**: `src/lib/pocketbase.js` - Auth and data sync
- **Game State**: `src/stores/gameState.js` - Centralized state management
- **UI Controller**: `src/components/uiController.js` - Modal and UI management

### Game Mechanics
- **Systems**: Colonize and develop star systems with buildings
- **Resources**: Food, ore, goods, fuel (produced by buildings)
- **Fleets**: Send military forces between systems (2hr travel time)
- **Trade Routes**: Automated cargo transport between owned systems
- **Economy**: Production/consumption simulation every tick
- **Combat**: Simple strength-based system for fleet vs population

## Database Schema

### Collections
- **systems**: x, y, richness, owner_id, pop, morale, resources, building levels
- **fleets**: owner_id, from_id, to_id, eta, strength
- **trade_routes**: owner_id, from_id, to_id, cargo, capacity, eta
- **treaties**: type, a_id, b_id, created_at, expires_at, status
- **users**: username, color, alliance_id, last_seen (+ Discord OAuth)

### Indexes
- Systems: (x,y), owner_id
- Fleets: owner_id, eta
- Trade routes: owner_id, eta
- Treaties: (a_id, b_id), status

## Game Tick Processing (Hourly)
1. **Economy**: Update production/consumption for all systems
2. **Buildings**: Apply completion (instant for MVP)
3. **Trade**: Move cargo along active routes
4. **Fleets**: Resolve arrivals and combat
5. **Diplomacy**: Expire old treaties

## Authentication
- **Discord OAuth2**: Primary authentication method
- **Token-based**: Frontend uses Bearer tokens for API calls
- **PocketBase Auth**: Built-in user management and sessions

## API Endpoints
- `GET /api/map` - Return systems and lanes data
- `POST /api/orders/fleet` - Send fleet between systems
- `POST /api/orders/build` - Queue building construction
- `POST /api/orders/trade` - Create trade route
- `POST /api/diplomacy` - Propose treaty
- `WS /api/stream` - Real-time updates (TODO)

## Environment Variables
- `VITE_POCKETBASE_URL`: Backend URL (default: http://localhost:8090)
  - For development: http://localhost:8090
  - For production: https://api.xan-nation.com

## File Structure
```
/cmd/                   # Go executables
  main.go              # PocketBase server with game logic
  seed.go              # Map generation utility
/internal/             # Go internal packages
  tick/                # Game tick processing
  economy/             # Economic simulation
  map/                 # Map generation and data
/migrations/           # PocketBase schema migrations
/pkg/                  # Go API handlers
/src/                  # Frontend source
  lib/                 # PocketBase client and utilities
  stores/              # State management
  components/          # Canvas renderer and UI
/static/               # Static assets
/pb_data/              # PocketBase database and files
```

## Production Deployment
- **Single binary**: `xan-nation` contains everything
- **Static files**: Frontend builds to `/web` directory
- **Database**: SQLite file in `/pb_data`
- **Reverse proxy**: Caddy serves static files and proxies API
- **Docker**: Single container deployment

## Discord Setup
To enable Discord OAuth:
1. Create Discord application at https://discord.com/developers/applications
2. Add OAuth2 redirect: `http://localhost:8090/_/redirect/discord`
3. Get Client ID and Client Secret from Discord
4. Configure in PocketBase admin panel under Settings > Auth providers > Discord