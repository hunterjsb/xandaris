package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&BuildingRepairSystem{
		BaseSystem: NewBaseSystem("BuildingRepair", 50),
	})
}

// BuildingRepairSystem auto-repairs damaged buildings over time.
// Bases are always immediately restored. Other buildings repair
// after 2000 ticks if the planet has population and Iron.
type BuildingRepairSystem struct {
	*BaseSystem
	damagedAt map[int]int64 // buildingID → tick when damaged
}

func (brs *BuildingRepairSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := brs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if brs.damagedAt == nil {
		brs.damagedAt = make(map[int]int64)
	}

	for _, sys := range game.GetSystems() {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			for _, be := range planet.Buildings {
				b, ok := be.(*entities.Building)
				if !ok {
					continue
				}

				// Base must always be operational
				if b.BuildingType == entities.BuildingBase && !b.IsOperational {
					b.IsOperational = true
					delete(brs.damagedAt, b.GetID())
					continue
				}

				if b.IsOperational {
					delete(brs.damagedAt, b.GetID())
					continue
				}

				// Track when building went offline
				if _, tracked := brs.damagedAt[b.GetID()]; !tracked {
					brs.damagedAt[b.GetID()] = tick
				}

				// Repair after 2000 ticks if planet has population and some Iron
				elapsed := tick - brs.damagedAt[b.GetID()]
				if elapsed >= 2000 && planet.Population > 0 {
					ironCost := 5 * b.Level
					if ironCost < 5 {
						ironCost = 5
					}
					if planet.GetStoredAmount(entities.ResIron) >= ironCost {
						planet.RemoveStoredResource(entities.ResIron, ironCost)
						b.IsOperational = true
						delete(brs.damagedAt, b.GetID())
						game.LogEvent("event", planet.Owner,
							fmt.Sprintf("🔧 %s on %s repaired (-%d Iron)", b.Name, planet.Name, ironCost))
					}
				}
			}
		}
	}
}
