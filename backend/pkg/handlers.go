package pkg

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"

	mapgen "github.com/hunterjsb/xandaris/internal/map"
	"github.com/hunterjsb/xandaris/internal/tick"
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
		if req.FromID == "" || req.ToID == "" || req.Strength <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Missing required fields",
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

		// Check if target system exists
		_, err = app.Dao().FindRecordById("systems", req.ToID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Target system not found",
			})
		}

		// Check if source has enough population
		if fromSystem.GetInt("pop") < req.Strength*10 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Insufficient population for fleet strength",
			})
		}

		// Calculate ETA (12 ticks travel time = 2 minutes at 6 ticks/minute)
		currentTick := tick.GetCurrentTick(app)
		travelTime := int64(12)
		etaTick := currentTick + travelTime

		// Create fleet record
		fleetCollection, err := app.Dao().FindCollectionByNameOrId("fleets")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Fleet collection not found",
			})
		}

		fleet := models.NewRecord(fleetCollection)
		fleet.Set("owner_id", user.Id)
		fleet.Set("from_id", req.FromID)
		fleet.Set("to_id", req.ToID)
		fleet.Set("eta_tick", etaTick)
		fleet.Set("strength", req.Strength)

		if err := app.Dao().SaveRecord(fleet); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create fleet",
			})
		}

		// Reduce population from source system
		fromSystem.Set("pop", fromSystem.GetInt("pop")-req.Strength*10)
		if err := app.Dao().SaveRecord(fromSystem); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update source system",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"fleet_id": fleet.Id,
			"eta_tick": etaTick,
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

		// Validate building type and calculate cost and build time
		var fieldName string
		var cost int
		var buildTimeTicks int64
		currentLevel := 0

		// Define build times for each building type
		// (Habitat: 5 ticks, Farm: 8 ticks, Mine: 10, Factory: 12, Shipyard: 15, Bank: 20)
		buildTimes := map[string]int64{
			"habitat":  5,
			"farm":     8,
			"mine":     10,
			"factory":  12,
			"shipyard": 15,
			"bank":     20,
		}

		switch req.BuildingType {
		case "habitat":
			fieldName = "hab_lvl"
			currentLevel = system.GetInt("hab_lvl")
			cost = 100 * (currentLevel + 1)
			buildTimeTicks = buildTimes["habitat"]
		case "farm":
			fieldName = "farm_lvl"
			currentLevel = system.GetInt("farm_lvl")
			cost = 150 * (currentLevel + 1)
			buildTimeTicks = buildTimes["farm"]
		case "mine":
			fieldName = "mine_lvl"
			currentLevel = system.GetInt("mine_lvl")
			cost = 200 * (currentLevel + 1)
			buildTimeTicks = buildTimes["mine"]
		case "factory":
			fieldName = "fac_lvl"
			currentLevel = system.GetInt("fac_lvl")
			cost = 300 * (currentLevel + 1)
			buildTimeTicks = buildTimes["factory"]
		case "shipyard":
			fieldName = "yard_lvl"
			currentLevel = system.GetInt("yard_lvl")
			cost = 500 * (currentLevel + 1)
			buildTimeTicks = buildTimes["shipyard"]
		case "bank":
			// Banks are special: cost is global credits, level isn't directly on system
			cost = 1000 // Fixed cost for bank
			buildTimeTicks = buildTimes["bank"]
			// Check if system already has a bank or a bank is being built
			existingBank, _ := app.Dao().FindFirstRecordByFilter("banks", "system_id = {:systemId}", map[string]interface{}{"systemId": system.Id})
			if existingBank != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "System already has a crypto server bank"})
			}
			// Check building queue for pending bank
			pendingBankBuild, _ := app.Dao().FindFirstRecordByFilter("building_queue", "system_id = {:systemId} && building_type = 'bank'", map[string]interface{}{"systemId": system.Id})
			if pendingBankBuild != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Crypto server bank construction already in progress for this system"})
			}

			userCredits := user.GetInt("credits")
			if userCredits < cost {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("Insufficient credits. Required: %d, Have: %d", cost, userCredits),
				})
			}
			user.Set("credits", userCredits-cost)
			if err := app.Dao().SaveRecord(user); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to deduct credits"})
			}
		default:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid building type",
			})
		}

		// Check if max level reached (not applicable for bank in this direct check)
		if req.BuildingType != "bank" {
			if currentLevel >= 10 {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Maximum building level reached",
				})
			}
			// Check resources (using goods as currency for now) for non-bank buildings
			if system.GetInt("goods") < cost {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("Insufficient goods. Required: %d", cost),
				})
			}
			// Deduct resources for non-bank buildings
			system.Set("goods", system.GetInt("goods")-cost)
			if err := app.Dao().SaveRecord(system); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to deduct goods from system"})
			}
		}

		// Check if there's already a building of the same type in the queue for this system
		existingQueueItem, _ := app.Dao().FindFirstRecordByFilter(
			"building_queue",
			"system_id = {:systemId} && building_type = {:buildingType}",
			map[string]interface{}{"systemId": req.SystemID, "buildingType": req.BuildingType},
		)
		if existingQueueItem != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("Building of type %s already in queue for this system", req.BuildingType),
			})
		}


		// Create a new record in the building_queue
		queueCollection, err := app.Dao().FindCollectionByNameOrId("building_queue")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Building queue collection not found",
			})
		}

		currentTick := tick.GetCurrentTick(app)
		completionTick := currentTick + buildTimeTicks
		targetLevel := currentLevel + 1
		if req.BuildingType == "bank" { // Bank target level is always 1 (or conceptual)
			targetLevel = 1
		}

		queueRecord := models.NewRecord(queueCollection)
		queueRecord.Set("system_id", req.SystemID)
		queueRecord.Set("owner_id", user.Id)
		queueRecord.Set("building_type", req.BuildingType)
		queueRecord.Set("target_level", targetLevel)
		queueRecord.Set("completion_tick", completionTick)
		// PocketBase automatically adds "created"

		if err := app.Dao().SaveRecord(queueRecord); err != nil {
			// Attempt to refund resources if queue save fails
			if req.BuildingType == "bank" {
				user.Set("credits", user.GetInt("credits")+cost)
				app.Dao().SaveRecord(user) // Best effort refund
			} else {
				system.Set("goods", system.GetInt("goods")+cost)
				app.Dao().SaveRecord(system) // Best effort refund
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to add building to queue",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"message":         "Building construction queued",
			"building_type":   req.BuildingType,
			"target_level":    targetLevel,
			"cost":            cost,
			"completion_tick": completionTick,
			"queue_id":        queueRecord.Id,
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

// handleBankConstruction is now integrated into ApplyBuildingCompletions or not needed if bank creation is handled by queue logic
// func handleBankConstruction(app *pocketbase.PocketBase, c echo.Context, user *models.Record, system *models.Record, buildingType string) error {
// ... (original content commented out or removed as it's being replaced by the queue system)
// }