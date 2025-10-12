package entities

import (
	"fmt"
	"sort"
)

// GeneratorInfo holds information about a registered generator
type GeneratorInfo struct {
	EntityType EntityType
	SubType    string
	Weight     float64
}

// GetAllGeneratorInfo returns information about all registered generators
func GetAllGeneratorInfo() []GeneratorInfo {
	info := make([]GeneratorInfo, 0, len(registry))
	for _, gen := range registry {
		info = append(info, GeneratorInfo{
			EntityType: gen.GetEntityType(),
			SubType:    gen.GetSubType(),
			Weight:     gen.GetWeight(),
		})
	}
	return info
}

// GetGeneratorCount returns the total number of registered generators
func GetGeneratorCount() int {
	return len(registry)
}

// GetGeneratorCountByType returns the count of generators for a specific type
func GetGeneratorCountByType(entityType EntityType) int {
	count := 0
	for _, gen := range registry {
		if gen.GetEntityType() == entityType {
			count++
		}
	}
	return count
}

// GetTotalWeightByType returns the sum of all weights for a specific type
func GetTotalWeightByType(entityType EntityType) float64 {
	total := 0.0
	for _, gen := range registry {
		if gen.GetEntityType() == entityType {
			total += gen.GetWeight()
		}
	}
	return total
}

// GetGeneratorProbability calculates the spawn probability for a specific generator
func GetGeneratorProbability(gen EntityGenerator) float64 {
	totalWeight := GetTotalWeightByType(gen.GetEntityType())
	if totalWeight == 0 {
		return 0
	}
	return gen.GetWeight() / totalWeight * 100.0
}

// PrintEntityStats prints comprehensive statistics about registered entities
func PrintEntityStats() {
	fmt.Println("=== Entity Generator Statistics ===")
	fmt.Printf("Total Generators: %d\n\n", GetGeneratorCount())

	// Get stats by type
	stats := GetRegistryStats()

	// Sort entity types for consistent output
	types := make([]EntityType, 0, len(stats))
	for t := range stats {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return string(types[i]) < string(types[j])
	})

	// Print by type
	for _, entityType := range types {
		count := stats[entityType]
		totalWeight := GetTotalWeightByType(entityType)

		fmt.Printf("%s: %d generators (total weight: %.1f)\n", entityType, count, totalWeight)

		// List all generators of this type
		generators := GetGeneratorsByType(entityType)
		for _, gen := range generators {
			probability := GetGeneratorProbability(gen)
			fmt.Printf("  - %-15s (weight: %.1f, probability: %.1f%%)\n",
				gen.GetSubType(), gen.GetWeight(), probability)
		}
		fmt.Println()
	}
}

// GetMostCommonGenerator returns the generator with the highest weight for a type
func GetMostCommonGenerator(entityType EntityType) EntityGenerator {
	generators := GetGeneratorsByType(entityType)
	if len(generators) == 0 {
		return nil
	}

	maxWeight := 0.0
	var mostCommon EntityGenerator

	for _, gen := range generators {
		if gen.GetWeight() > maxWeight {
			maxWeight = gen.GetWeight()
			mostCommon = gen
		}
	}

	return mostCommon
}

// GetRarestGenerator returns the generator with the lowest weight for a type
func GetRarestGenerator(entityType EntityType) EntityGenerator {
	generators := GetGeneratorsByType(entityType)
	if len(generators) == 0 {
		return nil
	}

	minWeight := generators[0].GetWeight()
	rarest := generators[0]

	for _, gen := range generators {
		if gen.GetWeight() < minWeight {
			minWeight = gen.GetWeight()
			rarest = gen
		}
	}

	return rarest
}

// ValidateRegistry checks if the registry is properly configured
func ValidateRegistry() []string {
	warnings := make([]string, 0)

	// Check if registry is empty
	if len(registry) == 0 {
		warnings = append(warnings, "No generators registered!")
		return warnings
	}

	// Check each generator
	for i, gen := range registry {
		// Check for zero or negative weights
		if gen.GetWeight() <= 0 {
			warnings = append(warnings, fmt.Sprintf(
				"Generator %d (%s/%s) has invalid weight: %.2f",
				i, gen.GetEntityType(), gen.GetSubType(), gen.GetWeight()))
		}

		// Check for empty subtype
		if gen.GetSubType() == "" {
			warnings = append(warnings, fmt.Sprintf(
				"Generator %d (%s) has empty subtype",
				i, gen.GetEntityType()))
		}
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, gen := range registry {
		key := fmt.Sprintf("%s:%s", gen.GetEntityType(), gen.GetSubType())
		if seen[key] {
			warnings = append(warnings, fmt.Sprintf(
				"Duplicate generator found: %s/%s",
				gen.GetEntityType(), gen.GetSubType()))
		}
		seen[key] = true
	}

	return warnings
}
