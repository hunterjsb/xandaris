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
				if planet.PlanetType == "Terrestrial" && planet.IsHabitable() && hasOilResource(planet) && hasIronResource(planet) {
					validSystems = append(validSystems, system)
					break
				}
			}
		}
	}

	// Fallback: Find any habitable terrestrial planet (without Oil requirement)
	if len(validSystems) == 0 {
		for _, system := range systems {
			for _, entity := range system.GetEntities() {
				if planet, ok := entity.(*Planet); ok {
					if planet.Owner != "" {
						continue
					}
					if planet.PlanetType == "Terrestrial" && planet.IsHabitable() {
						validSystems = append(validSystems, system)
						break
					}
				}
			}
		}
	}

	// Second fallback: use any system with a habitable planet
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

	// Find the best terrestrial planet with Oil and Iron in that system
	var bestPlanet *Planet
	bestHabitability := -1

	for _, entity := range homeSystem.GetEntities() {
		if planet, ok := entity.(*Planet); ok {
			if planet.Owner != "" {
				continue
			}
			if !planet.IsHabitable() {
				continue
			}
			if planet.PlanetType == "Terrestrial" && hasOilResource(planet) && hasIronResource(planet) && planet.Habitability > bestHabitability {
				bestPlanet = planet
				bestHabitability = planet.Habitability
			}
		}
	}

	// Fallback: Find best terrestrial planet without Oil requirement
	if bestPlanet == nil {
		for _, entity := range homeSystem.GetEntities() {
			if planet, ok := entity.(*Planet); ok {
				if planet.Owner != "" {
					continue
				}
				if !planet.IsHabitable() {
					continue
				}
				if planet.PlanetType == "Terrestrial" && planet.Habitability > bestHabitability {
					bestPlanet = planet
					bestHabitability = planet.Habitability
				}
			}
		}
	}

	// If no terrestrial planet, pick any habitable planet
	if bestPlanet == nil {
		for _, entity := range homeSystem.GetEntities() {
			if planet, ok := entity.(*Planet); ok {
				if planet.Owner != "" {
					continue
				}
				if planet.IsHabitable() {
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
	bestPlanet.Population = 1000 // 1,000 starting population
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

// hasOilResource checks if a planet has an Oil resource deposit
func hasOilResource(planet *Planet) bool {
	for _, resourceEntity := range planet.Resources {
		if resource, ok := resourceEntity.(*Resource); ok {
			if resource.ResourceType == "Oil" {
				return true
			}
		}
	}
	return false
}

// hasIronResource checks if a planet has an Iron resource deposit
func hasIronResource(planet *Planet) bool {
	for _, resourceEntity := range planet.Resources {
		if resource, ok := resourceEntity.(*Resource); ok {
			if resource.ResourceType == "Iron" {
				return true
			}
		}
	}
	return false
}
