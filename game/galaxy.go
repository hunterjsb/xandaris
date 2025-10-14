package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

const (
	systemCount      = 40
	maxHyperlanes    = 3
	minDistance      = 60.0
	maxDistance      = 180.0
	minSystemSpacing = 45.0
)

// GalaxyGenerator handles galaxy and system generation
type GalaxyGenerator struct {
	screenWidth  int
	screenHeight int
}

// NewGalaxyGenerator creates a new galaxy generator
func NewGalaxyGenerator(screenWidth, screenHeight int) *GalaxyGenerator {
	return &GalaxyGenerator{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
	}
}

// GenerateSystems creates systems at random coordinates
func (gg *GalaxyGenerator) GenerateSystems(seed int64) []*entities.System {
	systems := make([]*entities.System, 0, systemCount)
	colors := getSystemColors()

	// Generate systems with random positions
	for i := 0; i < systemCount; i++ {
		var x, y float64
		var validPosition bool
		attempts := 0

		// Keep trying until we find a position that's not too close to existing systems
		for !validPosition && attempts < 200 {
			x = 80 + rand.Float64()*(float64(gg.screenWidth)-160)
			y = 80 + rand.Float64()*(float64(gg.screenHeight)-160)
			validPosition = true

			// Check distance to all existing systems
			for _, existing := range systems {
				distance := math.Sqrt(math.Pow(x-existing.X, 2) + math.Pow(y-existing.Y, 2))
				if distance < minSystemSpacing {
					validPosition = false
					break
				}
			}
			attempts++
		}

		system := &entities.System{
			ID:          i,
			X:           x,
			Y:           y,
			Name:        fmt.Sprintf("SYS-%d", i+1),
			Color:       colors[rand.Intn(len(colors))],
			Connections: make([]int, 0),
		}

		systems = append(systems, system)

		// Generate entities for this system using the entity generator system
		systemSeed := int64(i) + seed
		generatedEntities := entities.GenerateEntitiesForSystem(i, systemSeed)
		for _, entity := range generatedEntities {
			system.AddEntity(entity)
		}
	}

	return systems
}

// GenerateHyperlanes creates connections between systems
func (gg *GalaxyGenerator) GenerateHyperlanes(systems []*entities.System) []entities.Hyperlane {
	hyperlanes := make([]entities.Hyperlane, 0)

	for _, system := range systems {
		// Find nearby systems for potential connections
		var nearbySystemsWithDistance []struct {
			system   *entities.System
			distance float64
		}

		for _, other := range systems {
			if other.ID == system.ID {
				continue
			}

			distance := math.Sqrt(math.Pow(system.X-other.X, 2) + math.Pow(system.Y-other.Y, 2))
			if distance >= minDistance && distance <= maxDistance {
				nearbySystemsWithDistance = append(nearbySystemsWithDistance, struct {
					system   *entities.System
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
			for _, hyperlane := range hyperlanes {
				if (hyperlane.From == system.ID && hyperlane.To == other.ID) ||
					(hyperlane.From == other.ID && hyperlane.To == system.ID) {
					connectionExists = true
					break
				}
			}

			if !connectionExists {
				// Add hyperlane
				hyperlanes = append(hyperlanes, entities.Hyperlane{
					From: system.ID,
					To:   other.ID,
				})

				// Add to both systems' connection lists
				system.Connections = append(system.Connections, other.ID)
				other.Connections = append(other.Connections, system.ID)
			}
		}
	}

	return hyperlanes
}

// getSystemColors returns available system colors
func getSystemColors() []color.RGBA {
	return []color.RGBA{
		{100, 100, 200, 255}, // Blue
		{200, 100, 150, 255}, // Purple
		{150, 200, 100, 255}, // Green
		{200, 150, 100, 255}, // Orange
		{200, 200, 100, 255}, // Yellow
		{200, 100, 100, 255}, // Red
		{150, 150, 200, 255}, // Light Blue
		{180, 120, 180, 255}, // Pink
	}
}
