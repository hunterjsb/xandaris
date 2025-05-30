# Xandaris - 4X Space Strategy Game

A numbers-only 4X-lite space strategy game built with vanilla JavaScript and Canvas, featuring crypto server banking and real-time multiplayer.

## Features

- **Real-time 4X gameplay** with 10-second ticks
- **Crypto server banking** with passive income
- **Player-driven economy** with trade routes
- **Colony management** and fleet operations
- **Canvas-based map** with smooth zoom and pan
- **Real-time updates** via WebSocket
- **Discord OAuth** authentication

## Tech Stack

- **Frontend**: Vanilla JS + Canvas + Tailwind CSS
- **Backend**: PocketBase (Go + SQLite)
- **Auth**: Discord OAuth2
- **Real-time**: WebSocket
- **Build**: Vite

## Getting Started

### Prerequisites

- Node.js 18+
- PocketBase server running on port 8090

### Installation

1. Clone and install dependencies:
```bash
cd xandaris
npm install
```

2. Set up environment:
```bash
cp .env.example .env
# Edit .env with your PocketBase URL
```

3. Start development server:
```bash
npm run dev
```

The app will be available at `http://localhost:5173`

### Building for Production

```bash
npm run build
```

Output will be in the `web/` directory, ready to serve statically.

## Game Mechanics

### Core Gameplay
- **Systems**: Colonize and develop star systems
- **Resources**: Food, ore, goods, fuel, credits
- **Buildings**: Habitats, farms, mines, factories, shipyards
- **Fleets**: Send military forces between systems
- **Trade**: Create automated cargo routes
- **Diplomacy**: Form alliances and treaties

### Controls
- **Mouse**: Pan map, select systems, right-click for context menu
- **Wheel**: Zoom in/out
- **Keyboard Shortcuts**:
  - `F`: Send fleet from selected system
  - `T`: Create trade route from selected system
  - `B`: Build in selected system
  - `C`: Center on selected system
  - `H`: Fit all systems in view
  - `ESC`: Close modals/menus

## Project Structure

```
src/
├── lib/
│   └── pocketbase.js      # PocketBase client & auth
├── stores/
│   └── gameState.js       # Game state management
├── components/
│   ├── mapRenderer.js     # Canvas map renderer
│   └── uiController.js    # UI interactions
├── styles.css             # Tailwind + custom styles
└── main.js               # App entry point
```

## API Integration

The frontend expects these PocketBase collections:
- `users` - Player accounts
- `systems` - Star systems with resources/buildings
- `fleets` - Moving military units
- `trade_routes` - Automated cargo transport
- `treaties` - Diplomatic agreements

And these API endpoints:
- `GET /map` - Initial map data
- `POST /orders/fleet` - Send fleet
- `POST /orders/build` - Queue building
- `POST /orders/trade` - Create trade route
- `POST /diplomacy` - Propose treaty
- `WS /stream` - Real-time updates

## Development

This project follows the patterns established in the vibe-chuck reference project:
- Component-based architecture
- Store-based state management
- Real-time data synchronization
- Responsive design with Tailwind
- Modal-based interactions

## License

MIT