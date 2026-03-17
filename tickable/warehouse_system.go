package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&WarehouseSystem{
		BaseSystem: NewBaseSystem("Warehouse", 97),
	})
}

// WarehouseSystem tracks storage utilization across factions and
// auto-expands storage on planets with high tech and Trading Posts.
//
// Base storage is 1000 per resource (from DEFAULT_RESOURCE_CAPACITY).
// This system adds bonus capacity:
//   - Trading Post L1: +200 capacity
//   - Trading Post L2: +500
//   - Trading Post L3: +1000
//   - Trading Post L4: +2000
//   - Trading Post L5: +5000 (unlimited effective)
//   - Tech level bonus: +200 per tech level
//
// Also warns when planets are near capacity so factions can sell
// or expand before overflow.
type WarehouseSystem struct {
	*BaseSystem
	lastWarning map[int]int64
}

func (ws *WarehouseSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := ws.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ws.lastWarning == nil {
		ws.lastWarning = make(map[int]int64)
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			// Calculate bonus capacity from TP and tech
			bonus := int(planet.TechLevel * 200)
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
					switch b.Level {
					case 1:
						bonus += 200
					case 2:
						bonus += 500
					case 3:
						bonus += 1000
					case 4:
						bonus += 2000
					case 5:
						bonus += 5000
					}
				}
			}

			// Apply bonus capacity to all stored resources
			baseCap := planet.GetStorageCapacity()
			totalCap := baseCap + bonus
			for _, storage := range planet.StoredResources {
				if storage != nil && storage.Capacity < totalCap {
					storage.Capacity = totalCap
				}
			}

			// Warn when near capacity
			for res, storage := range planet.StoredResources {
				if storage == nil || storage.Capacity <= 0 {
					continue
				}
				ratio := float64(storage.Amount) / float64(storage.Capacity)
				if ratio > 0.90 && tick-ws.lastWarning[planet.GetID()] > 5000 {
					ws.lastWarning[planet.GetID()] = tick
					game.LogEvent("alert", planet.Owner,
						fmt.Sprintf("📦 %s storage nearly full: %s at %d/%d (%.0f%%). Sell surplus or upgrade Trading Post!",
							planet.Name, res, storage.Amount, storage.Capacity, ratio*100))
					break
				}
			}
		}
	}

	_ = rand.Intn // suppress unused import if needed
}
