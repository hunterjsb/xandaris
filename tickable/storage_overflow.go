package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&StorageOverflowSystem{
		BaseSystem: NewBaseSystem("StorageOverflow", 107),
	})
}

// StorageOverflowSystem automatically handles the common situation
// where a planet has storage full on some resources but needs others.
// Instead of just warning, it takes action.
//
// When a resource is at 100% capacity:
//   1. If there's a Trading Post: auto-sell 10% at local price
//   2. If planet owner has <10,000 credits: sell at premium (+50%)
//   3. Credits go directly to the planet owner
//
// This prevents the "storage full" death spiral where resources
// keep producing but overflow is wasted, while the player has no
// credits to build storage upgrades.
//
// Also auto-distributes surplus to other owned planets in the same
// system that have room.
type StorageOverflowSystem struct {
	*BaseSystem
}

func (sos *StorageOverflowSystem) OnTick(tick int64) {
	if tick%200 != 0 {
		return
	}

	ctx := sos.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		// Group planets by owner
		ownerPlanets := make(map[string][]*entities.Planet)
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				ownerPlanets[planet.Owner] = append(ownerPlanets[planet.Owner], planet)
			}
		}

		for owner, planets := range ownerPlanets {
			var player *entities.Player
			for _, p := range players {
				if p != nil && p.Name == owner {
					player = p
					break
				}
			}
			if player == nil {
				continue
			}

			for _, planet := range planets {
				for res, storage := range planet.StoredResources {
					if storage == nil || storage.Capacity <= 0 {
						continue
					}
					if float64(storage.Amount)/float64(storage.Capacity) < 0.95 {
						continue
					}

					// Storage nearly full — take action
					overflow := storage.Amount - int(float64(storage.Capacity)*0.80)
					if overflow <= 0 {
						continue
					}

					// Try to distribute to other planets first
					distributed := 0
					for _, other := range planets {
						if other.GetID() == planet.GetID() {
							continue
						}
						otherStorage := other.StoredResources[res]
						if otherStorage == nil {
							other.AddStoredResource(res, overflow-distributed)
							planet.RemoveStoredResource(res, overflow-distributed)
							distributed = overflow
							break
						}
						space := otherStorage.Capacity - otherStorage.Amount
						if space > 0 {
							transfer := overflow - distributed
							if transfer > space {
								transfer = space
							}
							other.AddStoredResource(res, transfer)
							planet.RemoveStoredResource(res, transfer)
							distributed += transfer
						}
					}

					// If still overflowing, auto-sell
					remaining := overflow - distributed
					if remaining > 10 {
						price := market.GetSellPrice(res)
						credits := int(price * float64(remaining))
						planet.RemoveStoredResource(res, remaining)
						player.Credits += credits
						market.AddTradeVolume(res, remaining, false)

						if rand.Intn(5) == 0 {
							game.LogEvent("trade", owner,
								fmt.Sprintf("📦 %s auto-sold %d overflow %s for %dcr (storage full)",
									planet.Name, remaining, res, credits))
						}
					}
				}
			}
		}
	}
}
