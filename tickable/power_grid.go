package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PowerGridSystem{
		BaseSystem: NewBaseSystem("PowerGrid", 105),
	})
}

// PowerGridSystem creates inter-planet power sharing within a system.
// When one planet has excess power and another has a deficit, power
// can flow between them via "power conduits" (automatic if both have TPs).
//
// This solves the common scenario where one planet has a Fusion Reactor
// generating excess MW while a neighboring planet is in power crisis.
//
// Requirements:
//   - Both planets in the same system
//   - Both owned by the same faction
//   - Both have Trading Posts (the conduit infrastructure)
//
// Transfer: up to 25% of excess power per tick
// Cost: 1 Fuel per 50 MW transferred (transmission losses)
type PowerGridSystem struct {
	*BaseSystem
}

func (pgs *PowerGridSystem) OnTick(tick int64) {
	if tick%50 != 0 {
		return
	}

	ctx := pgs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		// Group planets by owner
		ownerPlanets := make(map[string][]*entities.Planet)
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}
			// Must have TP
			hasTP := false
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
					hasTP = true
					break
				}
			}
			if hasTP {
				ownerPlanets[planet.Owner] = append(ownerPlanets[planet.Owner], planet)
			}
		}

		for _, planets := range ownerPlanets {
			if len(planets) < 2 {
				continue
			}

			// Find surplus and deficit planets
			var surplus, deficit []*entities.Planet
			for _, p := range planets {
				ratio := p.GetPowerRatio()
				if ratio > 1.2 { // 20%+ excess
					surplus = append(surplus, p)
				} else if ratio < 0.5 { // under 50% power
					deficit = append(deficit, p)
				}
			}

			// Transfer power from surplus to deficit
			for _, src := range surplus {
				for _, dst := range deficit {
					excess := src.PowerGenerated - src.PowerConsumed
					if excess <= 0 {
						continue
					}

					transfer := excess * 0.25 // 25% of excess
					if transfer < 5 {
						continue
					}

					// Cost: 1 Fuel per 50 MW
					fuelCost := int(transfer / 50)
					if fuelCost < 1 {
						fuelCost = 1
					}
					if src.GetStoredAmount(entities.ResFuel) < fuelCost {
						continue
					}

					src.RemoveStoredResource(entities.ResFuel, fuelCost)
					dst.PowerGenerated += transfer

					if rand.Intn(20) == 0 {
						game.LogEvent("logistics", src.Owner,
							fmt.Sprintf("⚡ Power grid: %.0f MW transferred from %s to %s (cost: %d Fuel)",
								transfer, src.Name, dst.Name, fuelCost))
					}
				}
			}
		}
	}
}
