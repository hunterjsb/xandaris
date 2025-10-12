package entities

import (
	"math/rand"
)

// GenerationParams contains parameters for entity generation
type GenerationParams struct {
	SystemID      int
	OrbitDistance float64
	OrbitAngle    float64
	SystemSeed    int64
}

// EntityGenerator is the interface that all entity generators must implement
type EntityGenerator interface {
	// Generate creates a new entity instance
	Generate(params GenerationParams) interface{}

	// GetWeight returns the spawn probability weight (higher = more common)
	GetWeight() float64

	// GetEntityType returns the type of entity this generates (e.g., "Planet", "Station")
	GetEntityType() string

	// GetSubType returns the subtype (e.g., "Military", "Lava", "Trading")
	GetSubType() string
}

// Registry holds all registered entity generators
var registry []EntityGenerator

// RegisterGenerator adds a generator to the registry
// This should be called in init() functions of entity files
func RegisterGenerator(gen EntityGenerator) {
	registry = append(registry, gen)
}

// GetGeneratorsByType returns all generators of a specific entity type
func GetGeneratorsByType(entityType string) []EntityGenerator {
	var result []EntityGenerator
	for _, gen := range registry {
		if gen.GetEntityType() == entityType {
			result = append(result, gen)
		}
	}
	return result
}

// GetAllGenerators returns all registered generators
func GetAllGenerators() []EntityGenerator {
	return registry
}

// SelectRandomGenerator picks a random generator from a list based on weights
func SelectRandomGenerator(generators []EntityGenerator) EntityGenerator {
	if len(generators) == 0 {
		return nil
	}

	// Calculate total weight
	totalWeight := 0.0
	for _, gen := range generators {
		totalWeight += gen.GetWeight()
	}

	// Pick random value
	r := rand.Float64() * totalWeight

	// Find the generator
	currentWeight := 0.0
	for _, gen := range generators {
		currentWeight += gen.GetWeight()
		if r <= currentWeight {
			return gen
		}
	}

	// Fallback to last generator (shouldn't happen)
	return generators[len(generators)-1]
}

// GenerateEntitiesForSystem generates a collection of entities for a system
func GenerateEntitiesForSystem(systemID int, seed int64) []interface{} {
	rand.Seed(seed)
	entities := make([]interface{}, 0)

	// Generate planets (2-6 per system)
	planetCount := 2 + rand.Intn(5)
	planetGenerators := GetGeneratorsByType("Planet")
	if len(planetGenerators) > 0 {
		for i := 0; i < planetCount; i++ {
			gen := SelectRandomGenerator(planetGenerators)
			params := GenerationParams{
				SystemID:      systemID,
				OrbitDistance: 30.0 + float64(i)*20.0,
				OrbitAngle:    rand.Float64() * 6.28,
				SystemSeed:    seed,
			}
			entity := gen.Generate(params)
			entities = append(entities, entity)
		}
	}

	// Generate stations (0-1 per system, 40% chance)
	if rand.Float32() < 0.4 {
		stationGenerators := GetGeneratorsByType("Station")
		if len(stationGenerators) > 0 {
			gen := SelectRandomGenerator(stationGenerators)
			params := GenerationParams{
				SystemID:      systemID,
				OrbitDistance: 70.0 + rand.Float64()*30.0,
				OrbitAngle:    rand.Float64() * 6.28,
				SystemSeed:    seed,
			}
			entity := gen.Generate(params)
			entities = append(entities, entity)
		}
	}

	return entities
}
