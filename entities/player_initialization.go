package entities

import (
	"math/rand"
)

// InitializePlayer sets up a new player with a starting planet
func InitializePlayer(player *Player, systems []*System) {
	// Find systems with habitable terrestrial planets
	validSystems := make([]*System, 0)

	for _, system := range systems {
		// Check if system has habitable terrestrial planets
		for _, entity := range system.GetEntities() {
			if planet, ok := entity.(*Planet); ok {
				if planet.PlanetType == "Terrestrial" && planet.IsHabitable() {
					validSystems = append(validSystems, system)
					break
				}
			}
		}
	}

	// Fallback: use any system with a planet
	if len(validSystems) == 0 {
		for _, system := range systems {
			if system.HasEntityType(EntityTypePlanet) {
				validSystems = append(validSystems, system)
			}
		}
	}

	if len(validSystems) == 0 {
		return // No valid systems found
	}

	// Pick a random system
	homeSystem := validSystems[rand.Intn(len(validSystems))]
	player.HomeSystem = homeSystem

	// Find the best terrestrial planet in that system
	var bestPlanet *Planet
	bestHabitability := 0

	for _, entity := range homeSystem.GetEntities() {
		if planet, ok := entity.(*Planet); ok {
			if planet.PlanetType == "Terrestrial" && planet.Habitability > bestHabitability {
				bestPlanet = planet
				bestHabitability = planet.Habitability
			}
		}
	}

	// If no terrestrial planet, pick any habitable planet
	if bestPlanet == nil {
		for _, entity := range homeSystem.GetEntities() {
			if planet, ok := entity.(*Planet); ok {
				if planet.IsHabitable() {
					bestPlanet = planet
					break
				}
			}
		}
	}

	// Last resort: pick any planet
	if bestPlanet == nil {
		for _, entity := range homeSystem.GetEntities() {
			if planet, ok := entity.(*Planet); ok {
				bestPlanet = planet
				break
			}
		}
	}

	if bestPlanet != nil {
		// Set up the home planet
		player.HomePlanet = bestPlanet
		bestPlanet.Population = 100000000 // 100 million starting population
		bestPlanet.Owner = player.Name    // Mark planet as owned by player

		// Mark all resources on the home planet as owned by player
		for _, resource := range bestPlanet.Resources {
			if res, ok := resource.(*Resource); ok {
				res.Owner = player.Name
			}
		}

		player.AddOwnedPlanet(bestPlanet)
	}
}
