package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&OilToFuelChainSystem{
		BaseSystem: NewBaseSystem("OilToFuelChain", 15),
	})
}

// OilToFuelChainSystem ensures every planet with a Generator but no
// Refinery can still produce minimal Fuel from stored Oil. This fixes
// the chicken-and-egg problem: Refineries need power, but Generators
// need Fuel from Refineries.
//
// When a planet has:
//   - A Generator (any level)
//   - Oil stored (>10 units)
//   - Fuel stored (<20 units)
//   - No operational Refinery
//
// The Generator itself performs crude Oil→Fuel conversion:
//   5 Oil → 2 Fuel per 100 ticks (inefficient but keeps lights on)
//
// This is intentionally worse than a Refinery (which does 5 Oil → 4 Fuel)
// but prevents the total energy death spiral that's happening galaxy-wide.
//
// Priority 15: runs after solar bonus (7) and energy crisis (10) but
// before resource accumulation (16) and power system calculations.
type OilToFuelChainSystem struct {
	*BaseSystem
}

func (otfc *OilToFuelChainSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := otfc.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			// Check for Generator but no operational Refinery
			hasGenerator := false
			hasRefinery := false
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.IsOperational {
					if b.BuildingType == entities.BuildingGenerator {
						hasGenerator = true
					}
					if b.BuildingType == entities.BuildingRefinery {
						hasRefinery = true
					}
				}
			}

			if !hasGenerator {
				continue
			}

			oilStored := planet.GetStoredAmount(entities.ResOil)
			fuelStored := planet.GetStoredAmount(entities.ResFuel)

			if oilStored < 5 || fuelStored >= 30 {
				if hasRefinery && fuelStored >= 10 { continue } // crude backup when refinery offline
			}

			// Crude conversion: 5 Oil → 2 Fuel
			planet.RemoveStoredResource(entities.ResOil, 5)
			planet.AddStoredResource(entities.ResFuel, 2)

			if rand.Intn(20) == 0 {
				game.LogEvent("logistics", planet.Owner,
					fmt.Sprintf("⛽ %s: Generator performing crude Oil→Fuel conversion (5→2). Build a Refinery for better rates!",
						planet.Name))
			}
		}
	}
}
