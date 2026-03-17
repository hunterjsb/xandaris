package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&EmergencySupplySystem{
		BaseSystem: NewBaseSystem("EmergencySupply", 12),
	})
}

// EmergencySupplySystem prevents complete resource starvation on
// planets by generating small emergency supplies when critical
// resources hit zero. This prevents unrecoverable death spirals.
//
// When a planet has:
//   - 0 Water + population > 100: emergency water ration (+5 Water)
//   - 0 Iron + active mines: emergency ore extraction (+5 Iron)
//   - 0 Fuel + active generators: emergency fuel cell (+5 Fuel)
//
// Emergency supplies are small (5 units) and fire every 500 ticks.
// They're survival rations, not production — just enough to prevent
// total collapse while the player/agent fixes the supply chain.
//
// This runs at priority 12 (after fuel reserve at 11) to provide
// a second safety net.
type EmergencySupplySystem struct {
	*BaseSystem
	lastEmergency map[int]int64 // planetID → last emergency tick
}

func (ess *EmergencySupplySystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := ess.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ess.lastEmergency == nil {
		ess.lastEmergency = make(map[int]int64)
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Population < 100 {
				continue
			}

			pid := planet.GetID()
			if tick-ess.lastEmergency[pid] < 500 {
				continue
			}

			supplied := false

			// Critical Water
			if planet.GetStoredAmount(entities.ResWater) == 0 {
				planet.AddStoredResource(entities.ResWater, 5)
				supplied = true
			}

			// Critical Fuel (if has generator)
			if planet.GetStoredAmount(entities.ResFuel) == 0 {
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingGenerator {
						planet.AddStoredResource(entities.ResFuel, 5)
						supplied = true
						break
					}
				}
			}

			// Critical Iron (if has mine)
			if planet.GetStoredAmount(entities.ResIron) == 0 {
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingMine {
						planet.AddStoredResource(entities.ResIron, 5)
						supplied = true
						break
					}
				}
			}

			if supplied {
				ess.lastEmergency[pid] = tick
				// Only log occasionally to avoid spam
				if rand.Intn(5) == 0 {
					game.LogEvent("alert", planet.Owner,
						fmt.Sprintf("🆘 Emergency supplies on %s — critical resources at zero. Import supplies ASAP!",
							planet.Name))
				}
			}
		}
	}
}
