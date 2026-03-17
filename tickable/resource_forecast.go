package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ResourceForecastSystem{
		BaseSystem: NewBaseSystem("ResourceForecast", 139),
	})
}

// ResourceForecastSystem predicts which resources will be in shortage
// soon based on consumption trends. If a resource is being consumed
// faster than produced, it forecasts when stockpiles will hit zero.
//
// Per planet, tracks:
//   - Current stored amount
//   - Change since last check (consumption rate)
//   - Predicted ticks until depletion
//
// Warns factions 5000 ticks before a critical resource hits zero:
//   "⚠️ Planet X: Fuel will deplete in ~8 minutes at current rate!"
//
// This gives factions time to import or adjust before a crisis.
type ResourceForecastSystem struct {
	*BaseSystem
	prevStored map[int]map[string]int // planetID → resource → amount at last check
	lastWarn   map[int]int64          // planetID → last warning tick
}

func (rfs *ResourceForecastSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := rfs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if rfs.prevStored == nil {
		rfs.prevStored = make(map[int]map[string]int)
		rfs.lastWarn = make(map[int]int64)
	}

	systems := game.GetSystems()
	criticalResources := []string{entities.ResFuel, entities.ResWater, entities.ResIron}

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Population < 100 {
				continue
			}

			pid := planet.GetID()

			if rfs.prevStored[pid] == nil {
				rfs.prevStored[pid] = make(map[string]int)
				for _, res := range criticalResources {
					rfs.prevStored[pid][res] = planet.GetStoredAmount(res)
				}
				continue
			}

			for _, res := range criticalResources {
				current := planet.GetStoredAmount(res)
				prev := rfs.prevStored[pid][res]
				rfs.prevStored[pid][res] = current

				if current >= prev || current > 100 {
					continue // not declining or has plenty
				}

				// Calculate depletion rate
				consumeRate := prev - current // per 1000 ticks
				if consumeRate <= 0 {
					continue
				}

				ticksUntilZero := int64(current / consumeRate * 1000)
				if ticksUntilZero > 5000 || ticksUntilZero <= 0 {
					continue // not urgent
				}

				// Rate limit warnings
				if tick-rfs.lastWarn[pid] < 5000 {
					continue
				}
				rfs.lastWarn[pid] = tick

				minutesLeft := ticksUntilZero / 600
				if minutesLeft < 1 {
					minutesLeft = 1
				}

				if rand.Intn(2) == 0 { // don't spam every check
					game.LogEvent("alert", planet.Owner,
						fmt.Sprintf("⚠️ %s: %s depleting! %d remaining, ~%d min until zero at current rate. Import or reduce consumption!",
							planet.Name, res, current, minutesLeft))
				}
			}
		}
	}
}
