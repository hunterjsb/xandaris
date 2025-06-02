package pkg

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"

	"github.com/hunterjsb/xandaris/internal/credits"
	"github.com/hunterjsb/xandaris/internal/tick"
)

// RegisterAPIRoutes sets up all game API endpoints
func RegisterAPIRoutes(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Test endpoint
		e.Router.GET("/api/test", func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "API routes working!"})
		})

		// Game data endpoints
		e.Router.GET("/api/map", getMapData(app))
		e.Router.GET("/api/systems", getSystems(app))
		e.Router.GET("/api/systems/:id", getSystem(app))
		e.Router.GET("/api/planets", getPlanets(app))
		e.Router.GET("/api/buildings", getBuildings(app))
		e.Router.GET("/api/fleets", getFleets(app))
		e.Router.GET("/api/trade_routes", getTradeRoutes(app))
		e.Router.GET("/api/treaties", getTreaties(app))
		e.Router.GET("/api/hyperlanes", getHyperlanes(app))

		// Game actions
		e.Router.POST("/api/orders/fleet", sendFleet(app), apis.ActivityLogger(app), apis.RequireAdminOrRecordAuth())
		e.Router.POST("/api/orders/multi-fleet", sendMultiFleet(app), apis.ActivityLogger(app), apis.RequireAdminOrRecordAuth())
		e.Router.POST("/api/orders/build", queueBuilding(app), apis.ActivityLogger(app), apis.RequireAdminOrRecordAuth())
		e.Router.POST("/api/orders/trade", createTradeRoute(app), apis.ActivityLogger(app), apis.RequireAdminOrRecordAuth())
		e.Router.POST("/api/orders/colonize", colonizePlanet(app), apis.ActivityLogger(app), apis.RequireAdminOrRecordAuth())
		e.Router.POST("/api/diplomacy", proposeTreaty(app), apis.ActivityLogger(app), apis.RequireAdminOrRecordAuth())

		// Status endpoint
		e.Router.GET("/api/status", getStatus(app))

		// User resources endpoint
		e.Router.GET("/api/user/resources", getUserResources(app))

		// Building and Resource Types
		e.Router.GET("/api/building_types", getBuildingTypes(app))
		e.Router.GET("/api/resource_types", getResourceTypes(app))

		// Worldgen endpoints
		e.Router.GET("/api/worldgen", getWorldgen(app))
		e.Router.GET("/api/worldgen/:seed", getWorldgenWithSeed(app))

		return nil
	})
}

// MapData represents the frontend-expected map structure
type MapData struct {
	Systems []SystemData `json:"systems"`
	Planets []PlanetData `json:"planets"`
	Lanes   []LaneData   `json:"lanes"`
}

type SystemData struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	OwnerID  string `json:"owner_id"`
	Richness int    `json:"richness"`
}

type PlanetData struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	SystemID      string                 `json:"system_id"`
	PlanetType    string                 `json:"type"`
	Size          int                    `json:"size"`
	Population    int                    `json:"population"`
	MaxPopulation int                    `json:"max_population"`
	ColonizedBy   string                 `json:"colonized_by"`
	ColonizedAt   string                 `json:"colonized_at"`
	Pop           int                    `json:"pop"`
	Morale        int                    `json:"morale"`
	Food          int                    `json:"food"`
	Ore           int                    `json:"ore"`
	Goods         int                    `json:"goods"`
	Fuel          int                    `json:"fuel"`
	Credits       int                    `json:"credits"`
	Buildings     map[string]int         `json:"buildings"`
	Resources     map[string]interface{} `json:"resources"`
}

type BuildingData struct {
	ID             string `json:"id"`
	PlanetID       string `json:"planet_id"`
	SystemID       string `json:"system_id"`
	OwnerID        string `json:"owner_id"`
	Type           string `json:"type"`
	Name           string `json:"name"`
	Level          int    `json:"level"`
	Active         bool   `json:"active"`
	CreditsPerTick int    `json:"credits_per_tick"`
	SystemName     string `json:"system_name,omitempty"`
}

type FleetData struct {
	ID       string `json:"id"`
	OwnerID  string `json:"owner_id"`
	Name     string `json:"name"`
	FromID   string `json:"from_id"`
	ToID     string `json:"to_id"`
	ETA      string `json:"eta"`
	ETATick  int    `json:"eta_tick"`
	Strength int    `json:"strength"`
}

type TradeRouteData struct {
	ID       string `json:"id"`
	OwnerID  string `json:"owner_id"`
	FromID   string `json:"from_id"`
	ToID     string `json:"to_id"`
	Cargo    string `json:"cargo"`
	Capacity int    `json:"capacity"`
	ETA      string `json:"eta"`
	ETATick  int    `json:"eta_tick"`
}

type TreatyData struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	AID       string `json:"a_id"`
	BID       string `json:"b_id"`
	ExpiresAt string `json:"expires_at"`
	Status    string `json:"status"`
}

type LaneData struct {
	From     string `json:"from"`
	To       string `json:"to"`
	FromX    int    `json:"fromX"`
	FromY    int    `json:"fromY"`
	ToX      int    `json:"toX"`
	ToY      int    `json:"toY"`
	Distance int    `json:"distance"`
}

// BuildingTypeData represents the data structure for building types
type BuildingTypeData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Cost        int    `json:"cost"`
	Description string `json:"description"`
	Category    string `json:"category"`
	BuildTime   int    `json:"build_time"`
	Icon        string `json:"icon"`
	MaxLevel    int    `json:"max_level"`
}

// ResourceTypeData represents the data structure for resource types
type ResourceTypeData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// getMapData returns the complete map with systems, planets, and lanes
func getMapData(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get all systems
		systems, err := app.Dao().FindRecordsByExpr("systems", nil, nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch systems"})
		}

		// Get Null planet type ID once
		nullPlanetTypes, err := app.Dao().FindRecordsByFilter("planet_types", "name = 'Null'", "", 1, 0)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch planet types"})
		}
		
		var nullPlanetTypeID string
		if len(nullPlanetTypes) > 0 {
			nullPlanetTypeID = nullPlanetTypes[0].Id
		}

		// Get all non-Null planets efficiently
		var planets []*models.Record
		if nullPlanetTypeID != "" {
			planets, err = app.Dao().FindRecordsByFilter("planets", "planet_type != '"+nullPlanetTypeID+"'", "", 0, 0)
		} else {
			planets, err = app.Dao().FindRecordsByExpr("planets", nil, nil)
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch planets"})
		}

		// Transform systems data
		systemsData := make([]SystemData, len(systems))
		for i, system := range systems {
			systemsData[i] = SystemData{
				ID:       system.Id,
				Name:     system.GetString("name"),
				X:        system.GetInt("x"),
				Y:        system.GetInt("y"),
				OwnerID:  system.GetString("owner_id"),
				Richness: system.GetInt("richness"),
			}
		}

		// Transform planets data
		planetsData := make([]PlanetData, len(planets))
		for i, planet := range planets {
			// Calculate aggregated data for each planet
			totalPop := planet.GetInt("population") // Start with the planet's own population
			totalFood := 0
			totalOre := 0
			totalGoods := 0
			totalFuel := 0
			totalCredits := 0
			buildingCounts := make(map[string]int)

			// Get buildings for this planet
			buildings, _ := app.Dao().FindRecordsByFilter("buildings", fmt.Sprintf("planet_id='%s'", planet.Id), "", 0, 0)
			for _, building := range buildings {
				buildingType := building.GetString("building_type")
				buildingCounts[buildingType]++

				// Calculate building production based on type
				switch buildingType {
				case "farm":
					totalFood += building.GetInt("level") * 10
				case "mine":
					totalOre += building.GetInt("level") * 8
				case "factory":
					totalGoods += building.GetInt("level") * 6
				case "refinery":
					totalFuel += building.GetInt("level") * 5
				case "bank":
					totalCredits += building.GetInt("level") * 1
				}
			}

			planetsData[i] = PlanetData{
				ID:            planet.Id,
				Name:          planet.GetString("name"),
				SystemID:      planet.GetString("system_id"),
				PlanetType:    planet.GetString("planet_type"),
				Size:          planet.GetInt("size"),
				Population:    planet.GetInt("population"), // This is base population, totalPop includes this.
				MaxPopulation: planet.GetInt("max_population"),
				ColonizedBy:   planet.GetString("colonized_by"),
				ColonizedAt:   planet.GetString("colonized_at"),
				Pop:           totalPop, // This is the aggregated population for the planet
				Morale:        75,       // Default morale for planet
				Food:          totalFood,
				Ore:           totalOre,
				Goods:         totalGoods,
				Fuel:          totalFuel,
				Credits:       totalCredits,
				Buildings:     buildingCounts,
				Resources: map[string]interface{}{
					"food":  totalFood,
					"ore":   totalOre,
					"goods": totalGoods,
					"fuel":  totalFuel,
				},
			}
		}

		// Generate lanes between nearby systems
		lanes := generateLanes(systemsData)

		mapData := MapData{
			Systems: systemsData,
			Planets: planetsData,
			Lanes:   lanes,
		}

		return c.JSON(http.StatusOK, mapData)
	}
}

// getBuildingTypes returns all building types
func getBuildingTypes(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		records, err := app.Dao().FindRecordsByExpr("building_types", nil, nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "Failed to fetch building types",
				"details": err.Error(),
			})
		}

		buildingTypesData := make([]BuildingTypeData, len(records))
		for i, record := range records {
			buildingTypesData[i] = BuildingTypeData{
				ID:          record.Id,
				Name:        record.GetString("name"),
				Cost:        record.GetInt("cost"),
				Description: record.GetString("description"),
				Category:    record.GetString("category"),
				BuildTime:   record.GetInt("build_time"),
				Icon:        record.GetString("icon"),
				MaxLevel:    record.GetInt("max_level"),
			}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"page":       1,
			"perPage":    len(buildingTypesData),
			"totalItems": len(buildingTypesData),
			"items":      buildingTypesData,
		})
	}
}

// getResourceTypes returns all resource types
func getResourceTypes(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		records, err := app.Dao().FindRecordsByExpr("resource_types", nil, nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "Failed to fetch resource types",
				"details": err.Error(),
			})
		}

		resourceTypesData := make([]ResourceTypeData, len(records))
		for i, record := range records {
			resourceTypesData[i] = ResourceTypeData{
				ID:   record.Id,
				Name: record.GetString("name"),
				Icon: record.GetString("icon"),
			}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"page":       1,
			"perPage":    len(resourceTypesData),
			"totalItems": len(resourceTypesData),
			"items":      resourceTypesData,
		})
	}
}

// getSystems returns systems data with frontend compatibility
func getSystems(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.QueryParam("user_id")

		var systems []*models.Record
		var err error

		if userID != "" {
			filter := fmt.Sprintf("owner_id='%s'", userID)
			systems, err = app.Dao().FindRecordsByFilter("systems", filter, "x,y", 0, 0)
		} else {
			systems, err = app.Dao().FindRecordsByExpr("systems", nil, nil)
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "Failed to fetch systems",
				"details": err.Error(),
			})
		}

		// Transform to frontend format
		systemsData := make([]SystemData, len(systems))
		for i, system := range systems {
			systemsData[i] = SystemData{
				ID:       system.Id,
				Name:     system.GetString("name"),
				X:        system.GetInt("x"),
				Y:        system.GetInt("y"),
				OwnerID:  system.GetString("owner_id"),
				Richness: system.GetInt("richness"),
			}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"page":       1,
			"perPage":    len(systemsData),
			"totalItems": len(systemsData),
			"items":      systemsData,
		})
	}
}

// getSystem returns a single system with detailed data
func getSystem(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		systemID := c.PathParam("id")

		system, err := app.Dao().FindRecordById("systems", systemID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "System not found"})
		}

		// Get Null planet type ID once
		nullPlanetTypes, _ := app.Dao().FindRecordsByFilter("planet_types", "name = 'Null'", "", 1, 0)
		
		// Get non-Null planets in this system efficiently
		var planets []*models.Record
		if len(nullPlanetTypes) > 0 {
			nullPlanetTypeID := nullPlanetTypes[0].Id
			planets, _ = app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("system_id='%s' AND planet_type != '%s'", systemID, nullPlanetTypeID), "", 0, 0)
		} else {
			planets, _ = app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("system_id='%s'", systemID), "", 0, 0)
		}

		planetsData := make([]PlanetData, len(planets))
		for i, planet := range planets {
			planetsData[i] = PlanetData{
				ID:            planet.Id,
				Name:          planet.GetString("name"),
				SystemID:      planet.GetString("system_id"),
				PlanetType:    planet.GetString("planet_type"),
				Size:          planet.GetInt("size"),
				Population:    planet.GetInt("population"),
				MaxPopulation: planet.GetInt("max_population"),
				ColonizedBy:   planet.GetString("colonized_by"),
				ColonizedAt:   planet.GetString("colonized_at"),
			}
		}

		systemData := SystemData{
			ID:       system.Id,
			Name:     system.GetString("name"),
			X:        system.GetInt("x"),
			Y:        system.GetInt("y"),
			OwnerID:  system.GetString("owner_id"),
			Richness: system.GetInt("richness"),
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"system":  systemData,
			"planets": planetsData,
		})
	}
}

// getBuildings returns buildings with frontend-compatible format
func getBuildings(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.QueryParam("owner_id")

		// Get all buildings
		buildings, err := app.Dao().FindRecordsByExpr("buildings", nil, nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch buildings"})
		}

		// Filter by user if needed
		if userID != "" {
			var filteredBuildings []*models.Record
			for _, buildingRecord := range buildings { // Renamed 'building' to 'buildingRecord'
				planet, err := app.Dao().FindRecordById("planets", buildingRecord.GetString("planet_id"))
				if err != nil {
					log.Printf("Warning: Planet %s for building %s not found during filtering: %v", buildingRecord.GetString("planet_id"), buildingRecord.Id, err)
					continue
				}
				// Only consider buildings on planets directly colonized by the userID
				if planet.GetString("colonized_by") == userID {
					filteredBuildings = append(filteredBuildings, buildingRecord)
				}
			}
			buildings = filteredBuildings // Update the main buildings slice
		}

		// Transform to frontend format
		buildingsData := make([]BuildingData, len(buildings))
		for i, building := range buildings { // This is the building record from the (potentially filtered) list
			// Get planet and system data
			planet, errPlanet := app.Dao().FindRecordById("planets", building.GetString("planet_id"))
			var systemID, ownerID, systemName string
			if errPlanet == nil && planet != nil {
				systemID = planet.GetString("system_id")
				// For OwnerID of the building, prioritize the planet's colonizer
				if planet.GetString("colonized_by") != "" {
					ownerID = planet.GetString("colonized_by")
				} else {
					// If planet is not colonized, owner might be derived from system (though less likely for owned buildings)
					if systemRecord, errSystem := app.Dao().FindRecordById("systems", systemID); errSystem == nil && systemRecord != nil {
						ownerID = systemRecord.GetString("owner_id")
					}
				}
				// Get system name
				if systemRecord, errSystem := app.Dao().FindRecordById("systems", systemID); errSystem == nil && systemRecord != nil {
					systemName = systemRecord.GetString("name")
				}
			} else if errPlanet != nil {
				log.Printf("Warning: Planet %s for building %s not found when creating BuildingData: %v", building.GetString("planet_id"), building.Id, errPlanet)
			}

			buildingTypeID := building.GetString("building_type")
			buildingTypeName := buildingTypeID // Fallback to ID if name not found
			isBank := false

			bt, errBT := app.Dao().FindRecordById("building_types", buildingTypeID)
			if errBT == nil && bt != nil {
				buildingTypeName = bt.GetString("name")
				// Assuming "Bank" is the exact name in the 'building_types' collection for bank buildings
				if buildingTypeName == "Bank" {
					isBank = true
				}
			} else if errBT != nil {
				log.Printf("Warning: Building type %s (ID: %s) for building %s not found when creating BuildingData: %v", buildingTypeName, buildingTypeID, building.Id, errBT)
			}

			creditsPerTick := 0
			if isBank {
				creditsPerTick = building.GetInt("level") // Or some other formula based on bt properties if available
			}

			buildingsData[i] = BuildingData{
				ID:             building.Id,
				PlanetID:       building.GetString("planet_id"),
				SystemID:       systemID,
				OwnerID:        ownerID,
				SystemName:     systemName,
				Type:           buildingTypeID,                                                         // Store the ID of the building type
				Name:           fmt.Sprintf("%s Level %d", buildingTypeName, building.GetInt("level")), // Use fetched name
				Level:          building.GetInt("level"),
				Active:         building.GetBool("active"),
				CreditsPerTick: creditsPerTick,
			}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"page":       1,
			"perPage":    len(buildingsData),
			"totalItems": len(buildingsData),
			"items":      buildingsData,
		})
	}
}

// getFleets returns fleet data in frontend format
func getFleets(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.QueryParam("owner_id")

		var fleets []*models.Record
		var err error

		if userID != "" {
			filter := fmt.Sprintf("owner_id='%s'", userID)
			fleets, err = app.Dao().FindRecordsByFilter("fleets", filter, "eta", 0, 0)
		} else {
			fleets, err = app.Dao().FindRecordsByExpr("fleets", nil, nil)
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch fleets"})
		}

		fleetsData := make([]map[string]interface{}, len(fleets))
		for i, fleet := range fleets {
			// Get ships in this fleet
			ships, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id='%s'", fleet.Id), "", 0, 0)
			if err != nil {
				ships = []*models.Record{} // Empty if error
			}

			// Process ship data
			shipData := make([]map[string]interface{}, len(ships))
			for j, ship := range ships {
				// Get ship type details
				shipType, err := app.Dao().FindRecordById("ship_types", ship.GetString("ship_type"))
				var shipTypeName string
				if err == nil {
					shipTypeName = shipType.GetString("name")
				} else {
					shipTypeName = "unknown"
				}

				shipData[j] = map[string]interface{}{
					"id":             ship.Id,
					"ship_type":      ship.GetString("ship_type"),
					"ship_type_name": shipTypeName,
					"count":          ship.GetInt("count"),
					"health":         ship.GetFloat("health"),
				}
			}

			fleetsData[i] = map[string]interface{}{
				"id":                 fleet.Id,
				"owner_id":           fleet.GetString("owner_id"),
				"name":               fleet.GetString("name"),
				"current_system":     fleet.GetString("current_system"),
				"destination_system": fleet.GetString("destination_system"),
				"eta":                fleet.GetDateTime("eta").String(),
				"ships":              shipData,
			}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"page":       1,
			"perPage":    len(fleetsData),
			"totalItems": len(fleetsData),
			"items":      fleetsData,
		})
	}
}

// getTradeRoutes returns trade routes in frontend format
func getTradeRoutes(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.QueryParam("owner_id")

		var routes []*models.Record
		var err error

		if userID != "" {
			filter := fmt.Sprintf("owner_id='%s'", userID)
			routes, err = app.Dao().FindRecordsByFilter("trade_routes", filter, "eta", 0, 0)
		} else {
			routes, err = app.Dao().FindRecordsByExpr("trade_routes", nil, nil)
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch trade routes"})
		}

		routesData := make([]TradeRouteData, len(routes))
		for i, route := range routes {
			routesData[i] = TradeRouteData{
				ID:       route.Id,
				OwnerID:  route.GetString("owner_id"),
				FromID:   route.GetString("from_id"),
				ToID:     route.GetString("to_id"),
				Cargo:    route.GetString("cargo"),
				Capacity: route.GetInt("capacity"),
				ETA:      route.GetDateTime("eta").String(),
				ETATick:  route.GetInt("eta_tick"),
			}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"page":       1,
			"perPage":    len(routesData),
			"totalItems": len(routesData),
			"items":      routesData,
		})
	}
}

// getTreaties returns treaties in frontend format
func getTreaties(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.QueryParam("user_id")

		var treaties []*models.Record
		var err error

		if userID != "" {
			filter := fmt.Sprintf("a_id='%s' || b_id='%s'", userID, userID)
			treaties, err = app.Dao().FindRecordsByFilter("treaties", filter, "-created", 0, 0)
		} else {
			treaties, err = app.Dao().FindRecordsByExpr("treaties", nil, nil)
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch treaties"})
		}

		treatiesData := make([]TreatyData, len(treaties))
		for i, treaty := range treaties {
			treatiesData[i] = TreatyData{
				ID:        treaty.Id,
				Type:      treaty.GetString("type"),
				AID:       treaty.GetString("a_id"),
				BID:       treaty.GetString("b_id"),
				ExpiresAt: treaty.GetDateTime("expires_at").String(),
				Status:    treaty.GetString("status"),
			}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"page":       1,
			"perPage":    len(treatiesData),
			"totalItems": len(treatiesData),
			"items":      treatiesData,
		})
	}
}

// Action endpoints
func sendFleet(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := struct {
			FromID   string `json:"from_id"`
			ToID     string `json:"to_id"`
			FleetID  string `json:"fleet_id"`
			Strength int    `json:"strength"`
		}{}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		var fleet *models.Record
		var err error

		if data.FleetID != "" {
			// Use specific fleet if provided
			fleet, err = app.Dao().FindRecordById("fleets", data.FleetID)
			if err != nil {
				return apis.NewBadRequestError("Fleet not found", err)
			}
			
			// Verify ownership
			if fleet.GetString("owner_id") != user.Id {
				return apis.NewForbiddenError("You don't own this fleet", nil)
			}
			
			// Verify fleet is at source system
			if fleet.GetString("current_system") != data.FromID {
				return apis.NewBadRequestError("Fleet is not at source system", nil)
			}
		} else {
			// Find an existing fleet at the source system
			fleetFilter := fmt.Sprintf("owner_id='%s' && current_system='%s'", user.Id, data.FromID)
			fmt.Printf("DEBUG Fleet Filter: %s\n", fleetFilter)
			fleets, err := app.Dao().FindRecordsByFilter("fleets", fleetFilter, "", 1, 0)
			if err != nil {
				fmt.Printf("DEBUG Fleet Filter Error: %v\n", err)
				return apis.NewBadRequestError("Failed to find fleets", err)
			}
			fmt.Printf("DEBUG Found %d fleets\n", len(fleets))

			if len(fleets) == 0 {
				return apis.NewBadRequestError("No available fleets at source system", nil)
			}

			// Use the first available fleet
			fleet = fleets[0]
		}

		// Check if fleet already has pending orders
		existingOrders, err := app.Dao().FindRecordsByFilter(
			"fleet_orders", 
			fmt.Sprintf("fleet_id='%s' && (status='pending' || status='processing')", fleet.Id),
			"", 1, 0,
		)
		if err == nil && len(existingOrders) > 0 {
			return apis.NewBadRequestError("Fleet already has pending orders", nil)
		}

		// Validate hyperlane range (same as navigation system - 800 units max)
		fromSystem, err := app.Dao().FindRecordById("systems", data.FromID)
		if err != nil {
			return apis.NewBadRequestError("Source system not found", err)
		}
		toSystem, err := app.Dao().FindRecordById("systems", data.ToID)
		if err != nil {
			return apis.NewBadRequestError("Target system not found", err)
		}

		deltaX := toSystem.GetFloat("x") - fromSystem.GetFloat("x")
		deltaY := toSystem.GetFloat("y") - fromSystem.GetFloat("y")
		distance := math.Sqrt(deltaX*deltaX + deltaY*deltaY)

		if distance > 800 {
			return apis.NewBadRequestError("Target system too far - outside hyperlane range", nil)
		}

		// Get Fleet Orders collection
		fleetOrdersCollection, err := app.Dao().FindCollectionByNameOrId("fleet_orders")
		if err != nil {
			log.Printf("Error finding fleet_orders collection: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Fleet Orders collection not found", err)
		}

		// Calculate execute_at_tick
		currentTick := tick.GetCurrentTick(app)
		// Travel duration is 2 ticks (20 seconds at 10 seconds/tick) for faster testing
		travelDurationInTicks := int64(2)
		executeAtTick := currentTick + travelDurationInTicks

		// Create a new fleet_order record
		order := models.NewRecord(fleetOrdersCollection)
		order.Set("user_id", user.Id)
		order.Set("fleet_id", fleet.Id)
		order.Set("type", "move") // Type is "move" for fleet_orders
		order.Set("status", "pending")
		order.Set("execute_at_tick", executeAtTick)
		order.Set("destination_system_id", data.ToID)
		order.Set("original_system_id", data.FromID)
		order.Set("travel_time_ticks", travelDurationInTicks)

		if err := app.Dao().SaveRecord(order); err != nil {
			log.Printf("Error saving fleet order: %v", err)
			return apis.NewBadRequestError("Failed to create fleet move order", err)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":  true,
			"order_id": order.Id,
		})
	}
}

func sendMultiFleet(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := struct {
			FleetID   string `json:"fleet_id"`
			NextStop  string `json:"next_stop"`    // Immediate next system
			TravelSec int    `json:"travel_sec"`   // Travel time in seconds
		}{}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Validate fleet ownership and availability
		fleet, err := app.Dao().FindRecordById("fleets", data.FleetID)
		if err != nil {
			return apis.NewBadRequestError("Fleet not found", err)
		}

		if fleet.GetString("owner_id") != user.Id {
			return apis.NewForbiddenError("You don't own this fleet", nil)
		}

		// Check if fleet is already moving
		if fleet.GetString("destination_system") != "" {
			return apis.NewBadRequestError("Fleet is already in transit", nil)
		}

		// Validate next_stop exists
		if data.NextStop == "" {
			return apis.NewBadRequestError("Next stop system ID required", nil)
		}

		// Calculate ETA (use provided time or default to 120 seconds)
		travelTime := data.TravelSec
		if travelTime <= 0 {
			travelTime = 120
		}
		etaTime := time.Now().Add(time.Duration(travelTime) * time.Second)

		// Update fleet with next hop
		fleet.Set("next_stop", data.NextStop)
		fleet.Set("destination_system", data.NextStop) // For tracking purposes
		fleet.Set("eta", etaTime)

		if err := app.Dao().SaveRecord(fleet); err != nil {
			return apis.NewBadRequestError("Failed to start fleet movement", err)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":   true,
			"fleet_id":  fleet.Id,
			"next_stop": data.NextStop,
			"eta":       etaTime,
		})
	}
}

func getHyperlanes(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		hyperlanes, err := app.Dao().FindRecordsByFilter("hyperlanes", "id != ''", "", 0, 0)
		if err != nil {
			return apis.NewBadRequestError("Failed to fetch hyperlanes", err)
		}

		// Convert to simple format
		result := make([]map[string]interface{}, len(hyperlanes))
		for i, hyperlane := range hyperlanes {
			result[i] = map[string]interface{}{
				"id":          hyperlane.Id,
				"from_system": hyperlane.GetString("from_system"),
				"to_system":   hyperlane.GetString("to_system"),
				"distance":    hyperlane.GetFloat("distance"),
			}
		}

		return c.JSON(http.StatusOK, result)
	}
}

func queueBuilding(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := struct {
			PlanetID     string `json:"planet_id"` // Changed from SystemID
			BuildingType string `json:"building_type"`
		}{}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Verify Planet Ownership/Validity
		targetPlanet, err := app.Dao().FindRecordById("planets", data.PlanetID)
		if err != nil {
			return apis.NewNotFoundError("Planet not found.", err)
		}
		if targetPlanet.GetString("colonized_by") != user.Id {
			return apis.NewForbiddenError("You do not own this planet and cannot build on it.", nil)
		}

		// Get building type to check cost
		buildingTypeRecord, err := app.Dao().FindRecordById("building_types", data.BuildingType)
		if err != nil {
			return apis.NewBadRequestError(fmt.Sprintf("Building type %s not found", data.BuildingType), err)
		}

		// Check building cost and deduct resources
		costRaw := buildingTypeRecord.Get("cost")
		originalCostPayload := buildingTypeRecord.Get("cost")

		switch costValue := costRaw.(type) {
		case int64:
			cost := int(costValue)
			hasCredits, err := credits.HasSufficientCredits(app, user.Id, cost)
			if err != nil {
				return apis.NewBadRequestError("Failed to check credits", err)
			}
			if !hasCredits {
				userCredits, _ := credits.GetUserCredits(app, user.Id)
				return apis.NewBadRequestError(fmt.Sprintf("Insufficient credits. Need %d, have %d", cost, userCredits), nil)
			}
			if err := credits.DeductUserCredits(app, user.Id, cost); err != nil {
				return apis.NewBadRequestError("Failed to deduct credits", err)
			}
		case float64:
			cost := int(costValue)
			hasCredits, err := credits.HasSufficientCredits(app, user.Id, cost)
			if err != nil {
				return apis.NewBadRequestError("Failed to check credits", err)
			}
			if !hasCredits {
				userCredits, _ := credits.GetUserCredits(app, user.Id)
				return apis.NewBadRequestError(fmt.Sprintf("Insufficient credits. Need %d, have %d", cost, userCredits), nil)
			}
			if err := credits.DeductUserCredits(app, user.Id, cost); err != nil {
				return apis.NewBadRequestError("Failed to deduct credits", err)
			}
		case map[string]interface{}:
			costMap := costValue
			for resourceId, amountInterface := range costMap {
				amount, ok := amountInterface.(float64)
				if !ok {
					return apis.NewBadRequestError(fmt.Sprintf("Invalid amount type for resource %s in cost", resourceId), nil)
				}
				amountInt := int(amount)

				// Get resource name from ID
				resourceType, err := app.Dao().FindRecordById("resource_types", resourceId)
				if err != nil {
					return apis.NewBadRequestError(fmt.Sprintf("Invalid resource ID %s", resourceId), err)
				}
				resourceName := resourceType.GetString("name")

				// For now, only handle credits - other resources still use legacy system
				if resourceName == "credits" {
					hasCredits, err := credits.HasSufficientCredits(app, user.Id, amountInt)
					if err != nil {
						return apis.NewBadRequestError("Failed to check credits", err)
					}
					if !hasCredits {
						currentCredits, _ := credits.GetUserCredits(app, user.Id)
						return apis.NewBadRequestError(fmt.Sprintf("Insufficient credits. Need %d, have %d", amountInt, currentCredits), nil)
					}
					if err := credits.DeductUserCredits(app, user.Id, amountInt); err != nil {
						return apis.NewBadRequestError("Failed to deduct credits", err)
					}
				} else {
					// Legacy system for other resources
					currentResourceValue := user.GetInt(resourceName)
					if currentResourceValue < amountInt {
						return apis.NewBadRequestError(fmt.Sprintf("Insufficient %s. Need %d, have %d", resourceName, amountInt, currentResourceValue), nil)
					}
					user.Set(resourceName, currentResourceValue-amountInt)
					if err := app.Dao().SaveRecord(user); err != nil {
						return apis.NewBadRequestError("Failed to update user resources", err)
					}
				}
			}
		default:
			return apis.NewBadRequestError(fmt.Sprintf("Unsupported cost type: %T", costRaw), nil)
		}

		// Create building record directly
		buildingsCollection, err := app.Dao().FindCollectionByNameOrId("buildings")
		if err != nil {
			log.Printf("Error finding buildings collection: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Buildings collection not found", err)
		}

		building := models.NewRecord(buildingsCollection)
		building.Set("planet_id", data.PlanetID)
		building.Set("building_type", data.BuildingType) // This is the ID of the building type
		building.Set("level", 1)
		building.Set("active", true)
		// TODO: Set owner_id if your buildings schema requires it and it's not automatically handled
		// Example: building.Set("owner_id", user.Id) if buildings are directly owned by users

		if err := app.Dao().SaveRecord(building); err != nil {
			log.Printf("Error saving new building: %v", err)
			return apis.NewBadRequestError("Failed to create building", err)
		}

		// Get current credits for response
		currentCredits, _ := credits.GetUserCredits(app, user.Id)

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":           true,
			"building_id":       building.Id, // Return building_id instead of order_id
			"cost":              originalCostPayload,
			"credits_remaining": currentCredits,
		})
	}
}

func createTradeRoute(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := struct {
			FromID   string `json:"from_id"`
			ToID     string `json:"to_id"`
			Cargo    string `json:"cargo"`
			Capacity int    `json:"capacity"`
		}{}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Create trade route record
		collection, err := app.Dao().FindCollectionByNameOrId("trade_routes")
		if err != nil {
			return apis.NewBadRequestError("Trade routes collection not found", err)
		}

		route := models.NewRecord(collection)
		route.Set("owner_id", user.Id)
		route.Set("from_id", data.FromID)
		route.Set("to_id", data.ToID)
		route.Set("cargo", data.Cargo)
		route.Set("capacity", data.Capacity)
		route.Set("eta_tick", 6) // 1 hour = 6 ticks

		if err := app.Dao().SaveRecord(route); err != nil {
			return apis.NewBadRequestError("Failed to create trade route", err)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":  true,
			"route_id": route.Id,
		})
	}
}

func proposeTreaty(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := struct {
			PlayerID string `json:"player_id"`
			Type     string `json:"type"`
			Terms    string `json:"terms"`
		}{}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Create treaty record
		collection, err := app.Dao().FindCollectionByNameOrId("treaties")
		if err != nil {
			return apis.NewBadRequestError("Treaties collection not found", err)
		}

		treaty := models.NewRecord(collection)
		treaty.Set("type", data.Type)
		treaty.Set("a_id", user.Id)
		treaty.Set("b_id", data.PlayerID)
		treaty.Set("status", "proposed")

		if err := app.Dao().SaveRecord(treaty); err != nil {
			return apis.NewBadRequestError("Failed to create treaty", err)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":   true,
			"treaty_id": treaty.Id,
		})
	}
}

func colonizePlanet(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := struct {
			PlanetID string `json:"planet_id"`
			FleetID  string `json:"fleet_id"`
		}{}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Get the planet
		planet, err := app.Dao().FindRecordById("planets", data.PlanetID)
		if err != nil {
			return apis.NewBadRequestError("Planet not found", err)
		}

		// Check if planet is already colonized
		if planet.GetString("colonized_by") != "" {
			return apis.NewBadRequestError("Planet is already colonized", nil)
		}

		// Get the fleet
		fleet, err := app.Dao().FindRecordById("fleets", data.FleetID)
		if err != nil {
			return apis.NewBadRequestError("Fleet not found", err)
		}

		// Verify fleet ownership
		if fleet.GetString("owner_id") != user.Id {
			return apis.NewUnauthorizedError("You don't own this fleet", nil)
		}

		// Get the system the planet is in
		system, err := app.Dao().FindRecordById("systems", planet.GetString("system_id"))
		if err != nil {
			return apis.NewBadRequestError("System not found", err)
		}

		// Check if fleet is at the same system as the planet
		if fleet.GetString("current_system") != system.Id {
			return apis.NewBadRequestError("Fleet must be at the same system as the planet", nil)
		}

		// Find settler ship in the fleet
		settlerShipType, err := app.Dao().FindFirstRecordByFilter("ship_types", "name='settler'")
		if err != nil {
			return apis.NewBadRequestError("Settler ship type not found", err)
		}

		settlerShip, err := app.Dao().FindFirstRecordByFilter("ships", 
			fmt.Sprintf("fleet_id='%s' && ship_type='%s' && count > 0", fleet.Id, settlerShipType.Id))
		if err != nil {
			return apis.NewBadRequestError("No settler ships found in this fleet", nil)
		}

		// Consume one settler ship
		currentCount := settlerShip.GetInt("count")
		if currentCount <= 1 {
			// Delete the ship record if it's the last one
			if err := app.Dao().DeleteRecord(settlerShip); err != nil {
				return apis.NewBadRequestError("Failed to consume settler ship", err)
			}
		} else {
			// Decrease count by 1
			settlerShip.Set("count", currentCount-1)
			if err := app.Dao().SaveRecord(settlerShip); err != nil {
				return apis.NewBadRequestError("Failed to consume settler ship", err)
			}
		}

		// Check if fleet has any remaining ships, delete if empty
		remainingShips, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id='%s'", fleet.Id), "", 0, 0)
		if err == nil && len(remainingShips) == 0 {
			// Fleet is now empty, delete it
			if err := app.Dao().DeleteRecord(fleet); err != nil {
				log.Printf("Warning: Failed to delete empty fleet %s: %v", fleet.Id, err)
			} else {
				log.Printf("Deleted empty fleet %s after consuming last ship", fleet.Id)
			}
		}

		// Set colonization data
		planet.Set("colonized_by", user.Id)
		planet.Set("colonized_at", time.Now())

		if err := app.Dao().SaveRecord(planet); err != nil {
			return apis.NewBadRequestError("Failed to colonize planet", err)
		}

		// Create initial population
		if err := createInitialPopulation(app, planet, user.Id); err != nil {
			return apis.NewBadRequestError("Failed to create initial population", err)
		}

		// Create initial buildings (optional)
		if err := createInitialBuildings(app, planet); err != nil {
			// Don't fail colonization if buildings fail, just log it
			log.Printf("Warning: Failed to create initial buildings for planet %s: %v", planet.Id, err)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":   true,
			"planet_id": planet.Id,
			"fleet_id":  fleet.Id,
			"message":   "Planet colonized successfully using settler ship",
		})
	}
}

func colonizeWithShip(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := struct {
			PlanetID string `json:"planet_id"`
			FleetID  string `json:"fleet_id"`
		}{}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Feature not implemented"})
	}
}

func createInitialPopulation(app *pocketbase.PocketBase, planet *models.Record, ownerID string) error {
	populationCollection, err := app.Dao().FindCollectionByNameOrId("populations")
	if err != nil {
		return err
	}

	population := models.NewRecord(populationCollection)
	population.Set("owner_id", ownerID)
	population.Set("planet_id", planet.Id)
	population.Set("count", 100)    // Start with 100 population
	population.Set("happiness", 80) // Start with 80% happiness

	return app.Dao().SaveRecord(population)
}

func createInitialBuildings(app *pocketbase.PocketBase, planet *models.Record) error {
	// Check if this is the user's first colony by checking if they have any other colonies
	ownerID := planet.GetString("colonized_by")
	existingColonies, err := app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("colonized_by = '%s'", ownerID), "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to check existing colonies: %w", err)
	}

	isFirstColony := len(existingColonies) <= 1 // 1 because we just created this colony

	buildingCollection, err := app.Dao().FindCollectionByNameOrId("buildings")
	if err != nil {
		return err
	}

	if isFirstColony {
		// For first colony, create a starter crypto_server with credits
		cryptoServerType, err := app.Dao().FindFirstRecordByFilter("building_types", "name = 'crypto_server'")
		if err != nil {
			return fmt.Errorf("crypto_server building type not found: %w", err)
		}

		building := models.NewRecord(buildingCollection)
		building.Set("planet_id", planet.Id)
		building.Set("building_type", cryptoServerType.Id)
		building.Set("level", 1)
		building.Set("active", true)

		// Crypto servers start empty - they generate credits over time

		return app.Dao().SaveRecord(building)
	} else {
		// For subsequent colonies, create a basic base building
		baseType, err := app.Dao().FindFirstRecordByFilter("building_types", "name = 'base'")
		if err != nil {
			return fmt.Errorf("base building type not found: %w", err)
		}

		building := models.NewRecord(buildingCollection)
		building.Set("planet_id", planet.Id)
		building.Set("building_type", baseType.Id)
		building.Set("level", 1)
		building.Set("active", true)

		return app.Dao().SaveRecord(building)
	}
}

func getStatus(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"current_tick":     tick.GetCurrentTick(app),
			"ticks_per_minute": tick.GetTickRate(),
			"server_time":      time.Now().Format(time.RFC3339),
		})
	}
}

func getUserResources(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Get credits from crypto_server buildings
		userCredits, err := credits.GetUserCredits(app, user.Id)
		if err != nil {
			log.Printf("Failed to get credits for user %s: %v", user.Id, err)
			userCredits = 0
		}

		// Get other resources from user record (legacy system)
		resourceData := map[string]interface{}{
			"credits": userCredits,
			"food":     user.GetInt("food"),
			"ore":      user.GetInt("ore"),
			"fuel":     user.GetInt("fuel"),
			"metal":    user.GetInt("metal"),
			"oil":      user.GetInt("oil"),
			"titanium": user.GetInt("titanium"),
			"xanium":   user.GetInt("xanium"),
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"user_id":   user.Id,
			"resources": resourceData,
		})
	}
}

// Helper functions
func generateLanes(systems []SystemData) []LaneData {
	lanes := make([]LaneData, 0)
	minDistance := 200.0 // Minimum distance to avoid too many close connections
	maxDistance := 650.0 // Maximum distance for lane connections (scaled for larger galaxy)

	systemConnections := make(map[string]int) // Track connections per system
	connected := make(map[string]bool)        // Track which systems are in main component
	
	// Initialize connection tracking
	for _, sys := range systems {
		systemConnections[sys.ID] = 0
		connected[sys.ID] = false
	}

	// Phase 1: Build minimum spanning tree to ensure connectivity
	// Start with the center-most system
	centerX := 0.0
	centerY := 0.0
	for _, sys := range systems {
		centerX += float64(sys.X)
		centerY += float64(sys.Y)
	}
	centerX /= float64(len(systems))
	centerY /= float64(len(systems))
	
	// Find system closest to center as starting point
	var startSystem SystemData
	minDistFromCenter := math.Inf(1)
	for _, sys := range systems {
		x := float64(sys.X)
		y := float64(sys.Y)
		dist := math.Sqrt((x-centerX)*(x-centerX) + (y-centerY)*(y-centerY))
		if dist < minDistFromCenter {
			minDistFromCenter = dist
			startSystem = sys
		}
	}
	
	connected[startSystem.ID] = true
	connectedSystems := []SystemData{startSystem}
	
	// Prim's algorithm for MST
	for len(connectedSystems) < len(systems) {
		var bestConnection struct {
			from SystemData
			to   SystemData
			dist float64
		}
		bestConnection.dist = math.Inf(1)
		
		// Find shortest edge from connected to unconnected
		for _, connectedSys := range connectedSystems {
			for _, sys := range systems {
				if connected[sys.ID] {
					continue
				}
				
				dx := float64(sys.X - connectedSys.X)
				dy := float64(sys.Y - connectedSys.Y)
				dist := math.Sqrt(dx*dx + dy*dy)
				
				if dist < bestConnection.dist && dist <= maxDistance {
					bestConnection.from = connectedSys
					bestConnection.to = sys
					bestConnection.dist = dist
				}
			}
		}
		
		// Add best connection if found
		if bestConnection.dist < math.Inf(1) {
			lanes = append(lanes, LaneData{
				From:     bestConnection.from.ID,
				To:       bestConnection.to.ID,
				FromX:    bestConnection.from.X,
				FromY:    bestConnection.from.Y,
				ToX:      bestConnection.to.X,
				ToY:      bestConnection.to.Y,
				Distance: int(bestConnection.dist),
			})
			
			connected[bestConnection.to.ID] = true
			connectedSystems = append(connectedSystems, bestConnection.to)
			systemConnections[bestConnection.from.ID]++
			systemConnections[bestConnection.to.ID]++
		} else {
			break // No more reachable systems
		}
	}
	
	// Phase 2: Add inter-branch connections to link different spiral arms
	// Find systems that could be on different branches by analyzing their angular position
	centerX = 0.0
	centerY = 0.0
	for _, sys := range systems {
		centerX += float64(sys.X)
		centerY += float64(sys.Y)
	}
	centerX /= float64(len(systems))
	centerY /= float64(len(systems))
	
	// Group systems by angular sectors around the galaxy center
	type SystemWithAngle struct {
		system SystemData
		angle  float64
		radius float64
	}
	
	systemAngles := make([]SystemWithAngle, len(systems))
	for i, sys := range systems {
		dx := float64(sys.X) - centerX
		dy := float64(sys.Y) - centerY
		angle := math.Atan2(dy, dx)
		radius := math.Sqrt(dx*dx + dy*dy)
		systemAngles[i] = SystemWithAngle{sys, angle, radius}
	}
	
	// Create inter-branch connections between systems in different angular sectors
	interBranchConnections := 0
	maxInterBranch := len(systems) / 6 // Conservative number of cross-connections
	
	for i, sys1 := range systemAngles {
		if interBranchConnections >= maxInterBranch || systemConnections[sys1.system.ID] >= 3 {
			continue
		}
		
		// Look for systems in different angular sectors at similar radius
		for j, sys2 := range systemAngles {
			if i == j || systemConnections[sys2.system.ID] >= 3 {
				continue
			}
			
			// Check if systems are in different branches (angular separation)
			angleDiff := math.Abs(sys1.angle - sys2.angle)
			if angleDiff > math.Pi {
				angleDiff = 2*math.Pi - angleDiff
			}
			
			// Systems should be on different branches (60+ degrees apart) but similar radius
			radiusDiff := math.Abs(sys1.radius - sys2.radius)
			distance := math.Sqrt(math.Pow(float64(sys2.system.X-sys1.system.X), 2) + 
								  math.Pow(float64(sys2.system.Y-sys1.system.Y), 2))
			
			if angleDiff > math.Pi/3 && radiusDiff < sys1.radius*0.3 && 
			   distance >= minDistance && distance <= maxDistance*0.7 {
				
				// Check if lane already exists
				laneExists := false
				for _, lane := range lanes {
					if (lane.From == sys1.system.ID && lane.To == sys2.system.ID) ||
					   (lane.From == sys2.system.ID && lane.To == sys1.system.ID) {
						laneExists = true
						break
					}
				}
				
				if !laneExists {
					lanes = append(lanes, LaneData{
						From:     sys1.system.ID,
						To:       sys2.system.ID,
						FromX:    sys1.system.X,
						FromY:    sys1.system.Y,
						ToX:      sys2.system.X,
						ToY:      sys2.system.Y,
						Distance: int(distance),
					})
					
					systemConnections[sys1.system.ID]++
					systemConnections[sys2.system.ID]++
					interBranchConnections++
					break // Only one inter-branch connection per system
				}
			}
		}
	}
	
	// Phase 3: Add galactic highways - long-range connections between distant regions
	highwayConnections := 0
	maxHighways := len(systems) / 10 // Long-range highways between regions
	
	// Find systems at different quadrants for highway connections
	quadrantSystems := make([][]SystemData, 4)
	for _, sys := range systems {
		// Determine quadrant based on position relative to center
		dx := float64(sys.X) - centerX
		dy := float64(sys.Y) - centerY
		
		var quadrant int
		if dx >= 0 && dy >= 0 {
			quadrant = 0 // Northeast
		} else if dx < 0 && dy >= 0 {
			quadrant = 1 // Northwest
		} else if dx < 0 && dy < 0 {
			quadrant = 2 // Southwest
		} else {
			quadrant = 3 // Southeast
		}
		
		quadrantSystems[quadrant] = append(quadrantSystems[quadrant], sys)
	}
	
	// Create highways between quadrants
	for q1 := 0; q1 < 4 && highwayConnections < maxHighways; q1++ {
		for q2 := q1 + 1; q2 < 4 && highwayConnections < maxHighways; q2++ {
			if len(quadrantSystems[q1]) == 0 || len(quadrantSystems[q2]) == 0 {
				continue
			}
			
			// Find closest systems between these quadrants
			var bestSys1, bestSys2 SystemData
			bestDistance := math.Inf(1)
			
			for _, sys1 := range quadrantSystems[q1] {
				if systemConnections[sys1.ID] >= 4 {
					continue
				}
				for _, sys2 := range quadrantSystems[q2] {
					if systemConnections[sys2.ID] >= 4 {
						continue
					}
					
					dx := float64(sys2.X - sys1.X)
					dy := float64(sys2.Y - sys1.Y)
					distance := math.Sqrt(dx*dx + dy*dy)
					
					if distance < bestDistance && distance <= maxDistance*1.2 {
						bestSys1 = sys1
						bestSys2 = sys2
						bestDistance = distance
					}
				}
			}
			
			// Add highway if found
			if bestDistance < math.Inf(1) {
				// Check if lane already exists
				laneExists := false
				for _, lane := range lanes {
					if (lane.From == bestSys1.ID && lane.To == bestSys2.ID) ||
					   (lane.From == bestSys2.ID && lane.To == bestSys1.ID) {
						laneExists = true
						break
					}
				}
				
				if !laneExists {
					lanes = append(lanes, LaneData{
						From:     bestSys1.ID,
						To:       bestSys2.ID,
						FromX:    bestSys1.X,
						FromY:    bestSys1.Y,
						ToX:      bestSys2.X,
						ToY:      bestSys2.Y,
						Distance: int(bestDistance),
					})
					
					systemConnections[bestSys1.ID]++
					systemConnections[bestSys2.ID]++
					highwayConnections++
				}
			}
		}
	}
	
	// Phase 4: Add a few strategic inner connections
	additionalConnections := 0
	maxAdditional := len(systems) / 12 // Even fewer additional connections now
	
	for i, sys1 := range systems {
		if additionalConnections >= maxAdditional {
			break
		}
		
		for j, sys2 := range systems {
			if i >= j || additionalConnections >= maxAdditional {
				continue
			}
			
			dx := float64(sys2.X - sys1.X)
			dy := float64(sys2.Y - sys1.Y)
			distance := math.Sqrt(dx*dx + dy*dy)
			
			// Conservative inner connections
			if distance >= minDistance && distance <= maxDistance*0.4 && 
			   systemConnections[sys1.ID] <= 2 && systemConnections[sys2.ID] <= 2 {
				
				// Check if lane already exists
				laneExists := false
				for _, lane := range lanes {
					if (lane.From == sys1.ID && lane.To == sys2.ID) ||
					   (lane.From == sys2.ID && lane.To == sys1.ID) {
						laneExists = true
						break
					}
				}
				
				if !laneExists && distance <= maxDistance*0.3 {
					lanes = append(lanes, LaneData{
						From:     sys1.ID,
						To:       sys2.ID,
						FromX:    sys1.X,
						FromY:    sys1.Y,
						ToX:      sys2.X,
						ToY:      sys2.Y,
						Distance: int(distance),
					})
					
					systemConnections[sys1.ID]++
					systemConnections[sys2.ID]++
					additionalConnections++
				}
			}
		}
	}

	return lanes
}

func getPlanets(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		systemID := c.QueryParam("system_id")

		var planets []*models.Record

		// Get Null planet type ID once
		nullPlanetTypes, err := app.Dao().FindRecordsByFilter("planet_types", "name = 'Null'", "", 1, 0)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch planet types"})
		}
		
		var nullPlanetTypeID string
		if len(nullPlanetTypes) > 0 {
			nullPlanetTypeID = nullPlanetTypes[0].Id
		}

		if systemID != "" {
			// Get all non-Null planets and filter by system
			var allPlanets []*models.Record
			if nullPlanetTypeID != "" {
				allPlanets, err = app.Dao().FindRecordsByFilter("planets", "planet_type != '"+nullPlanetTypeID+"'", "", 0, 0)
			} else {
				allPlanets, err = app.Dao().FindRecordsByExpr("planets", nil, nil)
			}
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch planets"})
			}

			// Filter planets that belong to this system
			var filteredPlanets []*models.Record
			for _, planet := range allPlanets {
				// Get system_id as string slice (relation field is stored as JSON array)
				systemIDs := planet.GetStringSlice("system_id")
				for _, id := range systemIDs {
					if id == systemID {
						filteredPlanets = append(filteredPlanets, planet)
						break
					}
				}
			}
			planets = filteredPlanets
		} else {
			if nullPlanetTypeID != "" {
				planets, err = app.Dao().FindRecordsByFilter("planets", "planet_type != '"+nullPlanetTypeID+"'", "", 0, 0)
			} else {
				planets, err = app.Dao().FindRecordsByExpr("planets", nil, nil)
			}
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch planets"})
			}
		}

		planetsData := make([]PlanetData, len(planets))
		for i, planet := range planets {
			// Get the first system_id from the relation array
			systemIDs := planet.GetStringSlice("system_id")
			systemID := ""
			if len(systemIDs) > 0 {
				systemID = systemIDs[0]
			}

			planetsData[i] = PlanetData{
				ID:            planet.Id,
				Name:          planet.GetString("name"),
				SystemID:      systemID,
				PlanetType:    planet.GetString("planet_type"),
				Size:          planet.GetInt("size"),
				Population:    planet.GetInt("population"),
				MaxPopulation: planet.GetInt("max_population"),
				ColonizedBy:   planet.GetString("colonized_by"),
				ColonizedAt:   planet.GetString("colonized_at"),
			}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"page":       1,
			"perPage":    len(planetsData),
			"totalItems": len(planetsData),
			"items":      planetsData,
		})
	}
}
