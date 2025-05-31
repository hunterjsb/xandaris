# Xandaris - 4X Space Strategy Game

Xandaris is a real-time 4X space strategy game featuring complex colony management, resource economics, and fleet logistics. Built with a relational database architecture supporting multi-planet systems, building production chains, and population dynamics.

## Quick Start

1. **Backend Setup**:
   ```bash
   cd backend
   go mod tidy
   go build -o xandaris cmd/main.go
   ./xandaris serve --dev --dir=pb_data
   ```

2. **Database Setup** (first time):
   ```bash
   # Seed the game universe with systems, planets, and reference data
   go build -o seed cmd/seed/main.go
   ./seed
   ```

3. **Frontend Setup**:
   ```bash
   cd frontend
   npm install
   npm run dev
   ```

4. **Play**: Open <http://localhost:5173> in your browser

## Game Architecture

### 4X Strategy Elements
- **eXplore**: Discover star systems across the galaxy
- **eXpand**: Colonize planets and build infrastructure  
- **eXploit**: Extract resources and manage complex economies
- **eXterminate**: Deploy fleets for conquest and defense

### Core Gameplay Loop
1. **Colonize Planets** - Establish colonies on habitable worlds
2. **Build Infrastructure** - Construct farms, mines, factories, spaceports
3. **Manage Population** - Keep citizens happy and employed
4. **Extract Resources** - Work resource nodes for materials
5. **Build Fleets** - Construct ships for expansion and defense
6. **Trade & Diplomacy** - Establish trade routes and treaties

## Game Systems

### **Planets & Colonies**
- **Planet Types**: Terran, Arid, Ocean, Arctic, Volcanic, Gas Giant
- **Colonization**: Players colonize planets within star systems
- **Population**: Citizens live on planets, work buildings, affect production
- **Happiness**: Population morale affects production efficiency

### **Buildings & Production**
- **Building Types**: Farms, Mines, Factories, Power Plants, Spaceports, Research Labs
- **Production Chains**: Buildings + Population + Resource Nodes = Output
- **Levels**: Buildings can be upgraded for increased capacity
- **Employment**: Population works in buildings for resource production

### **Resources & Economy**
- **Resource Types**: Food, Ore, Goods, Fuel, Water, Rare Metals
- **Resource Nodes**: Planets contain extractable resource deposits
- **Global Inventory**: Players have empire-wide resource storage
- **Consumption**: Population consumes food and fuel
- **Trade**: Automated cargo routes between systems

### **Fleets & Ships**
- **Ship Types**: Scout, Fighter, Frigate, Transport, Cruiser, Battleship
- **Fleet Composition**: Multiple ship types per fleet
- **Movement**: Fleets travel between star systems
- **Combat**: Strength-based combat resolution
- **Logistics**: Cargo capacity for resource transport

### **Real-Time Simulation**
- **Game Ticks**: Economy processes every 10 seconds
- **Production**: Buildings generate resources based on employment
- **Growth**: Population grows with adequate food and happiness
- **Fleet Movement**: 2-hour travel time between systems
- **Trade Routes**: Automated resource transport

## Technical Stack

### Backend (Go + PocketBase)
- **Database**: SQLite with 12+ relational collections
- **API**: Custom REST endpoints + PocketBase auto-generated CRUD
- **Authentication**: Discord OAuth2 integration
- **Real-time**: WebSocket connections for live updates
- **Game Engine**: Tick-based simulation system

### Frontend (Vanilla JS)
- **Renderer**: HTML5 Canvas for galaxy map
- **UI**: Tailwind CSS for modern interface
- **State**: Centralized game state management
- **Real-time**: WebSocket integration for live updates

## Database Collections

### Core Game Data
- **systems** - Star system coordinates and discovery status
- **planets** - Planets within systems, colonization data
- **populations** - Citizens living on planets or in fleets
- **buildings** - Infrastructure on planets
- **fleets** - Ship formations and movement
- **ships** - Individual vessels within fleets
- **trade_routes** - Automated cargo transport
- **resource_nodes** - Extractable deposits on planets

### Reference Data
- **planet_types** - Templates for planet characteristics
- **building_types** - Building costs and capabilities
- **ship_types** - Ship specifications and stats
- **resource_types** - Resource properties and consumption

## Development Commands

### Backend Management
```bash
# Main server
go build -o xandaris cmd/main.go
./xandaris serve --dev --dir=pb_data

# Database utilities
go build -o util cmd/util/main.go
./util check    # Check colony distribution
./util reset    # Reset to realistic colony numbers
./util clean    # Clear all game data

# Database seeding
go build -o seed cmd/seed/main.go
./seed
```

### Frontend Development
```bash
npm run dev      # Development server
npm run build    # Production build
npm run preview  # Preview production build
```

## API Endpoints

### Game Data
- `GET /api/map` - Galaxy systems, planets, and lanes
- `GET /api/systems` - Star systems with aggregated data
- `GET /api/status` - Game tick status and server info

### Player Actions
- `POST /api/orders/fleet` - Deploy fleet to target system
- `POST /api/orders/build` - Construct building on planet
- `POST /api/orders/trade` - Create automated trade route

### Real-time
- `WS /api/stream` - WebSocket for live game updates

## Performance Features

- **Smart Economy Processing**: Only processes planets with actual populations
- **Efficient Queries**: Indexed relationships for fast data access
- **Real-time Updates**: WebSocket for instant UI updates
- **Optimized Rendering**: Canvas-based galaxy visualization

## Production Deployment

- **Single Binary**: Complete game server in one executable
- **SQLite Database**: Self-contained data storage
- **Static Assets**: Frontend integrated into backend
- **Docker Ready**: Single container deployment
- **Minimal Resources**: Runs efficiently on t3.micro

## License

MIT