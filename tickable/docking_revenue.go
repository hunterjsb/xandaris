package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&DockingRevenueSystem{
		BaseSystem: NewBaseSystem("DockingRevenue", 28),
	})
}

// DockingRevenueSystem generates passive income from foreign ships
// using your Trading Posts. This incentivizes building high-level
// Trading Posts in strategic locations — you become the port authority.
//
// Revenue per foreign ship docked per interval:
//   TP L1: 10 cr   (basic docking fee)
//   TP L2: 25 cr   (with services)
//   TP L3: 50 cr   (full services + insurance)
//   TP L4: 100 cr  (premium port)
//   TP L5: 200 cr  (galactic trade hub)
//
// Friendly factions (Friendly or Allied relations) pay 50% fee.
// Hostile factions pay 200% fee (you're price-gouging enemies).
//
// This creates trade infrastructure as a profit center, not just a gate.
type DockingRevenueSystem struct {
	*BaseSystem
}

func (drs *DockingRevenueSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := drs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	dm := game.GetDiplomacyManager()
	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// For each planet with a Trading Post, count foreign ships docked
	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			// Find Trading Post level
			tpLevel := 0
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
					tpLevel = b.Level
					break
				}
			}
			if tpLevel == 0 {
				continue
			}

			// Count foreign ships in this system (not just docked — orbiting counts)
			totalRevenue := 0
			foreignCount := 0
			for _, p := range players {
				if p == nil || p.Name == planet.Owner {
					continue
				}
				for _, ship := range p.OwnedShips {
					if ship == nil || ship.CurrentSystem != sys.ID || ship.Status == entities.ShipStatusMoving {
						continue
					}

					baseRevenue := dockingFee(tpLevel)

					// Adjust by diplomacy
					if dm != nil {
						relation := dm.GetRelation(planet.Owner, p.Name)
						switch {
						case relation >= 2: // Allied
							baseRevenue = baseRevenue / 2
						case relation >= 1: // Friendly
							baseRevenue = baseRevenue * 3 / 4
						case relation <= -2: // Hostile
							baseRevenue = baseRevenue * 2
						}
					}

					totalRevenue += baseRevenue
					foreignCount++
				}
			}

			if totalRevenue > 0 {
				// Credit the planet owner
				for _, p := range players {
					if p != nil && p.Name == planet.Owner {
						p.Credits += totalRevenue
						break
					}
				}

				// Only log if significant
				if totalRevenue > 50 {
					game.LogEvent("logistics", planet.Owner,
						fmt.Sprintf("🏗️ %s earned %d cr in docking fees (%d foreign ships at TP L%d)",
							planet.Name, totalRevenue, foreignCount, tpLevel))
				}
			}
		}
	}
}

func dockingFee(tpLevel int) int {
	switch tpLevel {
	case 1:
		return 10
	case 2:
		return 25
	case 3:
		return 50
	case 4:
		return 100
	case 5:
		return 200
	default:
		return 5
	}
}
