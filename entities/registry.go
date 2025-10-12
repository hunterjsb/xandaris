package entities

import (
	"fmt"
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
	Generate(params GenerationParams) Entity

	// GetWeight returns the spawn probability weight (higher = more common)
	GetWeight() float64

	// GetEntityType returns the type of entity this generates (e.g., "Planet", "Station")
	GetEntityType() EntityType

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
func GetGeneratorsByType(entityType EntityType) []EntityGenerator {
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
	fmt.Println("[WARNING] Fallback to last generator")
	return generators[len(generators)-1]
}

// GenerateEntitiesForSystem generates a collection of entities for a system
func GenerateEntitiesForSystem(systemID int, seed int64) []Entity {
	rng := rand.New(rand.NewSource(seed))
	entities := make([]Entity, 0)

	// Generate exactly one star per system (always first)
	starGenerators := GetGeneratorsByType(EntityTypeStar)
	if len(starGenerators) > 0 {
		gen := SelectRandomGenerator(starGenerators)
		params := GenerationParams{
			SystemID:      systemID,
			OrbitDistance: 0.0, // Stars are at the center
			OrbitAngle:    0.0,
			SystemSeed:    seed,
		}
		entity := gen.Generate(params)
		entities = append(entities, entity)
	}

	// Generate planets (2-6 per system)
	planetCount := 2 + rng.Intn(5)
	planetGenerators := GetGeneratorsByType(EntityTypePlanet)
	if len(planetGenerators) > 0 {
		for i := 0; i < planetCount; i++ {
			gen := SelectRandomGenerator(planetGenerators)
			params := GenerationParams{
				SystemID:      systemID,
				OrbitDistance: 50.0 + float64(i)*30.0,
				OrbitAngle:    rng.Float64() * 6.28,
				SystemSeed:    seed,
			}
			entity := gen.Generate(params)
			entities = append(entities, entity)
		}
	}

	// Generate stations (0-1 per system, 40% chance)
	if rng.Float32() < 0.4 {
		stationGenerators := GetGeneratorsByType(EntityTypeStation)
		if len(stationGenerators) > 0 {
			gen := SelectRandomGenerator(stationGenerators)
			params := GenerationParams{
				SystemID:      systemID,
				OrbitDistance: 200.0 + rng.Float64()*100.0,
				OrbitAngle:    rng.Float64() * 6.28,
				SystemSeed:    seed,
			}
			entity := gen.Generate(params)
			entities = append(entities, entity)
		}
	}

	return entities
}

// GetRegistryStats returns statistics about registered generators
func GetRegistryStats() map[EntityType]int {
	stats := make(map[EntityType]int)
	for _, gen := range registry {
		stats[gen.GetEntityType()]++
	}
	return stats
}
