# Xandaris - 4X Space Strategy Game

Xandaris is a numbers-only 4X-lite strategy game you can run right in your browser. Build colonies, manage resources and send fleets in quick, real-time turns.

## Quick Start

1. Start the backend:
   ```bash
   cd backend
   go run cmd/main.go
   ```
2. Start the frontend:
   ```bash
   cd frontend
   npm install
   npm run dev
   ```
3. Open <http://localhost:5173> in your browser.

For full setup instructions and technical details, see [DEVELOPING.md](DEVELOPING.md).

## Gameplay Overview

- **Systems** – Colonize and develop star systems
- **Resources** – Food, ore, goods, fuel and credits
- **Buildings** – Habitats, farms, mines, factories and shipyards
- **Fleets** – Move your forces between systems
- **Trade** – Automate cargo routes between your worlds
- **Diplomacy** – Form alliances and treaties with other players

### Controls

- **Mouse** – Pan the map, select systems, right‑click for context
- **Wheel** – Zoom in and out
- **Keyboard Shortcuts**
  - `F` – Send fleet from selected system
  - `T` – Create trade route
  - `B` – Build in selected system
  - `C` – Center on selected system
  - `H` – Fit all systems in view
  - `ESC` – Close menus

## License

MIT
