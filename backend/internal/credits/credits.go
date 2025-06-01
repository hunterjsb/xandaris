package credits

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
)

// GetUserCredits gets the total credits available to a user from all their crypto_server buildings
func GetUserCredits(app *pocketbase.PocketBase, userID string) (int, error) {
	// Get all planets owned by the user
	planets, err := app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("colonized_by = '%s'", userID), "", 0, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to get user planets: %w", err)
	}

	totalCredits := 0
	
	for _, planet := range planets {
		// Get crypto_server buildings on this planet
		buildings, err := app.Dao().FindRecordsByFilter("buildings", fmt.Sprintf("planet_id = '%s' && active = true", planet.Id), "", 0, 0)
		if err != nil {
			continue
		}

		for _, building := range buildings {
			buildingTypeID := building.GetString("building_type")
			buildingType, err := app.Dao().FindRecordById("building_types", buildingTypeID)
			if err != nil {
				continue
			}

			// Check if this is a crypto_server building
			if buildingType.GetString("name") == "crypto_server" {
				// Get credits resource type ID
				creditsResource, err := app.Dao().FindFirstRecordByFilter("resource_types", "name = 'credits'")
				if err != nil {
					continue
				}

				// Check if this building produces credits
				res1TypeID := buildingType.GetString("res1_type")
				if res1TypeID == creditsResource.Id {
					// Get current credits stored in this building
					storedCredits := building.GetInt("res1_stored")
					totalCredits += storedCredits
				}
			}
		}
	}

	return totalCredits, nil
}

// DeductUserCredits deducts credits from user's crypto_server buildings
func DeductUserCredits(app *pocketbase.PocketBase, userID string, amount int) error {
	// Get all planets owned by the user
	planets, err := app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("colonized_by = '%s'", userID), "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to get user planets: %w", err)
	}

	remainingToDeduct := amount
	
	for _, planet := range planets {
		if remainingToDeduct <= 0 {
			break
		}

		// Get crypto_server buildings on this planet
		buildings, err := app.Dao().FindRecordsByFilter("buildings", fmt.Sprintf("planet_id = '%s' && active = true", planet.Id), "", 0, 0)
		if err != nil {
			continue
		}

		for _, building := range buildings {
			if remainingToDeduct <= 0 {
				break
			}

			buildingTypeID := building.GetString("building_type")
			buildingType, err := app.Dao().FindRecordById("building_types", buildingTypeID)
			if err != nil {
				continue
			}

			// Check if this is a crypto_server building
			if buildingType.GetString("name") == "crypto_server" {
				// Get credits resource type ID
				creditsResource, err := app.Dao().FindFirstRecordByFilter("resource_types", "name = 'credits'")
				if err != nil {
					continue
				}

				// Check if this building stores credits
				res1TypeID := buildingType.GetString("res1_type")
				if res1TypeID == creditsResource.Id {
					// Get current credits stored in this building
					storedCredits := building.GetInt("res1_stored")
					
					if storedCredits > 0 {
						// Deduct from this building's storage
						if storedCredits >= remainingToDeduct {
							// This building has enough credits
							building.Set("res1_stored", storedCredits-remainingToDeduct)
							remainingToDeduct = 0
						} else {
							// Deduct what we can from this building
							building.Set("res1_stored", 0)
							remainingToDeduct -= storedCredits
						}
						
						// Save the building
						if err := app.Dao().SaveRecord(building); err != nil {
							return fmt.Errorf("failed to update building storage: %w", err)
						}
					}
				}
			}
		}
	}

	if remainingToDeduct > 0 {
		return fmt.Errorf("insufficient credits: could not deduct %d credits", remainingToDeduct)
	}

	log.Printf("Deducted %d credits from user %s crypto_server buildings", amount, userID)
	return nil
}

// HasSufficientCredits checks if the user has enough credits
func HasSufficientCredits(app *pocketbase.PocketBase, userID string, amount int) (bool, error) {
	currentCredits, err := GetUserCredits(app, userID)
	if err != nil {
		return false, err
	}
	return currentCredits >= amount, nil
}

// AddCreditsToBuilding adds credits to a specific crypto_server building's storage
func AddCreditsToBuilding(app *pocketbase.PocketBase, buildingID string, amount int) error {
	building, err := app.Dao().FindRecordById("buildings", buildingID)
	if err != nil {
		return fmt.Errorf("failed to get building: %w", err)
	}

	buildingTypeID := building.GetString("building_type")
	buildingType, err := app.Dao().FindRecordById("building_types", buildingTypeID)
	if err != nil {
		return fmt.Errorf("failed to get building type: %w", err)
	}

	// Check if this is a crypto_server building
	if buildingType.GetString("name") != "crypto_server" {
		return fmt.Errorf("building is not a crypto_server")
	}

	// Get credits resource type ID
	creditsResource, err := app.Dao().FindFirstRecordByFilter("resource_types", "name = 'credits'")
	if err != nil {
		return fmt.Errorf("failed to get credits resource type: %w", err)
	}

	// Check if this building stores credits
	res1TypeID := buildingType.GetString("res1_type")
	if res1TypeID != creditsResource.Id {
		return fmt.Errorf("building does not store credits")
	}

	// Get current stored amount and capacity
	currentStored := building.GetInt("res1_stored")
	level := building.GetInt("level")
	if level == 0 {
		level = 1
	}
	capacity := buildingType.GetInt("res1_capacity") * level

	// Calculate new stored amount, capped at capacity
	newStored := currentStored + amount
	if newStored > capacity {
		newStored = capacity
	}

	building.Set("res1_stored", newStored)
	if err := app.Dao().SaveRecord(building); err != nil {
		return fmt.Errorf("failed to update building storage: %w", err)
	}

	log.Printf("Added %d credits to building %s (new total: %d/%d)", amount, buildingID, newStored, capacity)
	return nil
}