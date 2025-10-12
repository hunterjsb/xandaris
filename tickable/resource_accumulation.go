package tickable

import (
	"sync"
)

func init() {
	RegisterSystem(&ResourceAccumulationSystem{
		BaseSystem: NewBaseSystem("ResourceAccumulation", 10),
	})
}

// ResourceAccumulationSystem handles resource generation from planets
type ResourceAccumulationSystem struct {
	*BaseSystem
	tickCounter int64
}

// OnTick processes resource accumulation each tick
func (ras *ResourceAccumulationSystem) OnTick(tick int64) {
	ras.tickCounter++

	// Only accumulate resources every 10 ticks (once per second at 1x speed)
	if ras.tickCounter%10 != 0 {
		return
	}

	context := ras.GetContext()
	if context == nil {
		return
	}

	// Get players from context
	playersInterface := context.GetPlayers()
	if playersInterface == nil {
		return
	}

	// Type assert to player slice
	players, ok := playersInterface.([]*interface{})
	if !ok {
		return
	}

	// Process all players concurrently
	var wg sync.WaitGroup
	for _, playerInterface := range players {
		wg.Add(1)
		go func(pInterface *interface{}) {
			defer wg.Done()
			ras.processPlayer(pInterface)
		}(playerInterface)
	}
	wg.Wait()
}

// processPlayer handles resource accumulation for a single player
func (ras *ResourceAccumulationSystem) processPlayer(playerInterface *interface{}) {
	// This would need proper type assertion based on actual Player type
	// For now, this is a template that shows the structure

	// In the actual implementation, you would:
	// 1. Get the player's owned planets
	// 2. Process each planet concurrently using ProcessConcurrent
	// 3. Calculate production based on population, habitability, resources
	// 4. Safely accumulate to player's credits using mutex

	// Example structure:
	// player := (*playerInterface).(PlayerType)
	// planets := player.GetOwnedPlanets()
	//
	// var totalProduction int64
	// var mu sync.Mutex
	//
	// ProcessConcurrent(planets, 4, func(planet PlanetType) {
	//     production := calculateProduction(planet)
	//     mu.Lock()
	//     totalProduction += production
	//     mu.Unlock()
	// })
	//
	// player.AddCredits(totalProduction)
}

// calculateProduction calculates resource production for a planet
func (ras *ResourceAccumulationSystem) calculateProduction(planetInterface interface{}) int64 {
	// Base production: 1 credit per million population per second
	// Bonus for habitability
	// Bonus for owned resources
	// Multipliers from buildings/improvements

	// This is a placeholder calculation
	production := int64(100)

	return production
}

// GetProductionRate returns the current production rate per second for a player
func (ras *ResourceAccumulationSystem) GetProductionRate(playerInterface interface{}) int64 {
	// Calculate and return total production rate
	// Useful for UI display
	return 0
}

// GetProductionBreakdown returns detailed production info per planet
func (ras *ResourceAccumulationSystem) GetProductionBreakdown(playerInterface interface{}) map[string]int64 {
	// Return map of planet name -> production rate
	// Useful for detailed economy view
	return make(map[string]int64)
}
