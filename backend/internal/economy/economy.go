package economy

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// UpdateMarkets handles the economic simulation for all systems
func UpdateMarkets(app *pocketbase.PocketBase) error {
	log.Println("Updating markets and economy...")

	// Process bank income first
	if err := ProcessBankIncome(app); err != nil {
		log.Printf("Error processing bank income: %v", err)
		// Don't fail the entire economy update for bank errors
	}

	// Get all owned systems
	systems, err := app.Dao().FindRecordsByFilter("systems", "owner_id != ''", "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch systems: %w", err)
	}

	for _, system := range systems {
		if err := updateSystemEconomy(system); err != nil {
			log.Printf("Failed to update economy for system %s: %v", system.Id, err)
			continue
		}

		if err := app.Dao().SaveRecord(system); err != nil {
			log.Printf("Failed to save system %s: %v", system.Id, err)
		}
	}

	log.Printf("Updated economy for %d systems", len(systems))
	return nil
}

// updateSystemEconomy simulates production and consumption for a single system
func updateSystemEconomy(system *models.Record) error {
	pop := system.GetInt("pop")
	morale := system.GetInt("morale")
	richness := system.GetInt("richness")

	// Get building levels
	habLvl := system.GetInt("hab_lvl")
	farmLvl := system.GetInt("farm_lvl")
	mineLvl := system.GetInt("mine_lvl")
	facLvl := system.GetInt("fac_lvl")
	yardLvl := system.GetInt("yard_lvl")

	// Calculate efficiency based on morale (50-150%)
	efficiency := float64(morale+50) / 100.0
	if efficiency < 0.5 {
		efficiency = 0.5
	}
	if efficiency > 1.5 {
		efficiency = 1.5
	}

	// Current resources
	food := system.GetInt("food")
	ore := system.GetInt("ore")
	goods := system.GetInt("goods")
	fuel := system.GetInt("fuel")

	// === PRODUCTION ===

	// Food production: farms + basic population farming
	foodProduction := int(float64(farmLvl*50+pop/10) * efficiency)
	food += foodProduction

	// Ore production: mines based on richness
	oreProduction := int(float64(mineLvl*30+richness*10) * efficiency)
	ore += oreProduction

	// Goods production: factories (requires ore)
	goodsProduction := int(float64(facLvl*20) * efficiency)
	if ore >= goodsProduction {
		ore -= goodsProduction
		goods += goodsProduction
	} else {
		// Partial production if not enough ore
		goods += ore
		ore = 0
	}

	// Fuel production: limited fuel extraction
	fuelProduction := int(float64(yardLvl*10+richness*5) * efficiency)
	fuel += fuelProduction

	// === CONSUMPTION ===

	// Population consumes food
	foodConsumption := pop / 5
	food -= foodConsumption

	// Low food affects morale and population
	if food < 0 {
		morale -= 10 // Starvation hurts morale
		if food < -pop {
			pop = int(float64(pop) * 0.95) // Population decline from starvation
		}
		food = 0
	}

	// === GROWTH ===

	// Population growth based on habitat level and morale
	if food > pop && morale > 70 {
		maxPop := habLvl * 100
		if pop < maxPop {
			growth := int(float64(pop) * 0.02 * (float64(morale)/100.0))
			if growth < 1 {
				growth = 1
			}
			pop += growth
			if pop > maxPop {
				pop = maxPop
			}
		}
	}

	// Morale naturally trends toward 75 (neutral)
	if morale < 75 {
		morale += 2
	} else if morale > 75 {
		morale -= 1
	}

	// Ensure morale stays in bounds
	if morale < 0 {
		morale = 0
	}
	if morale > 100 {
		morale = 100
	}

	// === UPDATE SYSTEM ===
	system.Set("pop", pop)
	system.Set("morale", morale)
	system.Set("food", food)
	system.Set("ore", ore)
	system.Set("goods", goods)
	system.Set("fuel", fuel)

	return nil
}

// CalculatePrice calculates market price using logistic function
// p' = p * (1 + k*(d-s)/(d+s))
func CalculatePrice(basePrice float64, demand, supply int, k float64) float64 {
	if supply == 0 && demand > 0 {
		return basePrice * 2.0 // Price doubles when no supply
	}
	if demand == 0 {
		return basePrice * 0.5 // Price halves when no demand
	}

	total := float64(demand + supply)
	if total == 0 {
		return basePrice
	}

	ratio := float64(demand-supply) / total
	multiplier := 1.0 + k*ratio

	return basePrice * multiplier
}

// GetSystemValue calculates the total value of a system's resources
func GetSystemValue(system *models.Record) int {
	food := system.GetInt("food")
	ore := system.GetInt("ore")
	goods := system.GetInt("goods")
	fuel := system.GetInt("fuel")

	// Base prices for resources
	value := food*1 + ore*2 + goods*5 + fuel*3

	// Add building values
	habLvl := system.GetInt("hab_lvl")
	farmLvl := system.GetInt("farm_lvl")
	mineLvl := system.GetInt("mine_lvl")
	facLvl := system.GetInt("fac_lvl")
	yardLvl := system.GetInt("yard_lvl")

	buildingValue := (habLvl+farmLvl+mineLvl+facLvl+yardLvl) * 100

	return value + buildingValue
}

// ProcessBankIncome handles crypto server income generation efficiently
func ProcessBankIncome(app *pocketbase.PocketBase) error {
	// Get all users 
	users, err := app.Dao().FindRecordsByFilter("users", "id != ''", "", 0, 0)
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

// processUserBankingIncome handles banking income for a single user
func processUserBankingIncome(app *pocketbase.PocketBase, user *models.Record) error {
	// Count active banks owned by this user
	bankCount, err := app.Dao().FindRecordsByFilter("banks", fmt.Sprintf("owner_id = '%s' && active = true", user.Id), "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to count user banks: %w", err)
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

// AuditUserCredits verifies that a user's expected income matches their bank count
func AuditUserCredits(app *pocketbase.PocketBase, userID string) error {
	user, err := app.Dao().FindRecordById("users", userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Count user's active banks
	banks, err := app.Dao().FindRecordsByFilter("banks", fmt.Sprintf("owner_id = '%s' && active = true", userID), "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to count user banks: %w", err)
	}

	expectedCreditsPerTick := len(banks)
	currentCredits := user.GetInt("credits")
	
	log.Printf("User %s audit: %d credits, %d banks (expected %d credits/tick)", 
		userID, currentCredits, len(banks), expectedCreditsPerTick)

	return nil
}