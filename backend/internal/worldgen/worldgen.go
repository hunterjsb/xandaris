package worldgen

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// WorldType represents different planetary world types
type WorldType string

const (
	Abundant  WorldType = "Abundant"
	Fertile   WorldType = "Fertile"
	Mountain  WorldType = "Mountain"
	Desert    WorldType = "Desert"
	Volcanic  WorldType = "Volcanic"
	Highlands WorldType = "Highlands"
	Swamp     WorldType = "Swamp"
	Barren    WorldType = "Barren"
	Radiant   WorldType = "Radiant"
	Barred    WorldType = "Barred"
	Null      WorldType = "Null"
)

// WorldTypeRange defines the seed range for each world type
type WorldTypeRange struct {
	Start int
	End   int
}

// WorldTypeMap maps seed prefixes to world types
var WorldTypeMap = map[WorldType]WorldTypeRange{
	Abundant:  {0, 12},
	Fertile:   {13, 62},
	Mountain:  {63, 112},
	Desert:    {113, 137},
	Volcanic:  {138, 162},
	Highlands: {163, 200},
	Swamp:     {201, 238},
	Barren:    {239, 243},
	Radiant:   {244, 249},
	Barred:    {250, 250},
	Null:      {251, 999},
}

// ResourceDistribution represents probability distributions for each world type
// Each array represents the cumulative distribution for 8 different resource types
var ResourceDistributions = map[WorldType][][]float64{
	Abundant: {
		{0.05, 0.15, 0.30, 0.50, 0.70, 0.85, 0.95, 1.00}, // Resource 1 (Common)
		{0.10, 0.25, 0.45, 0.65, 0.80, 0.90, 0.97, 1.00}, // Resource 2
		{0.15, 0.35, 0.55, 0.70, 0.82, 0.91, 0.97, 1.00}, // Resource 3
		{0.08, 0.20, 0.40, 0.60, 0.75, 0.87, 0.95, 1.00}, // Resource 4
		{0.12, 0.28, 0.48, 0.68, 0.82, 0.92, 0.98, 1.00}, // Resource 5
		{0.18, 0.38, 0.58, 0.72, 0.84, 0.93, 0.98, 1.00}, // Resource 6
		{0.20, 0.40, 0.60, 0.75, 0.86, 0.94, 0.98, 1.00}, // Resource 7
		{0.25, 0.45, 0.65, 0.78, 0.88, 0.95, 0.99, 1.00}, // Resource 8
	},
	Fertile: {
		{0.02, 0.08, 0.20, 0.40, 0.65, 0.82, 0.93, 1.00}, // Emphasis on biological resources
		{0.01, 0.05, 0.15, 0.35, 0.60, 0.80, 0.92, 1.00},
		{0.15, 0.35, 0.55, 0.70, 0.82, 0.91, 0.97, 1.00},
		{0.20, 0.40, 0.60, 0.75, 0.85, 0.92, 0.97, 1.00},
		{0.25, 0.45, 0.65, 0.78, 0.87, 0.93, 0.97, 1.00},
		{0.30, 0.50, 0.68, 0.80, 0.88, 0.94, 0.98, 1.00},
		{0.35, 0.55, 0.70, 0.82, 0.90, 0.95, 0.98, 1.00},
		{0.40, 0.60, 0.75, 0.85, 0.92, 0.96, 0.99, 1.00},
	},
	Mountain: {
		{0.40, 0.60, 0.75, 0.85, 0.92, 0.96, 0.99, 1.00}, // Rich in minerals
		{0.35, 0.55, 0.70, 0.82, 0.90, 0.95, 0.98, 1.00},
		{0.05, 0.15, 0.30, 0.50, 0.70, 0.85, 0.95, 1.00},
		{0.10, 0.25, 0.45, 0.65, 0.80, 0.90, 0.97, 1.00},
		{0.30, 0.50, 0.68, 0.80, 0.88, 0.94, 0.98, 1.00},
		{0.25, 0.45, 0.65, 0.78, 0.87, 0.93, 0.97, 1.00},
		{0.20, 0.40, 0.60, 0.75, 0.85, 0.92, 0.97, 1.00},
		{0.15, 0.35, 0.55, 0.70, 0.82, 0.91, 0.97, 1.00},
	},
	Desert: {
		{0.50, 0.70, 0.82, 0.90, 0.95, 0.98, 0.99, 1.00}, // Rare resources concentrated
		{0.60, 0.75, 0.85, 0.92, 0.96, 0.98, 0.99, 1.00},
		{0.70, 0.82, 0.90, 0.95, 0.97, 0.99, 1.00, 1.00},
		{0.65, 0.78, 0.87, 0.93, 0.97, 0.99, 1.00, 1.00},
		{0.55, 0.72, 0.84, 0.91, 0.96, 0.98, 0.99, 1.00},
		{0.45, 0.65, 0.78, 0.88, 0.94, 0.97, 0.99, 1.00},
		{0.40, 0.60, 0.75, 0.85, 0.92, 0.96, 0.99, 1.00},
		{0.35, 0.55, 0.70, 0.82, 0.90, 0.95, 0.98, 1.00},
	},
	Volcanic: {
		{0.30, 0.50, 0.68, 0.80, 0.88, 0.94, 0.98, 1.00}, // Energy and rare minerals
		{0.45, 0.65, 0.78, 0.88, 0.94, 0.97, 0.99, 1.00},
		{0.40, 0.60, 0.75, 0.85, 0.92, 0.96, 0.99, 1.00},
		{0.25, 0.45, 0.65, 0.78, 0.87, 0.93, 0.97, 1.00},
		{0.20, 0.40, 0.60, 0.75, 0.85, 0.92, 0.97, 1.00},
		{0.35, 0.55, 0.70, 0.82, 0.90, 0.95, 0.98, 1.00},
		{0.50, 0.70, 0.82, 0.90, 0.95, 0.98, 0.99, 1.00},
		{0.55, 0.72, 0.84, 0.91, 0.96, 0.98, 0.99, 1.00},
	},
	Highlands: {
		{0.20, 0.40, 0.60, 0.75, 0.85, 0.92, 0.97, 1.00}, // Balanced distribution
		{0.25, 0.45, 0.65, 0.78, 0.87, 0.93, 0.97, 1.00},
		{0.30, 0.50, 0.68, 0.80, 0.88, 0.94, 0.98, 1.00},
		{0.18, 0.38, 0.58, 0.72, 0.84, 0.91, 0.96, 1.00},
		{0.22, 0.42, 0.62, 0.76, 0.86, 0.92, 0.97, 1.00},
		{0.28, 0.48, 0.66, 0.79, 0.87, 0.93, 0.97, 1.00},
		{0.32, 0.52, 0.69, 0.81, 0.89, 0.94, 0.98, 1.00},
		{0.26, 0.46, 0.64, 0.77, 0.86, 0.92, 0.97, 1.00},
	},
	Swamp: {
		{0.10, 0.25, 0.45, 0.65, 0.80, 0.90, 0.97, 1.00}, // Organic and water resources
		{0.05, 0.15, 0.35, 0.58, 0.75, 0.87, 0.95, 1.00},
		{0.15, 0.35, 0.55, 0.70, 0.82, 0.91, 0.97, 1.00},
		{0.08, 0.20, 0.40, 0.60, 0.75, 0.87, 0.95, 1.00},
		{0.12, 0.28, 0.48, 0.68, 0.82, 0.92, 0.98, 1.00},
		{0.25, 0.45, 0.65, 0.78, 0.87, 0.93, 0.97, 1.00},
		{0.35, 0.55, 0.70, 0.82, 0.90, 0.95, 0.98, 1.00},
		{0.30, 0.50, 0.68, 0.80, 0.88, 0.94, 0.98, 1.00},
	},
	Barren: {
		{0.80, 0.90, 0.95, 0.97, 0.99, 1.00, 1.00, 1.00}, // Very sparse resources
		{0.85, 0.92, 0.96, 0.98, 0.99, 1.00, 1.00, 1.00},
		{0.90, 0.95, 0.97, 0.99, 1.00, 1.00, 1.00, 1.00},
		{0.88, 0.94, 0.97, 0.99, 1.00, 1.00, 1.00, 1.00},
		{0.82, 0.91, 0.96, 0.98, 0.99, 1.00, 1.00, 1.00},
		{0.75, 0.87, 0.93, 0.97, 0.99, 1.00, 1.00, 1.00},
		{0.78, 0.88, 0.94, 0.97, 0.99, 1.00, 1.00, 1.00},
		{0.83, 0.92, 0.96, 0.98, 0.99, 1.00, 1.00, 1.00},
	},
	Radiant: {
		{0.25, 0.45, 0.65, 0.78, 0.87, 0.93, 0.97, 1.00}, // Energy-rich
		{0.15, 0.35, 0.55, 0.70, 0.82, 0.91, 0.97, 1.00},
		{0.35, 0.55, 0.70, 0.82, 0.90, 0.95, 0.98, 1.00},
		{0.45, 0.65, 0.78, 0.88, 0.94, 0.97, 0.99, 1.00},
		{0.40, 0.60, 0.75, 0.85, 0.92, 0.96, 0.99, 1.00},
		{0.30, 0.50, 0.68, 0.80, 0.88, 0.94, 0.98, 1.00},
		{0.20, 0.40, 0.60, 0.75, 0.85, 0.92, 0.97, 1.00},
		{0.10, 0.25, 0.45, 0.65, 0.80, 0.90, 0.97, 1.00},
	},
	Barred: {
		{0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 1.00}, // Special case: only one resource type
		{0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 1.00},
		{0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 1.00},
		{0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 1.00},
		{0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 1.00},
		{0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 1.00},
		{0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 0.12, 1.00},
		{0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 1.00},
	},
	Null: {
		{1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00}, // No resources
		{1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00},
		{1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00},
		{1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00},
		{1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00},
		{1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00},
		{1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00},
		{1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00, 1.00},
	},
}

// GetWorldTypeFromSeed determines world type from a 20-digit seed
func GetWorldTypeFromSeed(seed int64) WorldType {
	seedStr := fmt.Sprintf("%020d", seed)
	firstThreeDigits, _ := strconv.Atoi(seedStr[:3])
	
	for worldType, r := range WorldTypeMap {
		if firstThreeDigits >= r.Start && firstThreeDigits <= r.End {
			return worldType
		}
	}
	return Null
}

// GetSuccessFromRoll returns the number of successes for a given value using CDF lookup
func GetSuccessFromRoll(distribution []float64, value float64) int {
	normalizedValue := value / 10000000000.0 // Normalize 10-digit number to 0-1 range
	
	for i, threshold := range distribution {
		if normalizedValue <= threshold {
			return i
		}
	}
	return len(distribution) - 1
}

// Resolve20DigitSeed processes a 20-digit seed and returns 8 success values
func Resolve20DigitSeed(worldType WorldType, seed int64) []int {
	if worldType == Null {
		return []int{0, 0, 0, 0, 0, 0, 0, 0}
	}
	if worldType == Barred {
		return []int{0, 0, 0, 0, 0, 0, 0, 1}
	}
	
	distributions := ResourceDistributions[worldType]
	seedStr := fmt.Sprintf("%020d", seed)
	results := make([]int, 8)
	
	// Use digits 4-20 (17 digits), take 8 slices of 10 digits each
	for i := 0; i < 8; i++ {
		sliceStr := seedStr[3+i : 3+i+10]
		value, _ := strconv.ParseFloat(sliceStr, 64)
		results[i] = GetSuccessFromRoll(distributions[i], value)
	}
	
	return results
}

// ProcessSeed processes a single 20-digit seed
func ProcessSeed(seed int64) (WorldType, []int) {
	worldType := GetWorldTypeFromSeed(seed)
	results := Resolve20DigitSeed(worldType, seed)
	return worldType, results
}

// Process60DigitSeed processes a 60-digit seed to create 5 worlds
func Process60DigitSeed(seed int64) []WorldGenResult {
	rand.Seed(time.Now().UnixNano())
	
	// Generate a 60-digit seed if the input is too small
	fullSeed := seed
	if seed < 1000000000000000000 { // Less than 18 digits
		fullSeed = rand.Int63n(999999999999999999) + 1000000000000000000
	}
	
	seedStr := fmt.Sprintf("%060d", fullSeed)
	
	// Extract 5 overlapping 20-digit subseeds
	subseeds := []string{
		seedStr[0:20],
		seedStr[10:30],
		seedStr[20:40],
		seedStr[30:50],
		seedStr[40:60],
	}
	
	results := make([]WorldGenResult, 5)
	for i, subseedStr := range subseeds {
		subseedInt, _ := strconv.ParseInt(subseedStr, 10, 64)
		worldType, successes := ProcessSeed(subseedInt)
		results[i] = WorldGenResult{
			Subseed:   subseedInt,
			WorldType: worldType,
			Successes: successes,
		}
	}
	
	return results
}

// WorldGenResult represents the result of world generation for a single subseed
type WorldGenResult struct {
	Subseed   int64
	WorldType WorldType
	Successes []int
}

// GenerateResourceNodesForPlanet creates resource nodes for a planet using world generation
func GenerateResourceNodesForPlanet(app *pocketbase.PocketBase, planet *models.Record) error {
	// Get planet type from database to determine world generation type
	planetTypeID := planet.GetString("planet_type")
	if planetTypeID == "" {
		return fmt.Errorf("planet %s has no planet_type", planet.Id)
	}
	
	planetType, err := app.Dao().FindRecordById("planet_types", planetTypeID)
	if err != nil {
		return fmt.Errorf("failed to fetch planet type %s: %w", planetTypeID, err)
	}
	
	planetTypeName := planetType.GetString("name")
	worldType := WorldType(planetTypeName)
	
	// Generate a unique seed for this planet based on its ID and position
	planetID := planet.Id
	systemID := planet.GetString("system_id")
	seed := GenerateSeedFromIDs(planetID, systemID)
	
	// Process the seed to get world generation results
	_, successes := ProcessSeed(seed)
	
	// Get all resource types from database
	resourceTypes, err := app.Dao().FindRecordsByExpr("resource_types", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch resource types: %w", err)
	}
	
	if len(resourceTypes) == 0 {
		return fmt.Errorf("no resource types found in database")
	}
	
	resourceNodeCollection, err := app.Dao().FindCollectionByNameOrId("resource_nodes")
	if err != nil {
		return fmt.Errorf("resource_nodes collection not found: %w", err)
	}
	
	// Create resource nodes based on the success values
	for i, successCount := range successes {
		if successCount > 0 && i < len(resourceTypes) {
			resourceType := resourceTypes[i]
			
			// Create multiple nodes for higher success counts
			nodeCount := successCount
			if nodeCount > 3 { // Cap at 3 nodes per resource type
				nodeCount = 3
			}
			
			for j := 0; j < nodeCount; j++ {
				record := models.NewRecord(resourceNodeCollection)
				record.Set("planet_id", planet.Id)
				record.Set("resource_type", resourceType.Id)
				
				// Richness based on success count and world type
				richness := calculateRichness(worldType, successCount, j)
				record.Set("richness", richness)
				record.Set("exhausted", false)
				
				if err := app.Dao().SaveRecord(record); err != nil {
					return fmt.Errorf("failed to save resource node: %w", err)
				}
			}
		}
	}
	
	return nil
}

// GenerateSeedFromIDs creates a deterministic seed from planet and system IDs
func GenerateSeedFromIDs(planetID, systemID string) int64 {
	// Create a more varied hash from the combined IDs
	combined := planetID + systemID
	var hash1, hash2, hash3 int64
	
	// Multiple hash passes for better distribution
	for i, char := range combined {
		hash1 = hash1*31 + int64(char)
		hash2 = hash2*37 + int64(char)*int64(i+1)
		hash3 = hash3*41 + int64(char)*int64(char)
	}
	
	// Combine the hashes
	seed := hash1 ^ hash2 ^ hash3
	
	// Ensure it's positive
	if seed < 0 {
		seed = -seed
	}
	
	// Create a varied 20-digit number by mixing the hash in different positions
	// Use modulo arithmetic to spread across the full range
	part1 := (seed % 99999) + 10000           // 5 digits: 10000-99999
	part2 := ((seed >> 16) % 99999) + 10000   // 5 digits: 10000-99999  
	part3 := ((seed >> 32) % 99999) + 10000   // 5 digits: 10000-99999
	part4 := ((seed >> 48) % 99999) + 10000   // 5 digits: 10000-99999
	
	// Combine into a 20-digit number
	result := part1*1000000000000000 + part2*100000000000 + part3*1000000 + part4
	
	return result
}

// calculateRichness determines resource richness based on world type and success
func calculateRichness(worldType WorldType, successCount, nodeIndex int) int {
	baseRichness := successCount + 1
	
	// Apply world type modifiers
	switch worldType {
	case Abundant:
		baseRichness += 2
	case Fertile:
		if nodeIndex < 2 { // First two resource types get bonus
			baseRichness += 3
		}
	case Mountain:
		if nodeIndex < 3 { // Mineral resources get bonus
			baseRichness += 4
		}
	case Desert:
		if successCount >= 5 { // Rare but rich deposits
			baseRichness += 3
		}
	case Volcanic:
		if nodeIndex >= 5 { // Energy resources get bonus
			baseRichness += 3
		}
	case Radiant:
		baseRichness += 1 // Moderate bonus across all resources
	case Barren:
		baseRichness = 1 // Always poor
	case Null:
		return 0 // No resources
	}
	
	// Diminishing returns for multiple nodes of same type
	baseRichness -= nodeIndex
	
	// Clamp between 1-10
	if baseRichness < 1 {
		baseRichness = 1
	}
	if baseRichness > 10 {
		baseRichness = 10
	}
	
	return baseRichness
}

// GenerateResourceNodesForAllPlanets regenerates resource nodes for all planets
func GenerateResourceNodesForAllPlanets(app *pocketbase.PocketBase) error {
	// Clear existing resource nodes
	existingNodes, err := app.Dao().FindRecordsByExpr("resource_nodes", nil, nil)
	if err == nil {
		for _, node := range existingNodes {
			_ = app.Dao().DeleteRecord(node)
		}
	}
	
	// Get all planets
	planets, err := app.Dao().FindRecordsByExpr("planets", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch planets: %w", err)
	}
	
	fmt.Printf("Generating enhanced resource nodes for %d planets using world generation system...\n", len(planets))
	
	for i, planet := range planets {
		if err := GenerateResourceNodesForPlanet(app, planet); err != nil {
			fmt.Printf("Warning: failed to generate resource nodes for planet %s: %v\n", planet.Id, err)
			continue // Skip this planet but continue with others
		}
		
		if (i+1)%50 == 0 {
			fmt.Printf("Processed %d/%d planets\n", i+1, len(planets))
		}
	}
	
	fmt.Printf("Successfully generated enhanced resource nodes for %d planets\n", len(planets))
	return nil
}

// SetPlanetTypeBasedOnSeed assigns a planet type based on world generation seed
func SetPlanetTypeBasedOnSeed(app *pocketbase.PocketBase, planet *models.Record) error {
	// Generate a unique seed for this planet
	planetID := planet.Id
	systemID := planet.GetString("system_id")
	seed := GenerateSeedFromIDs(planetID, systemID)
	
	// Get world type from seed
	worldType := GetWorldTypeFromSeed(seed)
	
	// Find corresponding planet type in database
	planetTypes, err := app.Dao().FindRecordsByFilter("planet_types", "name = '"+string(worldType)+"'", "", 1, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch planet types: %w", err)
	}
	
	if len(planetTypes) == 0 {
		// Fallback to a default planet type if worldgen type not found
		allPlanetTypes, err := app.Dao().FindRecordsByExpr("planet_types", nil, nil)
		if err != nil || len(allPlanetTypes) == 0 {
			return fmt.Errorf("no planet types found in database")
		}
		// Use first available planet type as fallback
		planet.Set("planet_type", allPlanetTypes[0].Id)
	} else {
		planet.Set("planet_type", planetTypes[0].Id)
	}
	
	return app.Dao().SaveRecord(planet)
}