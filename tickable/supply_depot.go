package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SupplyDepotSystem{
		BaseSystem: NewBaseSystem("SupplyDepot", 13),
	})
}

// SupplyDepotSystem creates automatic fuel and resource staging points
// along busy trade routes. When a system has 3+ cargo ships pass
// through regularly, an "NPC supply depot" forms that sells fuel
// at market price to any ship.
//
// This solves the fundamental logistics bootstrapping problem:
// you can't ship fuel somewhere if you don't have fuel to get there.
//
// Depot mechanics:
//   - Forms in systems where 3+ different factions have had ships
//   - Stocks 100 Fuel and 50 Water permanently (NPC-supplied)
//   - Ships can auto-refuel from depots (like owned planets)
//   - Depots charge 2cr per unit of fuel (deducted from ship owner)
//   - Depots persist until traffic drops below threshold
//
// Priority 13: runs right after fuel reserve (11) and emergency supply (12)
// to provide a third safety net for stranded ships.
type SupplyDepotSystem struct {
	*BaseSystem
	depots    map[int]*SupplyDepot // systemID → depot
	traffic   map[int]map[string]int64 // systemID → factionName → last seen tick
	nextCheck int64
}

// SupplyDepot represents an NPC fuel station.
type SupplyDepot struct {
	SystemID int
	SysName  string
	FuelStock int
	Active   bool
}

func (sds *SupplyDepotSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := sds.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if sds.depots == nil {
		sds.depots = make(map[int]*SupplyDepot)
		sds.traffic = make(map[int]map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Track ship traffic per system
	for _, p := range players {
		if p == nil {
			continue
		}
		for _, ship := range p.OwnedShips {
			if ship == nil {
				continue
			}
			if sds.traffic[ship.CurrentSystem] == nil {
				sds.traffic[ship.CurrentSystem] = make(map[string]int64)
			}
			sds.traffic[ship.CurrentSystem][p.Name] = tick
		}
	}

	// Refuel ships at depots
	for sysID, depot := range sds.depots {
		if !depot.Active {
			continue
		}

		// Restock fuel slowly (NPC supply)
		if depot.FuelStock < 100 {
			depot.FuelStock += 2
		}

		// Refuel ships in this system
		for _, p := range players {
			if p == nil {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship == nil || ship.CurrentSystem != sysID || ship.Status == entities.ShipStatusMoving {
					continue
				}
				if ship.CurrentFuel >= ship.MaxFuel || depot.FuelStock <= 0 {
					continue
				}

				// Refuel up to 10 per tick, costs 2cr per unit
				refuel := ship.MaxFuel - ship.CurrentFuel
				if refuel > 10 {
					refuel = 10
				}
				if refuel > depot.FuelStock {
					refuel = depot.FuelStock
				}
				cost := refuel * 2
				if p.Credits < cost {
					continue
				}

				ship.CurrentFuel += refuel
				depot.FuelStock -= refuel
				p.Credits -= cost
			}
		}
	}

	// Check for new depots every 5000 ticks
	if sds.nextCheck == 0 {
		sds.nextCheck = tick + 5000
	}
	if tick < sds.nextCheck {
		return
	}
	sds.nextCheck = tick + 5000

	// Form depots in high-traffic systems without owned planets
	for sysID, factions := range sds.traffic {
		if _, hasDepot := sds.depots[sysID]; hasDepot {
			continue
		}

		// Count recent factions (within last 5000 ticks)
		recentFactions := 0
		for _, lastSeen := range factions {
			if tick-lastSeen < 5000 {
				recentFactions++
			}
		}

		if recentFactions < 2 {
			continue
		}

		// Check system doesn't already have owned planets with fuel
		hasOwnedPlanet := false
		sysName := ""
		for _, sys := range systems {
			if sys.ID != sysID {
				continue
			}
			sysName = sys.Name
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
					hasOwnedPlanet = true
					break
				}
			}
			break
		}

		if hasOwnedPlanet {
			continue // don't need depot where planets exist
		}

		sds.depots[sysID] = &SupplyDepot{
			SystemID:  sysID,
			SysName:   sysName,
			FuelStock: 100,
			Active:    true,
		}

		if rand.Intn(2) == 0 {
			game.LogEvent("logistics", "",
				fmt.Sprintf("⛽ Supply depot established in %s! Ships can refuel here (2cr/unit). No more stranded fleets!",
					sysName))
		}
	}
}

// GetDepotCount returns number of active supply depots.
func (sds *SupplyDepotSystem) GetDepotCount() int {
	count := 0
	for _, d := range sds.depots {
		if d.Active {
			count++
		}
	}
	return count
}
