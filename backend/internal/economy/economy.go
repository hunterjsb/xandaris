package economy

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// UpdateMarkets handles the economic simulation for all colonized planets
func UpdateMarkets(app *pocketbase.PocketBase) error {
	log.Println("Updating markets and economy...")

	// Process bank income first (if banks still exist)
	if err := ProcessBankIncome(app); err != nil {
		log.Printf("Error processing bank income: %v", err)
		// Don't fail the entire economy update for bank errors
	}

	// Get all colonized planets (this is where the economy happens now)
	planets, err := app.Dao().FindRecordsByFilter("planets", "colonized_by != ''", "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch colonized planets: %w", err)
	}

	planetsProcessed := 0
	for _, planet := range planets {
		// Skip planets without populations (empty colonies)
		populations, err := app.Dao().FindRecordsByFilter("populations", 
			fmt.Sprintf("planet_id = '%s'", planet.Id), "", 1, 0)
		if err != nil || len(populations) == 0 {
			continue // Skip planets with no population
		}

		if err := updatePlanetEconomy(app, planet); err != nil {
			log.Printf("Failed to update economy for planet %s: %v", planet.Id, err)
			continue
		}

		if err := app.Dao().SaveRecord(planet); err != nil {
			log.Printf("Failed to save planet %s: %v", planet.Id, err)
		}
		planetsProcessed++
	}

	log.Printf("Updated economy for %d planets with populations (skipped %d empty colonies)", planetsProcessed, len(planets)-planetsProcessed)
	return nil
}

// updatePlanetEconomy simulates production and consumption for a single planet
func updatePlanetEconomy(app *pocketbase.PocketBase, planet *models.Record) error {
	// Get planet owner
	ownerID := planet.GetString("colonized_by")
	if ownerID == "" {
		return nil // Skip uncolonized planets
	}

	// Get planet size (affects base capacity)
	planetSize := planet.GetInt("size")
	if planetSize == 0 {
		planetSize = 1
	}

	// Get planet type for habitability modifier
	planetType, err := app.Dao().FindRecordById("planet_types", planet.GetString("planet_type"))
	if err != nil {
		log.Printf("Failed to get planet type for planet %s: %v", planet.Id, err)
		return nil
	}

	habitability := planetType.GetFloat("habitability")
	maxPop := int(float64(planetType.GetInt("base_max_population")) * float64(planetSize) * habitability)

	// Get population on this planet
	populations, err := app.Dao().FindRecordsByFilter("populations", 
		fmt.Sprintf("owner_id = '%s' && planet_id = '%s'", ownerID, planet.Id), "", 0, 0)
	if err != nil {
		log.Printf("Failed to get population for planet %s: %v", planet.Id, err)
		return nil
	}

	totalPop := 0
	totalHappiness := 75 // Default happiness
	for _, pop := range populations {
		totalPop += pop.GetInt("count")
		if pop.GetInt("happiness") > 0 {
			totalHappiness = pop.GetInt("happiness")
		}
	}

	// Get buildings on this planet first (some buildings like crypto_server work without population)
	buildings, err := app.Dao().FindRecordsByFilter("buildings", 
		fmt.Sprintf("planet_id = '%s' && active = true", planet.Id), "", 0, 0)
	if err != nil {
		log.Printf("Failed to get buildings for planet %s: %v", planet.Id, err)
		return nil
	}

	// Check if there are any crypto_servers that can work without population
	hasCryptoServers := false
	for _, building := range buildings {
		buildingTypeID := building.GetString("building_type")
		buildingType, err := app.Dao().FindRecordById("building_types", buildingTypeID)
		if err == nil && buildingType.GetString("name") == "crypto_server" {
			hasCryptoServers = true
			break
		}
	}

	// Only skip if no population AND no crypto servers
	if totalPop == 0 && !hasCryptoServers {
		return nil // No population or crypto servers to simulate
	}

	// Auto-employ population to buildings if not already employed
	if totalPop > 0 {
		log.Printf("DEBUG: Planet %s has %d total population, checking employment", planet.Id, totalPop)
		
		unemployedPops := make([]*models.Record, 0)
		for _, pop := range populations {
			if pop.GetString("employed_at") == "" {
				unemployedPops = append(unemployedPops, pop)
				log.Printf("DEBUG: Found unemployed population: %d people", pop.GetInt("count"))
			}
		}

		log.Printf("DEBUG: Found %d unemployed population groups", len(unemployedPops))

		// Assign unemployed population to buildings that need workers
		for _, building := range buildings {
			if len(unemployedPops) == 0 {
				break
			}

			buildingTypeID := building.GetString("building_type")
			buildingType, err := app.Dao().FindRecordById("building_types", buildingTypeID)
			if err != nil {
				log.Printf("DEBUG: Failed to get building type %s: %v", buildingTypeID, err)
				continue
			}

			buildingName := buildingType.GetString("name")
			log.Printf("DEBUG: Processing building %s (type: %s)", building.Id, buildingName)

			// Skip crypto servers (they don't need workers)
			if buildingName == "crypto_server" {
				log.Printf("DEBUG: Skipping crypto server %s", building.Id)
				continue
			}

			level := building.GetInt("level")
			if level == 0 {
				level = 1
			}
			// Use default worker capacity since the field doesn't exist in schema
			var baseWorkerCapacity int
			switch buildingName {
			case "mine":
				baseWorkerCapacity = 10
			case "farm":
				baseWorkerCapacity = 8
			case "factory":
				baseWorkerCapacity = 12
			case "power_plant":
				baseWorkerCapacity = 6
			case "oil_rig":
				baseWorkerCapacity = 8
			case "deep_mine":
				baseWorkerCapacity = 15
			case "metal_refinery":
				baseWorkerCapacity = 10
			case "oil_refinery":
				baseWorkerCapacity = 10
			default:
				baseWorkerCapacity = 5 // Default for other buildings
			}
			workerCapacity := baseWorkerCapacity * level

			// Count currently employed at this building
			currentWorkers := 0
			for _, pop := range populations {
				if pop.GetString("employed_at") == building.Id {
					currentWorkers += pop.GetInt("count")
				}
			}

			log.Printf("DEBUG: Building %s needs %d workers, has %d workers", building.Id, workerCapacity, currentWorkers)

			// Assign more workers if needed and available
			if currentWorkers < workerCapacity && len(unemployedPops) > 0 {
				pop := unemployedPops[0]
				pop.Set("employed_at", building.Id)
				if err := app.Dao().SaveRecord(pop); err != nil {
					log.Printf("Failed to employ population: %v", err)
				} else {
					log.Printf("Employed %d population to work %s building %s", 
						pop.GetInt("count"), buildingType.GetString("name"), building.Id)
					unemployedPops = unemployedPops[1:] // Remove from unemployed list
				}
			}
		}
		
		log.Printf("DEBUG: Employment complete. Remaining unemployed groups: %d", len(unemployedPops))
	}

	// Get resource nodes on this planet
	resourceNodes, err := app.Dao().FindRecordsByFilter("resource_nodes", 
		fmt.Sprintf("planet_id = '%s' && exhausted = false", planet.Id), "", 0, 0)
	if err != nil {
		log.Printf("Failed to get resource nodes for planet %s: %v", planet.Id, err)
		return nil
	}

	// Calculate production from buildings working resource nodes
	production := make(map[string]int)
	
	for _, building := range buildings {
		buildingTypeID := building.GetString("building_type")
		buildingType, err := app.Dao().FindRecordById("building_types", buildingTypeID)
		if err != nil {
			continue
		}

		level := building.GetInt("level")
		if level == 0 {
			level = 1
		}

		// Use default worker capacity since the field doesn't exist in schema
		var baseWorkerCapacity int
		buildingName := buildingType.GetString("name")
		switch buildingName {
		case "mine":
			baseWorkerCapacity = 10
		case "farm":
			baseWorkerCapacity = 8
		case "factory":
			baseWorkerCapacity = 12
		case "power_plant":
			baseWorkerCapacity = 6
		case "oil_rig":
			baseWorkerCapacity = 8
		case "deep_mine":
			baseWorkerCapacity = 15
		case "metal_refinery":
			baseWorkerCapacity = 10
		case "oil_refinery":
			baseWorkerCapacity = 10
		default:
			baseWorkerCapacity = 5 // Default for other buildings
		}
		workerCapacity := baseWorkerCapacity * level

		// Get employed population for this building
		employedPops, err := app.Dao().FindRecordsByFilter("populations",
			fmt.Sprintf("employed_at = '%s'", building.Id), "", 0, 0)
		if err != nil {
			continue
		}

		workers := 0
		for _, emp := range employedPops {
			workers += emp.GetInt("count")
		}

		if workers > workerCapacity {
			workers = workerCapacity
		}

		// Calculate efficiency based on happiness and workers
		efficiency := float64(totalHappiness) / 100.0 * float64(workers) / float64(workerCapacity)
		if efficiency > 1.0 {
			efficiency = 1.0
		}

		// Handle crypto_servers first (they work without workers)
		buildingName = buildingType.GetString("name")
		if buildingName == "crypto_server" {
			// Crypto servers produce credits automatically without needing workers
			creditsProduced := buildingType.GetInt("res1_quantity") * level
			
			// Add credits to the building's storage (up to capacity)
			currentStored := building.GetInt("res1_stored")
			capacity := buildingType.GetInt("res1_capacity") * level
			newStored := currentStored + creditsProduced
			if newStored > capacity {
				newStored = capacity
			}
			
			building.Set("res1_stored", newStored)
			if err := app.Dao().SaveRecord(building); err != nil {
				log.Printf("Failed to update crypto_server storage: %v", err)
			} else {
				log.Printf("Crypto server %s produced %d credits (stored: %d/%d)", 
					building.Id, creditsProduced, newStored, capacity)
			}
			continue // Skip worker-dependent processing for crypto_servers
		}

		// Different building types produce different resources (worker-dependent)
		baseProduction := int(float64(level*20) * efficiency)

		switch buildingName {
		case "farm":
			production["food"] += baseProduction
		case "mine":
			// Mines store ore in building storage instead of global production
			oreProduced := 0
			for _, node := range resourceNodes {
				resourceType, err := app.Dao().FindRecordById("resource_types", node.GetString("resource_type"))
				if err != nil {
					continue
				}
				if resourceType.GetString("name") == "ore" {
					richness := node.GetInt("richness")
					oreProduced += baseProduction * richness / 5
					break
				}
			}
			
			if oreProduced > 0 {
				// Store ore directly in mine building storage
				currentStored := building.GetInt("res1_stored")
				// Use massive capacity of 100,000 per level for better visualization
				capacity := 100000 * level
				newStored := currentStored + oreProduced
				if newStored > capacity {
					newStored = capacity
				}
				
				building.Set("res1_stored", newStored)
				if err := app.Dao().SaveRecord(building); err != nil {
					log.Printf("Failed to update mine storage: %v", err)
				} else {
					log.Printf("Mine %s produced %d ore (stored: %d/%d)", 
						building.Id, oreProduced, newStored, capacity)
				}
			} else {
				log.Printf("DEBUG: Mine %s produced 0 ore (efficiency: %f, workers: %d)", building.Id, efficiency, workers)
			}
		case "factory":
			production["goods"] += baseProduction
		case "power_plant":
			production["fuel"] += baseProduction
		}
	}

	// Get owner's current resources (stored globally per user)
	owner, err := app.Dao().FindRecordById("users", ownerID)
	if err != nil {
		log.Printf("Failed to get owner for planet %s: %v", planet.Id, err)
		return nil
	}

	// Update owner's resources from production
	for resource, amount := range production {
		currentAmount := owner.GetInt(resource)
		owner.Set(resource, currentAmount + amount)
	}

	// Only process consumption and population growth if there's population
	if totalPop > 0 {
		// Basic consumption for population
		consumption := map[string]int{
			"food": totalPop / 5,
			"fuel": totalPop / 10,
		}

		for resource, amount := range consumption {
			currentAmount := owner.GetInt(resource)
			newAmount := currentAmount - amount
			if newAmount < 0 {
				// Resource shortage affects happiness
				for _, pop := range populations {
					happiness := pop.GetInt("happiness")
					happiness -= 5
					if happiness < 0 {
						happiness = 0
					}
					pop.Set("happiness", happiness)
					app.Dao().SaveRecord(pop)
				}
				newAmount = 0
			}
			owner.Set(resource, newAmount)
		}

		// Population growth/decline
		if totalHappiness > 70 && production["food"] > consumption["food"] && totalPop < maxPop {
			// Population growth
			growthRate := 0.02 * (float64(totalHappiness) / 100.0)
			growth := int(float64(totalPop) * growthRate)
			if growth < 1 {
				growth = 1
			}

			if len(populations) > 0 {
				mainPop := populations[0]
				newCount := mainPop.GetInt("count") + growth
				if totalPop + growth > maxPop {
					newCount = maxPop - (totalPop - mainPop.GetInt("count"))
				}
				mainPop.Set("count", newCount)
				app.Dao().SaveRecord(mainPop)
			}
		}
	}

	// Save owner's updated resources
	if err := app.Dao().SaveRecord(owner); err != nil {
		log.Printf("Failed to save owner resources: %v", err)
	}

	return nil
}

// ProcessBankIncome handles legacy bank income (might not exist in new schema)
func ProcessBankIncome(app *pocketbase.PocketBase) error {
	// Check if banks collection exists
	_, err := app.Dao().FindCollectionByNameOrId("banks")
	if err != nil {
		// Banks don't exist in new schema, skip
		return nil
	}

	// Get all users 
	users, err := app.Dao().FindRecordsByExpr("users", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	log.Printf("Processing banking income for all users...")

	for _, user := range users {
		if err := processUserBankingIncome(app, user); err != nil {
			log.Printf("Failed to process banking income for user %s: %v", user.Id, err)
			continue
		}
	}

	return nil
}

// processUserBankingIncome handles banking income for a single user (legacy)
func processUserBankingIncome(app *pocketbase.PocketBase, user *models.Record) error {
	// Count active banks owned by this user
	bankCount, err := app.Dao().FindRecordsByFilter("banks", fmt.Sprintf("owner_id = '%s' && active = true", user.Id), "", 0, 0)
	if err != nil {
		return nil // Banks collection doesn't exist
	}

	creditsPerTick := len(bankCount)
	if creditsPerTick <= 0 {
		return nil // No banking income
	}

	// Add credits to user's balance
	currentCredits := user.GetInt("credits")
	newCredits := currentCredits + creditsPerTick
	user.Set("credits", newCredits)

	if err := app.Dao().SaveRecord(user); err != nil {
		return fmt.Errorf("failed to update user credits: %w", err)
	}

	log.Printf("User %s earned %d credits from %d banks (new balance: %d)", 
		user.Id, creditsPerTick, len(bankCount), newCredits)

	return nil
}

// GetPlanetValue calculates the total value of a planet's economy
func GetPlanetValue(app *pocketbase.PocketBase, planet *models.Record) int {
	value := 0

	// Get buildings on planet
	buildings, err := app.Dao().FindRecordsByFilter("buildings", 
		fmt.Sprintf("planet_id = '%s'", planet.Id), "", 0, 0)
	if err != nil {
		return 0
	}

	for _, building := range buildings {
		buildingType, err := app.Dao().FindRecordById("building_types", building.GetString("building_type"))
		if err != nil {
			continue
		}
		
		level := building.GetInt("level")
		cost := buildingType.GetInt("cost")
		value += cost * level
	}

	// Get population value
	populations, err := app.Dao().FindRecordsByFilter("populations", 
		fmt.Sprintf("planet_id = '%s'", planet.Id), "", 0, 0)
	if err == nil {
		for _, pop := range populations {
			value += pop.GetInt("count") * 10 // 10 credits per population
		}
	}

	return value
}