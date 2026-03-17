package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SupplyCrisisSystem{
		BaseSystem: NewBaseSystem("SupplyCrisis", 41),
	})
}

// SupplyCrisisSystem generates system-wide resource shortages that
// create economic pressure and trading opportunities.
//
// A supply crisis targets a specific resource in a specific system:
//   - All planets in the system consume 2x that resource
//   - Import price for that resource doubles in the system
//   - Factions who bring in supply earn bonus credits
//   - Crisis lasts 3000-6000 ticks (~5-10 min)
//
// This creates dynamic trade demand — a Water crisis in a distant system
// means someone with Water surplus can make huge profits shipping it there.
//
// Crises can cascade: if your only Fuel source is in crisis, your ships
// can't refuel, stranding your fleet until the crisis resolves.
type SupplyCrisisSystem struct {
	*BaseSystem
	crises    []*SupplyCrisis
	nextCrisis int64
}

// SupplyCrisis represents an active resource shortage in a system.
type SupplyCrisis struct {
	SystemID   int
	SystemName string
	Resource   string
	Severity   float64 // 1.5-3.0x consumption multiplier
	TicksLeft  int
	Active     bool
	Resolved   bool // true if players supplied enough to end early
}

func (scs *SupplyCrisisSystem) OnTick(tick int64) {
	if tick%200 != 0 {
		return
	}

	ctx := scs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	systems := game.GetSystems()

	if scs.nextCrisis == 0 {
		scs.nextCrisis = tick + 3000 + int64(rand.Intn(5000))
	}

	// Decay existing crises
	for _, crisis := range scs.crises {
		if !crisis.Active {
			continue
		}
		crisis.TicksLeft -= 200

		// Check if crisis resolved naturally
		if crisis.TicksLeft <= 0 {
			crisis.Active = false
			game.LogEvent("event", "",
				fmt.Sprintf("✅ %s shortage in %s has ended! Supply chains recovering",
					crisis.Resource, crisis.SystemName))
			continue
		}

		// Crisis drains the resource from affected planets
		scs.applyCrisisDrain(crisis, systems, game)
	}

	// Spawn new crisis
	if tick >= scs.nextCrisis {
		scs.nextCrisis = tick + 8000 + int64(rand.Intn(10000))
		scs.spawnCrisis(game, systems)
	}
}

func (scs *SupplyCrisisSystem) applyCrisisDrain(crisis *SupplyCrisis, systems []*entities.System, game GameProvider) {
	for _, sys := range systems {
		if sys.ID != crisis.SystemID {
			continue
		}
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}
			// Extra drain during crisis
			stored := planet.GetStoredAmount(crisis.Resource)
			drain := int(float64(stored) * 0.02 * crisis.Severity) // 2-6% per tick
			if drain > 0 && stored > drain {
				planet.RemoveStoredResource(crisis.Resource, drain)
			}
		}
		break
	}
}

func (scs *SupplyCrisisSystem) spawnCrisis(game GameProvider, systems []*entities.System) {
	if len(systems) == 0 {
		return
	}

	// Don't have too many active crises
	activeCount := 0
	for _, c := range scs.crises {
		if c.Active {
			activeCount++
		}
	}
	if activeCount >= 2 {
		return
	}

	// Pick a system with owned planets
	var candidates []*entities.System
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				candidates = append(candidates, sys)
				break
			}
		}
	}
	if len(candidates) == 0 {
		return
	}

	sys := candidates[rand.Intn(len(candidates))]
	resources := []string{entities.ResWater, entities.ResFuel, entities.ResIron, entities.ResOil}
	res := resources[rand.Intn(len(resources))]

	crisis := &SupplyCrisis{
		SystemID:   sys.ID,
		SystemName: sys.Name,
		Resource:   res,
		Severity:   1.5 + rand.Float64()*1.5,
		TicksLeft:  3000 + rand.Intn(3000),
		Active:     true,
	}
	scs.crises = append(scs.crises, crisis)

	game.LogEvent("event", "",
		fmt.Sprintf("⚠️ SUPPLY CRISIS: %s shortage in %s! Consumption %.0f%% above normal. Ship supplies to help!",
			res, sys.Name, (crisis.Severity-1)*100))
}

// GetActiveCrises returns currently active supply crises (for API/dashboard).
func (scs *SupplyCrisisSystem) GetActiveCrises() []*SupplyCrisis {
	var result []*SupplyCrisis
	for _, c := range scs.crises {
		if c.Active {
			result = append(result, c)
		}
	}
	return result
}
