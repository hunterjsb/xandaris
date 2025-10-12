package main

import (
	"fmt"
	"image/color"
	"math/rand"
)

// Planet represents a planet entity
type Planet struct {
	ID            int
	Name          string
	Color         color.RGBA
	OrbitDistance float64
	OrbitAngle    float64
	Size          int // Radius in pixels
	PlanetType    string
	Population    int64
	Resources     []string
	Temperature   int // In Celsius
	Atmosphere    string
	HasRings      bool
}

func (p *Planet) GetID() int                { return p.ID }
func (p *Planet) GetName() string           { return p.Name }
func (p *Planet) GetType() EntityType       { return "Planet" }
func (p *Planet) GetOrbitDistance() float64 { return p.OrbitDistance }
func (p *Planet) GetOrbitAngle() float64    { return p.OrbitAngle }
func (p *Planet) GetColor() color.RGBA      { return p.Color }

func (p *Planet) GetDescription() string {
	return fmt.Sprintf("%s (%s)", p.Name, p.PlanetType)
}

// GetDetailedInfo returns detailed information about the planet
func (p *Planet) GetDetailedInfo() map[string]string {
	return map[string]string{
		"Type":        p.PlanetType,
		"Population":  fmt.Sprintf("%d", p.Population),
		"Temperature": fmt.Sprintf("%d°C", p.Temperature),
		"Atmosphere":  p.Atmosphere,
		"Size":        fmt.Sprintf("%d km radius", p.Size*1000),
	}
}

// Planet type constants
const (
	PlanetTypeTerrestrial = "Terrestrial"
	PlanetTypeGasGiant    = "Gas Giant"
	PlanetTypeIce         = "Ice World"
	PlanetTypeDesert      = "Desert"
	PlanetTypeOcean       = "Ocean"
	PlanetTypeLava        = "Lava"
)

// GetPlanetTypes returns all available planet types
func GetPlanetTypes() []string {
	return []string{
		PlanetTypeTerrestrial,
		PlanetTypeGasGiant,
		PlanetTypeIce,
		PlanetTypeDesert,
		PlanetTypeOcean,
		PlanetTypeLava,
	}
}

// GeneratePlanets creates random planets for a system
func GeneratePlanets(systemID int, count int) []*Planet {
	planets := make([]*Planet, 0)
	planetTypes := GetPlanetTypes()
	planetColors := GetPlanetColors()

	atmospheres := []string{"Breathable", "Toxic", "Thin", "None", "Dense", "Corrosive"}

	for i := 0; i < count; i++ {
		typeIdx := rand.Intn(len(planetTypes))
		planetType := planetTypes[typeIdx]

		// Generate planet properties based on type
		var temperature int
		var atmosphere string
		var population int64

		switch planetType {
		case PlanetTypeTerrestrial:
			temperature = -20 + rand.Intn(60) // -20 to 40°C
			atmosphere = atmospheres[rand.Intn(3)]
			population = int64(rand.Intn(2000000000))
		case PlanetTypeGasGiant:
			temperature = -150 + rand.Intn(50) // -150 to -100°C
			atmosphere = "Dense"
			population = 0 // Gas giants typically uninhabitable
		case PlanetTypeIce:
			temperature = -80 + rand.Intn(40)           // -80 to -40°C
			atmosphere = atmospheres[2:4][rand.Intn(2)] // Thin or None
			population = int64(rand.Intn(50000000))
		case PlanetTypeDesert:
			temperature = 20 + rand.Intn(80)            // 20 to 100°C
			atmosphere = atmospheres[1:3][rand.Intn(2)] // Toxic or Thin
			population = int64(rand.Intn(500000000))
		case PlanetTypeOcean:
			temperature = 0 + rand.Intn(40)             // 0 to 40°C
			atmosphere = atmospheres[0:2][rand.Intn(2)] // Breathable or Toxic
			population = int64(rand.Intn(3000000000))
		case PlanetTypeLava:
			temperature = 800 + rand.Intn(500) // 800 to 1300°C
			atmosphere = "Corrosive"
			population = 0 // Lava planets are uninhabitable
		}

		planet := &Planet{
			ID:            systemID*1000 + i,
			Name:          fmt.Sprintf("Planet %d", i+1),
			Color:         planetColors[typeIdx],
			OrbitDistance: 30.0 + float64(i)*20.0,
			OrbitAngle:    rand.Float64() * 6.28,
			Size:          4 + rand.Intn(4), // 4-7 pixels
			PlanetType:    planetType,
			Population:    population,
			Resources:     generatePlanetResources(planetType),
			Temperature:   temperature,
			Atmosphere:    atmosphere,
			HasRings:      rand.Float32() < 0.15, // 15% chance of rings
		}

		planets = append(planets, planet)
	}

	return planets
}

// generatePlanetResources generates resources based on planet type
func generatePlanetResources(planetType string) []string {
	resourcePool := map[string][]string{
		PlanetTypeTerrestrial: {"Iron", "Water", "Food", "Rare Metals"},
		PlanetTypeGasGiant:    {"Helium-3", "Hydrogen", "Gas Mining"},
		PlanetTypeIce:         {"Water", "Ice", "Frozen Gases"},
		PlanetTypeDesert:      {"Silicon", "Rare Minerals", "Solar Energy"},
		PlanetTypeOcean:       {"Water", "Food", "Algae", "Deep Sea Minerals"},
		PlanetTypeLava:        {"Rare Metals", "Geothermal Energy", "Volcanic Glass"},
	}

	resources := resourcePool[planetType]
	if len(resources) == 0 {
		return []string{"Unknown"}
	}

	// Return 1-3 random resources from the pool
	count := 1 + rand.Intn(3)
	if count > len(resources) {
		count = len(resources)
	}

	result := make([]string, 0)
	used := make(map[int]bool)

	for len(result) < count {
		idx := rand.Intn(len(resources))
		if !used[idx] {
			result = append(result, resources[idx])
			used[idx] = true
		}
	}

	return result
}

// IsHabitable returns whether the planet can support life
func (p *Planet) IsHabitable() bool {
	return p.Temperature > -50 && p.Temperature < 60 &&
		p.Atmosphere != "None" && p.Atmosphere != "Corrosive" &&
		p.PlanetType != PlanetTypeLava
}

// GetHabitabilityScore returns a habitability score from 0-100
func (p *Planet) GetHabitabilityScore() int {
	score := 50 // Base score

	// Temperature scoring
	if p.Temperature >= -10 && p.Temperature <= 30 {
		score += 30 // Ideal temperature
	} else if p.Temperature >= -30 && p.Temperature <= 50 {
		score += 15 // Acceptable temperature
	} else {
		score -= 20 // Poor temperature
	}

	// Atmosphere scoring
	switch p.Atmosphere {
	case "Breathable":
		score += 20
	case "Thin":
		score += 5
	case "Dense":
		score -= 5
	case "Toxic":
		score -= 15
	case "Corrosive":
		score -= 30
	case "None":
		score -= 25
	}

	// Planet type bonus/penalty
	switch p.PlanetType {
	case PlanetTypeTerrestrial:
		score += 10
	case PlanetTypeOcean:
		score += 15
	case PlanetTypeLava:
		score -= 40
	case PlanetTypeIce:
		score -= 10
	}

	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}
