package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ResourceDepletionSystem{
		BaseSystem: NewBaseSystem("ResourceDepletion", 14),
	})
}

// ResourceDepletionSystem gradually depletes resource nodes that are
// heavily mined, creating long-term economic pressure to expand.
//
// Each resource node has an Abundance value (0-100+). Mining (resource
// accumulation) reduces abundance over time. When abundance hits 0,
// the resource is exhausted and produces nothing.
//
// Depletion rate:
//   - Each mine extraction reduces abundance by 0.01 per tick
//   - Base depletion: ~1 abundance per 100 ticks (scaled by mine level)
//   - Higher mine levels deplete faster (extracting more aggressively)
//
// Recovery:
//   - Unmined resources slowly recover (+0.1 abundance per 1000 ticks)
//   - Exploration can discover new deposits
//   - Terraforming improves base habitability, which slows depletion
//
// This forces factions to:
//   1. Expand to new resource sources (colonization pressure)
//   2. Trade for resources they can no longer produce locally
//   3. Invest in exploration to find new deposits
//   4. Balance extraction rate vs sustainability
type ResourceDepletionSystem struct {
	*BaseSystem
	depletionLog map[int]int64 // resourceID → last depletion warning tick
}

func (rds *ResourceDepletionSystem) OnTick(tick int64) {
	if tick%200 != 0 {
		return
	}

	ctx := rds.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if rds.depletionLog == nil {
		rds.depletionLog = make(map[int]int64)
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			rds.processDepletion(tick, planet, game)
		}
	}
}

func (rds *ResourceDepletionSystem) processDepletion(tick int64, planet *entities.Planet, game GameProvider) {
	for _, re := range planet.Resources {
		res, ok := re.(*entities.Resource)
		if !ok {
			continue
		}

		// Check if there's an active mine on this resource
		hasMine := false
		mineLevel := 0
		for _, be := range planet.Buildings {
			b, ok := be.(*entities.Building)
			if !ok {
				continue
			}
			if b.BuildingType == entities.BuildingMine && b.IsOperational && b.ResourceNodeID == res.GetID() {
				hasMine = true
				mineLevel = b.Level
				break
			}
		}

		if hasMine && res.Abundance > 0 {
			// Depletion: higher mine level = faster depletion
			depletionRate := 0.02 * float64(mineLevel)
			res.Abundance -= int(depletionRate)
			if res.Abundance < 0 {
				res.Abundance = 0
			}

			// Warn when resource is getting low
			if res.Abundance <= 15 && res.Abundance > 0 {
				lastWarn := rds.depletionLog[res.GetID()]
				if tick-lastWarn > 10000 {
					rds.depletionLog[res.GetID()] = tick
					game.LogEvent("alert", planet.Owner,
						fmt.Sprintf("⛏️ %s deposit on %s running low! Abundance: %d. Explore for new deposits or import!",
							res.ResourceType, planet.Name, res.Abundance))
				}
			}

			if res.Abundance == 0 {
				lastWarn := rds.depletionLog[res.GetID()]
				if tick-lastWarn > 20000 {
					rds.depletionLog[res.GetID()] = tick
					game.LogEvent("alert", planet.Owner,
						fmt.Sprintf("🚫 %s deposit on %s is EXHAUSTED! Mine produces nothing. Find new sources!",
							res.ResourceType, planet.Name))
				}
			}
		} else if !hasMine && res.Abundance > 0 && res.Abundance < 100 {
			// Slow natural recovery when not being mined
			if rand.Intn(5) == 0 {
				res.Abundance++
			}
		}
	}
}
