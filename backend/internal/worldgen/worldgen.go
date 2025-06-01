package worldgen

import (
	"crypto/md5"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// SystemGenResult represents the result of system generation
type SystemGenResult struct {
	SystemSeed    string   `json:"system_seed"`
	SystemName    string   `json:"system_name"`
	SystemPlanets []Planet `json:"system_planets"`
}

// Planet represents a generated planet
type Planet struct {
	PlanetName      string `json:"planet_name"`
	PlanetType      string `json:"planet_type"`
	PlanetResources []int  `json:"planet_resources"`
}

// WorldType represents different planetary world types
type WorldType struct {
	Name        string
	Probability float64
	FoodCap     float64
	OreCap      float64
	OilCap      float64
	TitaniumCap float64
	XaniumCap   float64
	IsSpecial   bool
}

// World type definitions based on the provided table
var WorldTypes = []WorldType{
	{"Abundant", 0.0125, 0.25, 0.25, 0.2, 0.1, 0, false},
	{"Fertile", 0.05, 0.8, 0.2, 0, 0, 0, false},
	{"Mountain", 0.05, 0, 0.5, 0.2, 0.1, 0, false},
	{"Desert", 0.025, 0, 0, 0.5, 0.15, 0, false},
	{"Volcanic", 0.025, 0, 0.5, 0.4, 0, 0, false},
	{"Highlands", 0.0375, 0.5, 0.5, 0, 0, 0, false},
	{"Swamp", 0.0375, 0.5, 0, 0.2, 0.1, 0, false},
	{"Barren", 0.005, 0, 0.1, 0.12, 0.2, 0.025, false},
	{"Radiant", 0.00625, 0, 0, 0, 0.4, 0, false},
	{"Barred", 0.00125, 0, 0, 0, 0, 0, true}, // Special: exactly 2 Xanium deposits
	{"Null", 0.75, 0, 0, 0, 0, 0, true},      // Special: no resources, won't display
}

// Name generation themes
var nameThemes = map[string][]string{
	"Abundant":  {"Terra", "Verde", "Bounty", "Rich", "Prime", "Golden", "Lush", "Fertile"},
	"Fertile":   {"Bloom", "Garden", "Harvest", "Grove", "Field", "Eden", "Flora", "Gaia"},
	"Mountain":  {"Peak", "Ridge", "Stone", "Crag", "Summit", "Boulder", "Cliff", "Mesa"},
	"Desert":    {"Dune", "Sand", "Arid", "Scorch", "Dust", "Waste", "Burn", "Dry"},
	"Volcanic":  {"Forge", "Flame", "Magma", "Ember", "Scoria", "Cinder", "Lava", "Igneous"},
	"Highlands": {"High", "Plateau", "Uplift", "Moor", "Height", "Table", "Elevated", "Rise"},
	"Swamp":     {"Bog", "Marsh", "Fen", "Mire", "Wetland", "Delta", "Bayou", "Moss"},
	"Barren":    {"Void", "Empty", "Dead", "Waste", "Hollow", "Desolate", "Stark", "Bleak"},
	"Radiant":   {"Bright", "Shine", "Glow", "Beam", "Solar", "Stellar", "Luminous", "Radiant"},
	"Barred":    {"Locked", "Sealed", "Forbidden", "Quarantine", "Restricted", "Blocked", "Denied", "Barred"},
	"Null":      {"None", "Void", "Empty", "Null", "Zero", "Absent", "Missing", "Gone"},
}

var systemPrefixes = []string{
	"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta",
	"Iota", "Kappa", "Lambda", "Mu", "Nu", "Xi", "Omicron", "Pi",
	"Rho", "Sigma", "Tau", "Upsilon", "Phi", "Chi", "Psi", "Omega",
	"Proxima", "Centauri", "Vega", "Altair", "Rigel", "Betelgeuse", "Arcturus",
}

var constellations = []string{
	"Andromedae", "Aquarii", "Aquilae", "Arietis", "Aurigae", "Bootis",
	"Cancri", "Canis", "Capricorni", "Carinae", "Cassiopeiae", "Centauri",
	"Cephei", "Ceti", "Columbae", "Coronae", "Corvi", "Crateris",
	"Cygni", "Delphini", "Draconis", "Equulei", "Eridani", "Fornacis",
	"Geminorum", "Gruis", "Herculis", "Hydrae", "Leonis", "Librae",
	"Lupi", "Lyrae", "Orionis", "Pegasi", "Persei", "Piscium",
	"Sagittarii", "Scorpii", "Serpentis", "Tauri", "Ursae", "Virginis",
}

// ProcessSystemSeed processes a 32-digit seed to create up to 5 planets (API version - filters Null)
func ProcessSystemSeed(seed32 *big.Int) SystemGenResult {
	// Use seed to initialize random generator
	rng := rand.New(rand.NewSource(seed32.Int64()))
	
	systemName := GenerateSystemName(rng)
	var planets []Planet
	
	// Generate 1-5 planets based on seed
	numPlanets := rng.Intn(5) + 1
	
	for i := 0; i < numPlanets; i++ {
		worldType := selectWorldType(rng)
		
		// Skip Null worlds (they won't display to user)
		if worldType.Name == "Null" {
			continue
		}
		
		planet := Planet{
			PlanetName:      GeneratePlanetName(worldType.Name, rng),
			PlanetType:      worldType.Name,
			PlanetResources: generateResources(worldType, rng),
		}
		planets = append(planets, planet)
	}
	
	// If no planets generated, create at least one non-Null planet
	if len(planets) == 0 {
		worldType := WorldTypes[0] // Abundant
		planets = append(planets, Planet{
			PlanetName:      GeneratePlanetName(worldType.Name, rng),
			PlanetType:      worldType.Name,
			PlanetResources: generateResources(worldType, rng),
		})
	}
	
	return SystemGenResult{
		SystemSeed:    encodeSeedBase64(seed32),
		SystemName:    systemName,
		SystemPlanets: planets,
	}
}

// ProcessSystemSeedForDatabase processes a 32-digit seed including Null systems (Database version)
func ProcessSystemSeedForDatabase(seed32 *big.Int) SystemGenResult {
	// Use seed to initialize random generator
	rng := rand.New(rand.NewSource(seed32.Int64()))
	
	systemName := GenerateSystemName(rng)
	var planets []Planet
	
	// Generate 1-5 planets and roll for each planet individually
	numPlanets := rng.Intn(5) + 1
	
	for i := 0; i < numPlanets; i++ {
		worldType := selectWorldType(rng)
		
		// Include all world types for database (including Null)
		planet := Planet{
			PlanetName:      GeneratePlanetName(worldType.Name, rng),
			PlanetType:      worldType.Name,
			PlanetResources: generateResources(worldType, rng),
		}
		planets = append(planets, planet)
	}
	
	return SystemGenResult{
		SystemSeed:    encodeSeedBase64(seed32),
		SystemName:    systemName,
		SystemPlanets: planets,
	}
}

// selectWorldType selects a world type based on probability distribution
func selectWorldType(rng *rand.Rand) WorldType {
	roll := rng.Float64()
	cumulative := 0.0

	for _, wt := range WorldTypes {
		cumulative += wt.Probability
		if roll <= cumulative {
			return wt
		}
	}

	// Fallback to Null
	return WorldTypes[len(WorldTypes)-1]
}

// generateResources generates resource counts based on world type capabilities
func generateResources(worldType WorldType, rng *rand.Rand) []int {
	if worldType.Name == "Barred" {
		// Special case: exactly 2 Xanium deposits and nothing else
		return []int{0, 0, 0, 0, 2}
	}
	
	if worldType.Name == "Null" {
		// No resources
		return []int{0, 0, 0, 0, 0}
	}
	
	resources := make([]int, 5)
	
	// Food (cap 8) - FoodCap is probability of getting resources
	if worldType.FoodCap > 0 && rng.Float64() < worldType.FoodCap {
		resources[0] = rng.Intn(8) + 1 // 1-8 resources
	}
	
	// Ore (cap 8) - OreCap is probability of getting resources
	if worldType.OreCap > 0 && rng.Float64() < worldType.OreCap {
		resources[1] = rng.Intn(8) + 1 // 1-8 resources
	}
	
	// Oil (cap 5) - OilCap is probability of getting resources
	if worldType.OilCap > 0 && rng.Float64() < worldType.OilCap {
		resources[2] = rng.Intn(5) + 1 // 1-5 resources
	}
	
	// Titanium (cap 2) - TitaniumCap is probability of getting resources
	if worldType.TitaniumCap > 0 && rng.Float64() < worldType.TitaniumCap {
		resources[3] = rng.Intn(2) + 1 // 1-2 resources
	}
	
	// Xanium (special) - XaniumCap is probability of getting 1 resource
	if worldType.XaniumCap > 0 && rng.Float64() < worldType.XaniumCap {
		resources[4] = 1
	}
	
	return resources
}

// GenerateSystemName generates a system name
func GenerateSystemName(rng *rand.Rand) string {
	prefix := systemPrefixes[rng.Intn(len(systemPrefixes))]

	if rng.Float32() < 0.5 {
		// Format: "Alpha 42 Centauri"
		number := rng.Intn(100)
		constellation := constellations[rng.Intn(len(constellations))]
		return fmt.Sprintf("%s %02d %s", prefix, number, constellation)
	} else {
		// Format: "Alpha-123"
		number := rng.Intn(900) + 100
		return fmt.Sprintf("%s-%d", prefix, number)
	}
}

// GeneratePlanetName generates a planet name based on world type
func GeneratePlanetName(worldType string, rng *rand.Rand) string {
	themes, exists := nameThemes[worldType]
	if !exists {
		return "Unknown"
	}

	base := themes[rng.Intn(len(themes))]

	// Add suffix for variety
	suffixes := []string{"Prime", "Major", "Minor", "Alpha", "Beta", "Gamma", "I", "II", "III", "IV", "V"}
	if rng.Float32() < 0.3 {
		suffix := suffixes[rng.Intn(len(suffixes))]
		return fmt.Sprintf("%s %s", base, suffix)
	}

	return base
}

// encodeSeedBase64 converts seed to base64 string
func encodeSeedBase64(seed *big.Int) string {
	seedBytes := seed.Bytes()
	if len(seedBytes) < 15 {
		padded := make([]byte, 15)
		copy(padded[15-len(seedBytes):], seedBytes)
		seedBytes = padded
	}
	return fmt.Sprintf("%x", seedBytes)[:20] // Truncate for display
}

// GenerateRandomSystemSeed generates a random 32-digit seed
func GenerateRandomSystemSeed() *big.Int {
	seed32 := big.NewInt(0)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate random 32-digit number
	for i := 0; i < 32; i++ {
		digit := big.NewInt(int64(rng.Intn(10)))
		seed32.Mul(seed32, big.NewInt(10))
		seed32.Add(seed32, digit)
	}
	return seed32
}

// GenerateResourceNodesForSystem creates resource nodes for all planets in a system using worldgen
func GenerateResourceNodesForSystem(app *pocketbase.PocketBase, systemID string) error {
	// Get all planets in this system
	planets, err := app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("system_id='%s'", systemID), "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch planets for system %s: %w", systemID, err)
	}
	
	if len(planets) == 0 {
		return nil
	}
	
	// Generate a system seed based on system ID
	seed := GenerateSeedFromSystemID(systemID)
	seed32 := big.NewInt(seed)
	
	// Process the system seed to get planets with types and resources (use database version)
	system := ProcessSystemSeedForDatabase(seed32)
	
	// If system generates no planets (Null system), set all planets to Null type
	if len(system.SystemPlanets) == 0 {
		return setAllPlanetsToNull(app, planets)
	}
	
	// Map generated planets to database planets
	for i, planet := range planets {
		var planetData Planet
		
		if i < len(system.SystemPlanets) {
			// Use generated planet data
			planetData = system.SystemPlanets[i]
		} else {
			// Extra planets become Null
			planetData = Planet{
				PlanetType:      "Null",
				PlanetResources: []int{0, 0, 0, 0, 0},
			}
		}
		
		// Set planet type in database
		if err := setPlanetType(app, planet, planetData.PlanetType); err != nil {
			return err
		}
		
		// Create resource nodes if not Null
		if planetData.PlanetType != "Null" {
			if err := createResourceNodesForPlanet(app, planet, planetData.PlanetResources); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// GenerateResourceNodesForPlanet is kept for backward compatibility but now calls system-level generation
func GenerateResourceNodesForPlanet(app *pocketbase.PocketBase, planet *models.Record) error {
	systemID := planet.GetString("system_id")
	return GenerateResourceNodesForSystem(app, systemID)
}

// GenerateSeedFromIDs creates a deterministic seed from planet and system IDs
func GenerateSeedFromIDs(planetID, systemID string) int64 {
	combined := planetID + systemID
	hash := md5.Sum([]byte(combined))
	
	// Convert hash to int64
	var result int64
	for i := 0; i < 8 && i < len(hash); i++ {
		result = (result << 8) | int64(hash[i])
	}
	
	if result < 0 {
		result = -result
	}
	
	return result
}

// GenerateSeedFromSystemID creates a deterministic seed from system ID
func GenerateSeedFromSystemID(systemID string) int64 {
	hash := md5.Sum([]byte(systemID))
	
	// Convert hash to int64
	var result int64
	for i := 0; i < 8 && i < len(hash); i++ {
		result = (result << 8) | int64(hash[i])
	}
	
	if result < 0 {
		result = -result
	}
	
	return result
}

// Helper functions for system-level generation
func setAllPlanetsToNull(app *pocketbase.PocketBase, planets []*models.Record) error {
	for _, planet := range planets {
		if err := setPlanetType(app, planet, "Null"); err != nil {
			return err
		}
	}
	return nil
}

func setPlanetType(app *pocketbase.PocketBase, planet *models.Record, typeName string) error {
	planetTypes, err := app.Dao().FindRecordsByFilter("planet_types", fmt.Sprintf("name='%s'", typeName), "", 1, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch planet types: %w", err)
	}
	
	if len(planetTypes) > 0 {
		planet.Set("planet_type", planetTypes[0].Id)
		if err := app.Dao().SaveRecord(planet); err != nil {
			return fmt.Errorf("failed to update planet type: %w", err)
		}
	}
	
	return nil
}

func createResourceNodesForPlanet(app *pocketbase.PocketBase, planet *models.Record, resources []int) error {
	// Map our 5 resource types to the correct database resource types
	resourceMapping := map[int]string{
		0: "food",     // Food
		1: "ore",      // Ore  
		2: "oil",      // Oil
		3: "titanium", // Titanium
		4: "xanium",   // Xanium
	}
	
	resourceNodeCollection, err := app.Dao().FindCollectionByNameOrId("resource_nodes")
	if err != nil {
		return fmt.Errorf("resource_nodes collection not found: %w", err)
	}
	
	// Create resource nodes based on the generated resources
	for i, count := range resources {
		if count > 0 {
			resourceName, exists := resourceMapping[i]
			if !exists {
				continue
			}
			
			// Find the resource type by name
			resourceTypes, err := app.Dao().FindRecordsByFilter("resource_types", fmt.Sprintf("name='%s'", resourceName), "", 1, 0)
			if err != nil || len(resourceTypes) == 0 {
				continue
			}
			
			resourceType := resourceTypes[0]
			
			// Create nodes based on count (limit to 3 nodes per resource type)
			nodeCount := count
			if nodeCount > 3 {
				nodeCount = 3
			}
			
			for j := 0; j < nodeCount; j++ {
				record := models.NewRecord(resourceNodeCollection)
				record.Set("planet_id", planet.Id)
				record.Set("resource_type", resourceType.Id)
				record.Set("richness", count)
				record.Set("exhausted", false)
				
				if err := app.Dao().SaveRecord(record); err != nil {
					return fmt.Errorf("failed to save resource node: %w", err)
				}
			}
		}
	}
	
	return nil
}

// GenerateResourceNodesForAllPlanets regenerates resource nodes for all planets by processing systems
func GenerateResourceNodesForAllPlanets(app *pocketbase.PocketBase) error {
	// Clear existing resource nodes
	existingNodes, err := app.Dao().FindRecordsByExpr("resource_nodes", nil, nil)
	if err == nil {
		for _, node := range existingNodes {
			_ = app.Dao().DeleteRecord(node)
		}
	}
	
	// Get all systems
	systems, err := app.Dao().FindRecordsByExpr("systems", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch systems: %w", err)
	}
	
	fmt.Printf("Generating resource nodes for %d systems using table-based worldgen...\n", len(systems))
	
	for i, system := range systems {
		if err := GenerateResourceNodesForSystem(app, system.Id); err != nil {
			fmt.Printf("Warning: failed to generate resource nodes for system %s: %v\n", system.Id, err)
			continue
		}
		
		if (i+1)%50 == 0 {
			fmt.Printf("Processed %d/%d systems\n", i+1, len(systems))
		}
	}
	
	fmt.Printf("Successfully generated resource nodes for %d systems\n", len(systems))
	return nil
}
