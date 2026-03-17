package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ResourceScarcitySystem{
		BaseSystem: NewBaseSystem("ResourceScarcity", 45),
	})
}

// ResourceScarcitySystem introduces galactic-scale resource cycles.
// Every ~20,000 ticks a random resource enters a "scarcity period"
// where its base market price doubles and production slows.
//
// This simulates real-world commodity cycles:
//   - Oil shock: Oil production drops 30%, price spikes
//   - Water drought: Water scarce, population growth slows
//   - Iron shortage: Ship construction takes longer
//   - Electronics famine: Tech advancement stalls
//
// Scarcity lasts 5000-10000 ticks, then a "boom" follows where
// production increases 50% for the same duration.
//
// Factions who stockpile during cheap periods profit during scarcity.
// Factions who over-consume during booms suffer during the next crunch.
// This creates resource speculation as a viable strategy.
type ResourceScarcitySystem struct {
	*BaseSystem
	activeCycle *ScarcityCycle
	nextCycle   int64
	history     []string // recent cycle announcements
}

// ScarcityCycle represents an active resource boom or bust.
type ScarcityCycle struct {
	Resource  string
	Phase     string  // "scarcity" or "boom"
	Intensity float64 // 0.5-2.0 production multiplier
	TicksLeft int
}

func (rss *ResourceScarcitySystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := rss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if rss.nextCycle == 0 {
		rss.nextCycle = tick + 5000 + int64(rand.Intn(10000))
	}

	// Process active cycle
	if rss.activeCycle != nil {
		rss.activeCycle.TicksLeft -= 500
		if rss.activeCycle.TicksLeft <= 0 {
			// Cycle ends — announce
			if rss.activeCycle.Phase == "scarcity" {
				// Transition to boom
				game.LogEvent("event", "",
					fmt.Sprintf("📈 %s scarcity has ended! Entering BOOM phase — production +50%% for the next period",
						rss.activeCycle.Resource))
				rss.activeCycle = &ScarcityCycle{
					Resource:  rss.activeCycle.Resource,
					Phase:     "boom",
					Intensity: 1.5,
					TicksLeft: 5000 + rand.Intn(5000),
				}
			} else {
				// Boom ends
				game.LogEvent("event", "",
					fmt.Sprintf("📊 %s boom has ended. Markets returning to normal",
						rss.activeCycle.Resource))
				rss.activeCycle = nil
			}
		}
	}

	// Start new cycle
	if rss.activeCycle == nil && tick >= rss.nextCycle {
		rss.nextCycle = tick + 20000 + int64(rand.Intn(15000))
		rss.startNewCycle(game)
	}

	// Apply production effects
	if rss.activeCycle != nil {
		rss.applyProductionEffect(game)
	}
}

func (rss *ResourceScarcitySystem) startNewCycle(game GameProvider) {
	resources := []string{entities.ResOil, entities.ResIron, entities.ResWater,
		entities.ResRareMetals, entities.ResHelium3}
	res := resources[rand.Intn(len(resources))]

	rss.activeCycle = &ScarcityCycle{
		Resource:  res,
		Phase:     "scarcity",
		Intensity: 0.5 + rand.Float64()*0.3, // 50-80% of normal production
		TicksLeft: 5000 + rand.Intn(5000),
	}

	market := game.GetMarketEngine()
	if market != nil {
		// Spike demand to drive up prices
		market.AddTradeVolume(res, 1000, true)
	}

	game.LogEvent("event", "",
		fmt.Sprintf("⚠️ GALACTIC %s SHORTAGE! Production at %.0f%% of normal. Prices spiking! Stockpilers will profit!",
			res, rss.activeCycle.Intensity*100))
}

func (rss *ResourceScarcitySystem) applyProductionEffect(game GameProvider) {
	// This modifies the market to reflect scarcity/boom
	// During scarcity: drive up prices via demand spikes
	// During boom: drive down prices via supply increase
	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	if rss.activeCycle.Phase == "scarcity" {
		// Add artificial demand pressure
		market.AddTradeVolume(rss.activeCycle.Resource, 100, true)
	} else {
		// Add artificial supply pressure
		market.AddTradeVolume(rss.activeCycle.Resource, 100, false)
	}
}

// GetActiveCycle returns the current scarcity/boom cycle (for API).
func (rss *ResourceScarcitySystem) GetActiveCycle() *ScarcityCycle {
	return rss.activeCycle
}

// GetProductionMultiplier returns the production multiplier for a resource.
// Returns 1.0 if no cycle is active for that resource.
func (rss *ResourceScarcitySystem) GetProductionMultiplier(resource string) float64 {
	if rss.activeCycle != nil && rss.activeCycle.Resource == resource {
		return rss.activeCycle.Intensity
	}
	return 1.0
}
