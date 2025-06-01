package pkg

import (
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"

	mapgen "github.com/hunterjsb/xandaris/internal/map"
	"github.com/hunterjsb/xandaris/internal/tick"
	"github.com/hunterjsb/xandaris/internal/worldgen"
)

// MapHandler returns the current map data
func MapHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		mapData, err := mapgen.GetMapData(app)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to fetch map data",
			})
		}

		return c.JSON(http.StatusOK, mapData)
	}
}

// FleetOrderHandler handles fleet movement orders
func FleetOrderHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get authenticated user from PocketBase context
		info := c.Get(apis.ContextAuthRecordKey)
		if info == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Authentication required",
			})
		}

		// Parse request
		var req struct {
			FromID   string `json:"from_id"`
			ToID     string `json:"to_id"`
			Strength int    `json:"strength"`
		}

		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request format",
			})
		}

		// Validate input
		if req.FromID == "" || req.ToID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Missing required fields",
			})
		}

		user := info.(*models.Record)

		// Check if both systems exist
		_, err := app.Dao().FindRecordById("systems", req.FromID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Source system not found",
			})
		}

		_, err = app.Dao().FindRecordById("systems", req.ToID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Target system not found",
			})
		}

		// Find an existing fleet at the source system that's NOT already moving
		fleetFilter := fmt.Sprintf("owner_id='%s' && current_system='%s' && (destination_system='' || destination_system IS NULL)", user.Id, req.FromID)
		fmt.Printf("DEBUG: Fleet filter: %s\n", fleetFilter)
		fleets, err := app.Dao().FindRecordsByFilter("fleets", fleetFilter, "", 1, 0)
		if err != nil {
			fmt.Printf("DEBUG: Fleet query error: %v\n", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to find fleets: " + err.Error(),
			})
		}

		fmt.Printf("DEBUG: Found %d fleets\n", len(fleets))
		if len(fleets) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "No available fleets at source system",
			})
		}

		// Use the first available fleet
		fleet := fleets[0]
		fmt.Printf("DEBUG: Moving fleet %s from %s to %s\n", fleet.Id, fleet.GetString("current_system"), req.ToID)

		// Calculate ETA (2 minutes from now)
		etaTime := time.Now().Add(2 * time.Minute)

		// Move the existing fleet by setting destination and ETA
		fleet.Set("destination_system", req.ToID)
		fleet.Set("eta", etaTime)
		fmt.Printf("DEBUG: Set destination_system=%s, eta=%s\n", req.ToID, etaTime.Format("2006-01-02 15:04:05"))

		if err := app.Dao().SaveRecord(fleet); err != nil {
			fmt.Printf("DEBUG: Failed to save fleet: %v\n", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to move fleet: " + err.Error(),
			})
		}
		fmt.Printf("DEBUG: Fleet saved successfully!\n")

		return c.JSON(http.StatusOK, map[string]interface{}{
			"fleet_id": fleet.Id,
			"success":  true,
			"message":  fmt.Sprintf("Fleet %s dispatched from %s to %s", fleet.Id[:8], req.FromID[:8], req.ToID[:8]),
		})
	}
}

// BuildOrderHandler handles building construction orders
func BuildOrderHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check authentication
		info := c.Get(apis.ContextAuthRecordKey)
		if info == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Authentication required",
			})
		}

		// Parse request
		var req struct {
			SystemID     string `json:"system_id"`
			BuildingType string `json:"building_type"`
		}

		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request format",
			})
		}

		user := info.(*models.Record)

		// Get system
		system, err := app.Dao().FindRecordById("systems", req.SystemID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "System not found",
			})
		}

		// Verify ownership
		if system.GetString("owner_id") != user.Id {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "You don't own this system",
			})
		}

		// Validate building type and upgrade
		var fieldName string
		var cost int

		switch req.BuildingType {
		case "habitat":
			fieldName = "hab_lvl"
			cost = 100 * (system.GetInt("hab_lvl") + 1)
		case "farm":
			fieldName = "farm_lvl"
			cost = 150 * (system.GetInt("farm_lvl") + 1)
		case "mine":
			fieldName = "mine_lvl"
			cost = 200 * (system.GetInt("mine_lvl") + 1)
		case "factory":
			fieldName = "fac_lvl"
			cost = 300 * (system.GetInt("fac_lvl") + 1)
		case "shipyard":
			fieldName = "yard_lvl"
			cost = 500 * (system.GetInt("yard_lvl") + 1)
		case "bank":
			// Banks are special - they create a separate bank record
			return handleBankConstruction(app, c, user, system, req.BuildingType)
		default:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid building type",
			})
		}

		// Check if max level reached
		currentLevel := system.GetInt(fieldName)
		if currentLevel >= 10 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Maximum building level reached",
			})
		}

		// Check resources (using goods as currency for now)
		if system.GetInt("goods") < cost {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("Insufficient goods. Required: %d", cost),
			})
		}

		// Apply upgrade immediately (no build queue for now)
		system.Set(fieldName, currentLevel+1)
		system.Set("goods", system.GetInt("goods")-cost)

		if err := app.Dao().SaveRecord(system); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to upgrade building",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"building_type": req.BuildingType,
			"new_level":     currentLevel + 1,
			"cost":          cost,
		})
	}
}

// TradeOrderHandler handles trade route creation
func TradeOrderHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check authentication
		info := c.Get(apis.ContextAuthRecordKey)
		if info == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Authentication required",
			})
		}

		// Parse request
		var req struct {
			FromID   string `json:"from_id"`
			ToID     string `json:"to_id"`
			Cargo    string `json:"cargo"`
			Capacity int    `json:"capacity"`
		}

		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request format",
			})
		}

		user := info.(*models.Record)

		// Verify ownership of source system
		fromSystem, err := app.Dao().FindRecordById("systems", req.FromID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Source system not found",
			})
		}

		if fromSystem.GetString("owner_id") != user.Id {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "You don't own the source system",
			})
		}

		// Validate cargo type
		validCargo := map[string]bool{
			"food":  true,
			"ore":   true,
			"goods": true,
			"fuel":  true,
		}

		if !validCargo[req.Cargo] {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid cargo type",
			})
		}

		// Calculate first ETA (12 ticks travel time = 2 minutes at 6 ticks/minute)
		currentTick := tick.GetCurrentTick(app)
		travelTime := int64(12)
		etaTick := currentTick + travelTime

		// Create trade route record
		tradeCollection, err := app.Dao().FindCollectionByNameOrId("trade_routes")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Trade routes collection not found",
			})
		}

		trade := models.NewRecord(tradeCollection)
		trade.Set("owner_id", user.Id)
		trade.Set("from_id", req.FromID)
		trade.Set("to_id", req.ToID)
		trade.Set("cargo", req.Cargo)
		trade.Set("cap", req.Capacity)
		trade.Set("eta_tick", etaTick)

		if err := app.Dao().SaveRecord(trade); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create trade route",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"trade_id": trade.Id,
			"eta_tick": etaTick,
		})
	}
}

// DiplomacyHandler handles treaty proposals
func DiplomacyHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check authentication
		info := c.Get(apis.ContextAuthRecordKey)
		if info == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Authentication required",
			})
		}

		// Parse request
		var req struct {
			PlayerID string `json:"player_id"`
			Type     string `json:"type"`
			Terms    string `json:"terms"`
		}

		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request format",
			})
		}

		user := info.(*models.Record)

		// Validate treaty type
		validTypes := map[string]bool{
			"alliance":       true,
			"trade_pact":     true,
			"non_aggression": true,
			"ceasefire":      true,
		}

		if !validTypes[req.Type] {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid treaty type",
			})
		}

		// Create treaty record
		treatyCollection, err := app.Dao().FindCollectionByNameOrId("treaties")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Treaties collection not found",
			})
		}

		treaty := models.NewRecord(treatyCollection)
		treaty.Set("type", req.Type)
		treaty.Set("a_id", user.Id)
		treaty.Set("b_id", req.PlayerID)
		treaty.Set("created_at", time.Now())
		treaty.Set("status", "proposed")

		if err := app.Dao().SaveRecord(treaty); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create treaty",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"treaty_id": treaty.Id,
			"status":    "proposed",
		})
	}
}

// handleBankConstruction creates a new crypto server bank
func handleBankConstruction(app *pocketbase.PocketBase, c echo.Context, user *models.Record, system *models.Record, buildingType string) error {
	// Bank construction costs 1000 credits from user's global balance
	cost := 1000
	userCredits := user.GetInt("credits")
	
	if userCredits < cost {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Insufficient credits. Required: %d, Have: %d", cost, userCredits),
		})
	}

	// Check if system already has a bank (limit 1 per system)
	existingBank, err := app.Dao().FindFirstRecordByFilter("banks", "system_id = {:systemId}", map[string]interface{}{
		"systemId": system.Id,
	})
	if err == nil && existingBank != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "System already has a crypto server bank",
		})
	}

	// Create the bank
	bankCollection, err := app.Dao().FindCollectionByNameOrId("banks")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Banks collection not found",
		})
	}

	bank := models.NewRecord(bankCollection)
	bank.Set("name", fmt.Sprintf("CryptoServer-%s", system.Id[:8]))
	bank.Set("owner_id", user.Id)
	bank.Set("system_id", system.Id)
	bank.Set("security_level", 1)      // Starting security level
	bank.Set("processing_power", 10)   // Starting processing power
	bank.Set("credits_per_tick", 1)    // 1 credit per tick income
	bank.Set("active", true)
	bank.Set("last_income_tick", tick.GetCurrentTick(app))

	if err := app.Dao().SaveRecord(bank); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create crypto server",
		})
	}

	// Deduct credits from user
	user.Set("credits", userCredits-cost)
	if err := app.Dao().SaveRecord(user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to deduct credits",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"bank_id":     bank.Id,
		"name":        bank.GetString("name"),
		"income":      1,
		"cost":        cost,
		"new_balance": userCredits - cost,
	})
}

// getWorldgen generates a random system using worldgen
func getWorldgen(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Generate a random seed
		seed32 := worldgen.GenerateRandomSystemSeed()
		system := worldgen.ProcessSystemSeed(seed32)
		return c.JSON(http.StatusOK, system)
	}
}

// getWorldgenWithSeed generates a system using a provided seed
func getWorldgenWithSeed(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		seedStr := c.PathParam("seed")
		if seedStr == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Seed parameter required",
			})
		}

		// Parse seed as big.Int
		seed32 := new(big.Int)
		_, ok := seed32.SetString(seedStr, 10)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid seed format",
			})
		}

		system := worldgen.ProcessSystemSeed(seed32)
		return c.JSON(http.StatusOK, system)
	}
}