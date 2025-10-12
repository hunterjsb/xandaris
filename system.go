package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

const (
	systemCount      = 40
	circleRadius     = 8
	maxHyperlanes    = 3
	minDistance      = 60.0
	maxDistance      = 180.0
	minSystemSpacing = 45.0
)

// System represents a star system
type System struct {
	ID          int
	X           float64
	Y           float64
	Name        string
	Color       color.RGBA
	Connections []int // IDs of connected systems
	Entities    []entities.Entity
}

// Hyperlane represents a connection between two systems
type Hyperlane struct {
	From int
	To   int
}

// generateSystems creates systems at random coordinates
func (g *Game) generateSystems() {
	colors := GetSystemColors()

	// Generate systems with random positions
	for i := 0; i < systemCount; i++ {
		var x, y float64
		var validPosition bool
		attempts := 0

		// Keep trying until we find a position that's not too close to existing systems
		for !validPosition && attempts < 200 {
			x = 80 + rand.Float64()*(screenWidth-160)
			y = 80 + rand.Float64()*(screenHeight-160)
			validPosition = true

			// Check distance to all existing systems
			for _, existing := range g.systems {
				distance := math.Sqrt(math.Pow(x-existing.X, 2) + math.Pow(y-existing.Y, 2))
				if distance < minSystemSpacing {
					validPosition = false
					break
				}
			}
			attempts++
		}

		system := &System{
			ID:          i,
			X:           x,
			Y:           y,
			Name:        fmt.Sprintf("SYS-%d", i+1),
			Color:       colors[rand.Intn(len(colors))],
			Connections: make([]int, 0),
		}

		g.systems = append(g.systems, system)

		// Generate entities for this system using the new entity generator system
		seed := int64(i) + g.seed
		generatedEntities := entities.GenerateEntitiesForSystem(i, seed)
		for _, entity := range generatedEntities {
			system.AddEntity(entity)
		}
	}
}

// generateHyperlanes creates connections between systems
func (g *Game) generateHyperlanes() {
	for _, system := range g.systems {
		// Find nearby systems for potential connections
		var nearbySystemsWithDistance []struct {
			system   *System
			distance float64
		}

		for _, other := range g.systems {
			if other.ID == system.ID {
				continue
			}

			distance := math.Sqrt(math.Pow(system.X-other.X, 2) + math.Pow(system.Y-other.Y, 2))
			if distance >= minDistance && distance <= maxDistance {
				nearbySystemsWithDistance = append(nearbySystemsWithDistance, struct {
					system   *System
					distance float64
				}{other, distance})
			}
		}

		// Sort by distance (closest first)
		for i := 0; i < len(nearbySystemsWithDistance)-1; i++ {
			for j := i + 1; j < len(nearbySystemsWithDistance); j++ {
				if nearbySystemsWithDistance[i].distance > nearbySystemsWithDistance[j].distance {
					nearbySystemsWithDistance[i], nearbySystemsWithDistance[j] = nearbySystemsWithDistance[j], nearbySystemsWithDistance[i]
				}
			}
		}

		// Connect to closest systems (max connections per system)
		connectionsToMake := maxHyperlanes
		if len(nearbySystemsWithDistance) < maxHyperlanes {
			connectionsToMake = len(nearbySystemsWithDistance)
		}

		for i := 0; i < connectionsToMake; i++ {
			other := nearbySystemsWithDistance[i].system

			// Check if connection already exists
			connectionExists := false
			for _, hyperlane := range g.hyperlanes {
				if (hyperlane.From == system.ID && hyperlane.To == other.ID) ||
					(hyperlane.From == other.ID && hyperlane.To == system.ID) {
					connectionExists = true
					break
				}
			}

			if !connectionExists {
				// Add hyperlane
				g.hyperlanes = append(g.hyperlanes, Hyperlane{
					From: system.ID,
					To:   other.ID,
				})

				// Add to both systems' connection lists
				system.Connections = append(system.Connections, other.ID)
				other.Connections = append(other.Connections, system.ID)
			}
		}
	}
}

// GetContextMenuTitle implements ContextMenuProvider
func (s *System) GetContextMenuTitle() string {
	return s.Name
}

// GetContextMenuItems implements ContextMenuProvider
func (s *System) GetContextMenuItems() []string {
	items := []string{}

	// Add star information first
	starEntities := s.GetEntitiesByType(entities.EntityTypeStar)
	if len(starEntities) > 0 {
		star := starEntities[0]
		items = append(items, fmt.Sprintf("Star: %s", star.GetDescription()))
		items = append(items, "") // Empty line for spacing
	}

	// Add entity counts summary
	planetCount := len(s.GetEntitiesByType(entities.EntityTypePlanet))
	stationCount := len(s.GetEntitiesByType(entities.EntityTypeStation))

	items = append(items, fmt.Sprintf("Planets: %d", planetCount))
	if stationCount > 0 {
		items = append(items, fmt.Sprintf("Stations: %d", stationCount))
	}
	items = append(items, "") // Empty line for spacing

	// List planets
	for _, entity := range s.GetEntitiesByType(entities.EntityTypePlanet) {
		items = append(items, fmt.Sprintf("  - %s", entity.GetDescription()))
	}

	// List stations
	for _, entity := range s.GetEntitiesByType(entities.EntityTypeStation) {
		items = append(items, fmt.Sprintf("  - %s", entity.GetDescription()))
	}

	return items
}

// GetPosition implements Clickable interface
func (s *System) GetPosition() (float64, float64) {
	return s.X, s.Y
}

// GetClickRadius implements Clickable interface
func (s *System) GetClickRadius() float64 {
	return float64(circleRadius)
}

// HasOwnershipByPlayer checks if the system contains any planets owned by the specified player
func (s *System) HasOwnershipByPlayer(playerName string) bool {
	for _, entity := range s.Entities {
		if planet, ok := entity.(*entities.Planet); ok {
			if planet.Owner == playerName {
				return true
			}
		}
		if station, ok := entity.(*entities.Station); ok {
			if station.IsPlayerOwned() && station.Owner == playerName {
				return true
			}
		}
	}
	return false
}
