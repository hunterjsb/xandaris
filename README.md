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


### Gameplay
1. **Colonize Planets** - Establish colonies on habitable worlds
2. **Build Infrastructure** - Construct farms, mines, factories, spaceports
3. **Manage Population** - Keep citizens happy and employed
4. **Extract Resources** - Work resource nodes for materials
5. **Build Fleets** - Construct ships for expansion and defense
6. **Trade & Diplomacy** - Establish trade routes and treaties

## License

MIT
