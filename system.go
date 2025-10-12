package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
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
	Entities    []Entity
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

		// Generate entities for this system
		planetCount := 2 + rand.Intn(5) // 2-6 planets
		planets := GeneratePlanets(i, planetCount)
		for _, planet := range planets {
			system.AddEntity(planet)
		}

		// 40% chance of having a space station
		if rand.Float32() < 0.4 {
			station := GenerateSpaceStation(i, 70.0+rand.Float64()*30.0)
			system.AddEntity(station)
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
