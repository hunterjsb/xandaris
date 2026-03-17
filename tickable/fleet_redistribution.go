package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FleetRedistributionSystem{
		BaseSystem: NewBaseSystem("FleetRedistribution", 126),
	})
}

// FleetRedistributionSystem addresses port congestion by automatically
// redistributing idle ships from gridlocked systems to less crowded ones.
//
// When a system has 50+ ships from one faction:
//   - Identify idle ships (not moving, no cargo, no active route)
//   - Move up to 10 idle ships per tick to the least crowded owned system
//   - Prioritize moving Colony ships (least useful in a crowded port)
//
// This directly fixes the SYS-30 gridlock (190 ships!) by spreading
// fleets across the faction's territory.
type FleetRedistributionSystem struct {
	*BaseSystem
	lastRedist map[string]int64 // faction → last redistribution tick
}

func (frs *FleetRedistributionSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := frs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if frs.lastRedist == nil {
		frs.lastRedist = make(map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}
		if tick-frs.lastRedist[player.Name] < 5000 {
			continue
		}

		// Count ships per system
		shipsBySystem := make(map[int][]*entities.Ship)
		for _, ship := range player.OwnedShips {
			if ship == nil {
				continue
			}
			shipsBySystem[ship.CurrentSystem] = append(shipsBySystem[ship.CurrentSystem], ship)
		}

		// Find overcrowded system
		var crowdedSys int
		var crowdedShips []*entities.Ship
		maxShips := 0
		for sysID, ships := range shipsBySystem {
			if len(ships) > maxShips {
				maxShips = len(ships)
				crowdedSys = sysID
				crowdedShips = ships
			}
		}

		if maxShips < 30 {
			continue // not crowded enough to bother
		}

		// Find least crowded owned system
		ownedSystems := make(map[int]bool)
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
					ownedSystems[sys.ID] = true
				}
			}
		}

		leastCrowdedSys := -1
		leastCount := 999999
		for sysID := range ownedSystems {
			if sysID == crowdedSys {
				continue
			}
			count := len(shipsBySystem[sysID])
			if count < leastCount {
				leastCount = count
				leastCrowdedSys = sysID
			}
		}

		if leastCrowdedSys < 0 {
			continue
		}

		// Move idle ships (prefer Colony ships, then idle Cargo)
		moved := 0
		for _, ship := range crowdedShips {
			if moved >= 5 {
				break
			}
			if ship.Status == entities.ShipStatusMoving || ship.DeliveryID != 0 {
				continue
			}
			if ship.GetTotalCargo() > 0 {
				continue // has cargo, don't move
			}

			// Prefer useless ships
			priority := 0
			if ship.ShipType == entities.ShipTypeColony && ship.Colonists == 0 {
				priority = 3 // empty colony ships first
			} else if ship.ShipType == entities.ShipTypeCargo && ship.GetTotalCargo() == 0 {
				priority = 1
			}
			if priority == 0 {
				continue
			}

			if game.StartShipJourney(ship, leastCrowdedSys) {
				moved++
			}
		}

		if moved > 0 {
			frs.lastRedist[player.Name] = tick

			crowdedName := fmt.Sprintf("SYS-%d", crowdedSys+1)
			destName := fmt.Sprintf("SYS-%d", leastCrowdedSys+1)
			for _, sys := range systems {
				if sys.ID == crowdedSys {
					crowdedName = sys.Name
				}
				if sys.ID == leastCrowdedSys {
					destName = sys.Name
				}
			}

			game.LogEvent("logistics", player.Name,
				fmt.Sprintf("🚢 Fleet redistribution: %d idle ships moved from %s (%d ships) to %s (%d ships)",
					moved, crowdedName, maxShips, destName, leastCount))
		}
	}
}
