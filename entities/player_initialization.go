package entities

import (
	"math/rand"
)

// InitializePlayer sets up a new player with a starting planet
func InitializePlayer(player *Player, systems []*System) {
	// Find systems with habitable terrestrial planets that have both Oil and Iron
	validSystems := make([]*System, 0)

	for _, system := range systems {
		// Check if system has habitable terrestrial planets with Oil and Iron
		for _, entity := range system.GetEntities() {
			if planet, ok := entity.(*Planet); ok {
				if planet.Owner != "" {
					continue
				}
				if planet.PlanetType == "Terrestrial" && planet.IsHabitable() && hasResource(planet, ResWater) && hasResource(planet, ResIron) && hasResource(planet, ResOil) {
					validSystems = append(validSystems, system)
					break
				}
			}
		}
	}

	// Fallback: habitable terrestrial with at least Water
	if len(validSystems) == 0 {
		for _, system := range systems {
			for _, entity := range system.GetEntities() {
				if planet, ok := entity.(*Planet); ok {
					if planet.Owner != "" {
						continue
					}
					if planet.PlanetType == "Terrestrial" && planet.IsHabitable() && hasResource(planet, ResWater) {
						validSystems = append(validSystems, system)
						break
					}
				}
			}
		}
	}

	// Second fallback: any habitable planet with Water
	if len(validSystems) == 0 {
		for _, system := range systems {
			for _, entity := range system.GetEntitiesByType(EntityTypePlanet) {
				if planet, ok := entity.(*Planet); ok {
					if planet.Owner != "" {
						continue
					}
					if planet.IsHabitable() && hasResource(planet, ResWater) {
						validSystems = append(validSystems, system)
						break
					}
				}
			}
		}
	}

	// Last resort: any habitable planet at all
	if len(validSystems) == 0 {
		for _, system := range systems {
			for _, entity := range system.GetEntitiesByType(EntityTypePlanet) {
				if planet, ok := entity.(*Planet); ok {
					if planet.Owner != "" {
						continue
					}
					if planet.IsHabitable() {
						validSystems = append(validSystems, system)
						break
					}
				}
			}
		}
	}

	if len(validSystems) == 0 {
		return // No valid systems found
	}

	// Pick a random system
	homeSystem := validSystems[rand.Intn(len(validSystems))]
	player.HomeSystem = homeSystem

	// Find the best habitable planet — prefer one with Water + Iron
	var bestPlanet *Planet
	bestScore := -1

	for _, entity := range homeSystem.GetEntities() {
		if planet, ok := entity.(*Planet); ok {
			if planet.Owner != "" || !planet.IsHabitable() {
				continue
			}
			score := planet.Habitability
			if hasResource(planet, ResWater) {
				score += 100 // Strongly prefer Water
			}
			if hasResource(planet, ResIron) {
				score += 50
			}
			if hasResource(planet, ResOil) {
				score += 80 // Oil is critical for Fuel production
			}
			if score > bestScore {
				bestPlanet = planet
				bestScore = score
			}
		}
	}

	// Fallback: any habitable planet in this system
	if bestPlanet == nil {
		for _, entity := range homeSystem.GetEntities() {
			if planet, ok := entity.(*Planet); ok {
				if planet.Owner == "" && planet.IsHabitable() {
					bestPlanet = planet
					break
				}
			}
		}
	}

	if bestPlanet == nil {
		return
	}

	// Set up the home planet
	player.HomePlanet = bestPlanet
	bestPlanet.Population = 2000 // 2,000 starting colonists
	bestPlanet.Owner = player.Name
	bestPlanet.SetBaseOwner(player.Name)

	// Mark all resources on the home planet as owned by player
	for _, resource := range bestPlanet.Resources {
		if res, ok := resource.(*Resource); ok {
			res.Owner = player.Name
		}
	}

	player.AddOwnedPlanet(bestPlanet)
	bestPlanet.RebalanceWorkforce()

	// Create starting scout ship
	scoutShip := NewShip(
		1000+player.ID, // Unique ship ID
		player.Name+" Scout",
		ShipTypeScout,
		homeSystem.ID,
		player.Name,
		player.Color,
	)

	// Position ship in orbit around home planet
	scoutShip.OrbitDistance = bestPlanet.GetOrbitDistance()
	scoutShip.OrbitAngle = rand.Float64() * 6.28318 // Random angle around planet

	// Add ship to home system and player
	homeSystem.AddEntity(scoutShip)
	player.AddOwnedShip(scoutShip)
}

// hasResource checks if a planet has a resource deposit of the given type.
func hasResource(planet *Planet, resType string) bool {
	for _, resourceEntity := range planet.Resources {
		if resource, ok := resourceEntity.(*Resource); ok {
			if resource.ResourceType == resType {
				return true
			}
		}
	}
	return false
}
