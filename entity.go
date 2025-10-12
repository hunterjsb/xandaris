package main

import (
	"fmt"
	"image/color"
	"math/rand"
)

// EntityType represents the type of entity
type EntityType int

const (
	EntityTypePlanet EntityType = iota
	EntityTypeStation
	EntityTypeFleet
	EntityTypeAsteroid
	EntityTypeStar
)

func (e EntityType) String() string {
	switch e {
	case EntityTypePlanet:
		return "Planet"
	case EntityTypeStation:
		return "Station"
	case EntityTypeFleet:
		return "Fleet"
	case EntityTypeAsteroid:
		return "Asteroid"
	case EntityTypeStar:
		return "Star"
	default:
		return "Unknown"
	}
}

// Entity is the interface that all system entities must implement
type Entity interface {
	GetID() int
	GetName() string
	GetType() EntityType
	GetOrbitDistance() float64 // Distance from system center (for orbital positioning)
	GetOrbitAngle() float64    // Angle in radians for orbital positioning
	GetColor() color.RGBA
	GetDescription() string
}

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
}

func (p *Planet) GetID() int                { return p.ID }
func (p *Planet) GetName() string           { return p.Name }
func (p *Planet) GetType() EntityType       { return EntityTypePlanet }
func (p *Planet) GetOrbitDistance() float64 { return p.OrbitDistance }
func (p *Planet) GetOrbitAngle() float64    { return p.OrbitAngle }
func (p *Planet) GetColor() color.RGBA      { return p.Color }

func (p *Planet) GetDescription() string {
	return fmt.Sprintf("%s (%s)", p.Name, p.PlanetType)
}

// SpaceStation represents a space station entity
type SpaceStation struct {
	ID            int
	Name          string
	Color         color.RGBA
	OrbitDistance float64
	OrbitAngle    float64
	StationType   string
	Capacity      int
}

func (s *SpaceStation) GetID() int                { return s.ID }
func (s *SpaceStation) GetName() string           { return s.Name }
func (s *SpaceStation) GetType() EntityType       { return EntityTypeStation }
func (s *SpaceStation) GetOrbitDistance() float64 { return s.OrbitDistance }
func (s *SpaceStation) GetOrbitAngle() float64    { return s.OrbitAngle }
func (s *SpaceStation) GetColor() color.RGBA      { return s.Color }

func (s *SpaceStation) GetDescription() string {
	return fmt.Sprintf("%s Station", s.StationType)
}

// AddEntity adds an entity to a system
func (s *System) AddEntity(entity Entity) {
	if s.Entities == nil {
		s.Entities = make([]Entity, 0)
	}
	s.Entities = append(s.Entities, entity)
}

// GetEntitiesByType returns all entities of a specific type
func (s *System) GetEntitiesByType(entityType EntityType) []Entity {
	result := make([]Entity, 0)
	for _, entity := range s.Entities {
		if entity.GetType() == entityType {
			result = append(result, entity)
		}
	}
	return result
}

// GetEntityByID finds an entity by its ID
func (s *System) GetEntityByID(id int) Entity {
	for _, entity := range s.Entities {
		if entity.GetID() == id {
			return entity
		}
	}
	return nil
}

// GeneratePlanets creates random planets for a system
func GeneratePlanets(systemID int, count int) []*Planet {
	planets := make([]*Planet, 0)

	planetTypes := []string{"Terrestrial", "Gas Giant", "Ice World", "Desert", "Ocean", "Lava"}
	planetColors := []color.RGBA{
		{100, 150, 100, 255}, // Green (Terrestrial)
		{200, 180, 150, 255}, // Tan (Gas Giant)
		{150, 200, 255, 255}, // Light Blue (Ice)
		{200, 180, 100, 255}, // Yellow (Desert)
		{50, 100, 200, 255},  // Blue (Ocean)
		{255, 100, 50, 255},  // Red (Lava)
	}

	for i := 0; i < count; i++ {
		typeIdx := rand.Intn(len(planetTypes))
		planet := &Planet{
			ID:            systemID*1000 + i, // Unique ID based on system
			Name:          fmt.Sprintf("Planet %d", i+1),
			Color:         planetColors[typeIdx],
			OrbitDistance: 30.0 + float64(i)*20.0, // Orbital rings
			OrbitAngle:    rand.Float64() * 6.28,  // Random starting position
			Size:          4 + rand.Intn(4),       // 4-7 pixels
			PlanetType:    planetTypes[typeIdx],
			Population:    int64(rand.Intn(1000000000)),
			Resources:     []string{"TBD"},
		}
		planets = append(planets, planet)
	}

	return planets
}

// GenerateSpaceStation creates a random space station
func GenerateSpaceStation(systemID int, orbitDistance float64) *SpaceStation {
	stationTypes := []string{"Trading", "Military", "Research", "Mining"}

	return &SpaceStation{
		ID:            systemID*10000 + 999, // Unique ID
		Name:          "Station Alpha",
		Color:         color.RGBA{255, 100, 100, 255},
		OrbitDistance: orbitDistance,
		OrbitAngle:    rand.Float64() * 6.28,
		StationType:   stationTypes[rand.Intn(len(stationTypes))],
		Capacity:      1000 + rand.Intn(9000),
	}
}
