# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview
XANDARIS is a real-time 4X space strategy game featuring complex colony management, resource economics, and fleet logistics. The project consists of:
- **Frontend**: Vanilla JS + Canvas + Tailwind CSS (port 5173)
- **Backend**: Go + PocketBase + SQLite (port 8090)
- **Architecture**: Relational database with 12+ collections supporting multi-planet colonies

## Development Workflow
- Create branches using the pattern `<issue-number>-<short-description>`
- Example: `15-population-happiness-mechanics`
- PRs should include a summary of changes and a test plan
- When done, merge to master, pull changes, then delete the feature branch

## Build/Run Commands

### Backend (Go + PocketBase)
- Navigate to backend: `cd backend`
- Install dependencies: `go mod tidy`
- Build server: `go build -o xandaris cmd/main.go`
- Run server: `./xandaris serve --dev --dir=pb_data`
- **Restart server**: `./scripts/restart.sh` (builds, stops, and starts server)
- Run migrations: `./xandaris migrate up`

### Database Management
- Seed universe: `go build -o seed cmd/seed/main.go && ./seed`
- Check colonies: `go build -o util cmd/util/main.go && ./util check`
- Reset colonies: `./util reset`
- Clean data: `./util clean`

### Frontend (Vanilla JS + Vite)
- Install: `npm install`
- Dev server: `npm run dev`
- Build: `npm run build`
- Preview: `npm run preview`

## Key Components

### Backend Architecture
- **PocketBase**: Authentication, database, and REST API
- **Game Tick System**: 10-second real-time economy simulation
- **Relational Schema**: 12+ collections with proper foreign keys
- **Custom API**: `/api/map`, `/api/orders/*`, `/api/status`
- **Performance**: Smart processing - only planets with populations

### Frontend Architecture
- **Canvas Renderer**: `src/components/mapRenderer.js` - Galaxy visualization
- **PocketBase Client**: `src/lib/pocketbase.js` - Auth and data sync
- **Game State**: `src/stores/gameState.js` - Centralized state management
- **UI Controller**: `src/components/uiController.js` - Modal and UI management

### Game Mechanics (4X Strategy)
- **eXplore**: Star system discovery
- **eXpand**: Planet colonization with populations
- **eXploit**: Resource extraction via buildings + population + resource nodes
- **eXterminate**: Fleet-based combat and conquest

## Database Schema (Relational 4X Design)

### Core Collections
- **systems**: x, y coordinates, discovered_by
- **planets**: system_id, planet_type, size, colonized_by, colonized_at
- **populations**: owner_id, planet_id, count, happiness, employed_at
- **buildings**: planet_id, building_type, level, active, resource_nodes
- **fleets**: owner_id, current_system, destination_system, eta
- **ships**: fleet_id, ship_type, count, health
- **trade_routes**: owner_id, from_system, to_system, resource_type, active
- **resource_nodes**: planet_id, resource_type, richness, exhausted

### Reference Collections
- **planet_types**: name, base_max_population, habitability
- **building_types**: name, cost, worker_capacity, max_level
- **ship_types**: name, cost, strength, cargo_capacity
- **resource_types**: name, description, is_consumable

### Key Relationships
- Systems contain multiple planets
- Planets have resource nodes and buildings
- Buildings employ population to work resource nodes
- Fleets contain ships and transport population
- Trade routes connect systems for automated cargo

## Game Tick Processing (Every 10 seconds)
1. **Economy**: Process colonized planets with populations
   - Buildings produce resources based on employed population
   - Population happiness affects production efficiency
   - Resource consumption (food, fuel) affects population
   - Population growth/decline based on resources + happiness
2. **Buildings**: Apply construction completion
3. **Trade**: Move cargo along active routes
4. **Fleets**: Resolve arrivals and combat
5. **Diplomacy**: Process treaty changes

## Authentication
- **Discord OAuth2**: Primary authentication method
- **Collection Access Rules**: Users can access their own game data
- **PocketBase Auth**: Built-in user management and sessions

## API Endpoints

### Game Data (Custom)
- `GET /api/map` - Systems, planets, lanes with aggregated data
- `GET /api/systems` - Star systems with colony information
- `GET /api/status` - Game tick status and server information

### Player Actions (Custom)
- `POST /api/orders/fleet` - Deploy fleet between systems
- `POST /api/orders/build` - Queue building construction
- `POST /api/orders/trade` - Create automated trade route

### Collection Access (PocketBase Auto-generated)
- `GET /api/collections/fleets/records` - User's fleets
- `GET /api/collections/buildings/records` - Buildings data
- `GET /api/collections/populations/records` - User's populations
- `GET /api/collections/planets/records` - Planet information

### Real-time
- `WS /api/stream` - WebSocket for live game updates

## Environment Variables
- `VITE_POCKETBASE_URL`: Backend URL (default: http://localhost:8090)
  - For development: http://localhost:8090
  - For production: https://api.xan-nation.com

## File Structure
```
backend/
├── cmd/                    # Go executables
│   ├── main.go            # PocketBase server with game logic
│   ├── seed/main.go       # Universe generation utility
│   └── util/main.go       # Database management utilities
├── internal/              # Go internal packages
│   ├── tick/              # Game tick processing
│   ├── economy/           # Economic simulation
│   ├── map/               # Map generation
│   └── websocket/         # Real-time updates
├── migrations/            # PocketBase schema migrations
├── pkg/                   # API handlers
└── pb_data/               # PocketBase database and files

frontend/
├── src/                   # Frontend source
│   ├── lib/               # PocketBase client and utilities
│   ├── stores/            # State management
│   ├── components/        # Canvas renderer and UI
│   └── main.js            # Application entry point
└── static/                # Static assets
```

## Production Deployment
- **Single binary**: `xandaris` contains everything
- **Static files**: Frontend builds integrated into backend
- **Database**: SQLite file in `/pb_data`
- **Reverse proxy**: Caddy serves static files and proxies API
- **Docker**: Single container deployment with minimal resources

## Performance Considerations
- **Smart Economy**: Only processes planets with actual populations
- **Efficient Queries**: Indexed relationships for fast lookups
- **Real-time Updates**: WebSocket for instant frontend updates
- **Canvas Rendering**: Optimized galaxy visualization
- **Minimal Resources**: Runs efficiently on t3.micro

## Discord Setup
To enable Discord OAuth:
1. Create Discord application at https://discord.com/developers/applications
2. Add OAuth2 redirect: `http://localhost:8090/_/redirect/discord`
3. Get Client ID and Client Secret from Discord
4. Configure in PocketBase admin panel under Settings > Auth providers > Discord

## Common Development Tasks

### Adding New Game Mechanics
1. Update database schema in `migrations/`
2. Implement logic in `internal/` packages
3. Add API endpoints in `pkg/`
4. Update frontend in `src/`

### Database Changes
1. Create migration: `./xandaris migrate create "description"`
2. Apply migration: `./xandaris migrate up`
3. Reset if needed: `./util clean && ./seed`
4. After code changes: `./scripts/restart.sh` (auto-builds and restarts)

### Debugging Economy
- Check colony distribution: `./util check`
- Monitor tick processing: Watch server logs for "Updated economy"
- Reset to known state: `./util reset`

## Troubleshooting

### Common Issues
- **1438 colonized planets**: Run `./util reset` to fix unrealistic data
- **Frontend schema errors**: Ensure collections have proper access rules
- **Auto-cancellation errors**: Frontend handles these automatically
- **Empty economy processing**: Economy skips planets without populations

### Log Messages to Monitor
- `Updated economy for X planets with populations (skipped Y empty colonies)`
- `Game tick completed in Xms`
- `Resolved X fleet arrivals`
- `WebSocket client connected`