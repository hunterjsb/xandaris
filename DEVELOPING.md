# Developing Xandaris

This document covers the technical details for running and building the game.

## Tech Stack

- **Frontend**: Vanilla JS + Canvas + Tailwind CSS
- **Backend**: PocketBase (Go + SQLite)
- **Auth**: Discord OAuth2
- **Real-time**: WebSocket
- **Build**: Vite

## Getting Started

### Prerequisites

- Node.js 18+
- Go 1.20+ (for backend)

### Backend Setup

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```
2. Run the backend:
   ```bash
   cd backend
   go mod tidy
   go run cmd/main.go
   ```

### Frontend Setup

1. Install dependencies and start the dev server:
   ```bash
   cd frontend
   npm install
   npm run dev
   ```

The frontend is served at `http://localhost:5173`.

### Building for Production

```bash
cd frontend
npm run build
```

The static build outputs to `backend/pb_public/`.

## Project Structure

```text
.
├── frontend/
│   ├── src/
│   │   ├── lib/
│   │   │   └── pocketbase.js
│   │   ├── stores/
│   │   │   └── gameState.js
│   │   ├── components/
│   │   │   ├── mapRenderer.js
│   │   │   └── uiController.js
│   │   ├── styles.css
│   │   └── main.js
│   ├── static/
│   ├── index.html
│   ├── package.json
│   └── vite.config.js
├── backend/
│   ├── cmd/
│   │   ├── main.go
│   │   └── seed/
│   │       └── main.go
│   ├── internal/
│   ├── migrations/
│   ├── pkg/
│   ├── go.mod
│   ├── go.sum
│   └── pb_public/
├── .env.example
├── .gitignore
└── README.md
```

## API Integration

Collections expected by the frontend:
- `users` - Player accounts
- `systems` - Star systems with resources/buildings
- `fleets` - Moving military units
- `trade_routes` - Automated cargo transport
- `treaties` - Diplomatic agreements

API endpoints:
- `GET /map` - Initial map data
- `POST /orders/fleet` - Send fleet
- `POST /orders/build` - Queue building
- `POST /orders/trade` - Create trade route
- `POST /diplomacy` - Propose treaty
- `WS /stream` - Real-time updates

## Development Notes

This project follows the patterns established in the vibe-chuck reference project:
- Component-based architecture
- Store-based state management
- Real-time data synchronization
- Responsive design with Tailwind
- Modal-based interactions

## License

MIT
