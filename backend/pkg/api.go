package pkg

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
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
		e.Router.GET("/api/debug/buildings/:planetId", debugBuildings(app))
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
		e.Router.POST("/api/orders/fleet-route", sendFleetRoute(app), apis.ActivityLogger(app), apis.RequireAdminOrRecordAuth())
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
		
		// Ship Cargo
		e.Router.GET("/api/ship_cargo", getShipCargo(app), apis.RequireAdminOrRecordAuth())
		e.Router.GET("/api/ship_cargo/:ship_id", getIndividualShipCargo(app), apis.RequireAdminOrRecordAuth())
		e.Router.POST("/api/cargo/transfer", transferCargo(app), apis.RequireAdminOrRecordAuth())
		
		// Building Storage
		e.Router.GET("/api/building_storage", getBuildingStorage(app), apis.RequireAdminOrRecordAuth())

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
	BuildingType   string `json:"building_type"`
	Res1Stored     int    `json:"res1_stored"`
	Res2Stored     int    `json:"res2_stored"`
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
	ID               string `json:"id"`
	Name             string `json:"name"`
	Cost             int    `json:"cost"`
	CostResourceType string `json:"cost_resource_type"`
	CostResourceName string `json:"cost_resource_name"`
	CostQuantity     int    `json:"cost_quantity"`
	Description      string `json:"description"`
	Category         string `json:"category"`
	BuildTime        int    `json:"build_time"`
	Icon             string `json:"icon"`
	MaxLevel         int    `json:"max_level"`
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
		log.Printf("Processing %d planets in map API", len(planets))
		for i, planet := range planets {
			log.Printf("Processing planet %s (colonized by %s)", planet.Id, planet.GetString("colonized_by"))
			// Calculate aggregated data for each planet
			totalPop := 0
			totalFood := 0
			totalOre := 0
			totalGoods := 0
			totalFuel := 0
			totalCredits := 0
			buildingCounts := make(map[string]int)

			// Get population for this planet using raw DB query
			popRows := []struct {
				Count int `db:"count"`
			}{}
			if err := app.Dao().DB().Select("count").From("populations").Where(dbx.HashExp{"planet_id": planet.Id}).All(&popRows); err != nil {
				log.Printf("Error querying populations for planet %s: %v", planet.Id, err)
			} else {
				log.Printf("Found %d population records for planet %s", len(popRows), planet.Id)
				for _, row := range popRows {
					totalPop += row.Count
				}
			}

			// Get buildings for this planet using raw DB query
			buildingRows := []struct {
				BuildingType string `db:"building_type"`
				Level        int    `db:"level"`
			}{}
			if err := app.Dao().DB().Select("building_type", "level").From("buildings").Where(dbx.HashExp{"planet_id": planet.Id}).All(&buildingRows); err != nil {
				log.Printf("Error querying buildings for planet %s: %v", planet.Id, err)
			} else {
				log.Printf("Found %d buildings for planet %s", len(buildingRows), planet.Id)
				// Process building data
				for _, row := range buildingRows {
				buildingTypeID := row.BuildingType
				level := row.Level
				
				// Get building type name using raw query
				buildingTypeRows := []struct {
					Name string `db:"name"`
				}{}
				if err := app.Dao().DB().Select("name").From("building_types").Where(dbx.HashExp{"id": buildingTypeID}).All(&buildingTypeRows); err != nil {
					log.Printf("Error querying building type %s: %v", buildingTypeID, err)
				}
				
				buildingTypeName := buildingTypeID
				if len(buildingTypeRows) > 0 {
					buildingTypeName = buildingTypeRows[0].Name
				}
				
				buildingCounts[buildingTypeName]++
				log.Printf("Planet %s: Added building %s (total: %d)", planet.Id, buildingTypeName, buildingCounts[buildingTypeName])

				// Calculate building production based on type name
				switch buildingTypeName {
				case "farm":
					totalFood += level * 10
				case "mine":
					totalOre += level * 8
				case "factory":
					totalGoods += level * 6
				case "refinery":
					totalFuel += level * 5
				case "bank":
					totalCredits += level * 1
				}
				}
			}

			log.Printf("Planet %s final data: totalPop=%d, buildingCounts=%v", planet.Id, totalPop, buildingCounts)

			planetsData[i] = PlanetData{
				ID:            planet.Id,
				Name:          planet.GetString("name"),
				SystemID:      planet.GetString("system_id"),
				PlanetType:    planet.GetString("planet_type"),
				Size:          planet.GetInt("size"),
				Population:    totalPop, // This is base population, totalPop includes this.
				MaxPopulation: 1000,     // Default max population
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

// debugBuildings returns building data for a specific planet for debugging
func debugBuildings(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		planetId := c.PathParam("planetId")
		
		result := map[string]interface{}{
			"planetId": planetId,
		}
		
		// Test population query
		popRows := []struct {
			Count int `db:"count"`
		}{}
		err := app.Dao().DB().Select("count").From("populations").Where(dbx.HashExp{"planet_id": planetId}).All(&popRows)
		result["populationQuery"] = map[string]interface{}{
			"error": err,
			"rows": popRows,
		}
		
		// Test building query
		buildingRows := []struct {
			BuildingType string `db:"building_type"`
			Level        int    `db:"level"`
		}{}
		err = app.Dao().DB().Select("building_type", "level").From("buildings").Where(dbx.HashExp{"planet_id": planetId}).All(&buildingRows)
		
		// Test building type name lookup
		buildingCounts := make(map[string]int)
		buildingDetails := []map[string]interface{}{}
		
		for _, row := range buildingRows {
			buildingTypeID := row.BuildingType
			level := row.Level
			
			// Get building type name using raw query
			buildingTypeRows := []struct {
				Name string `db:"name"`
			}{}
			nameErr := app.Dao().DB().Select("name").From("building_types").Where(dbx.HashExp{"id": buildingTypeID}).All(&buildingTypeRows)
			
			buildingTypeName := buildingTypeID
			if nameErr == nil && len(buildingTypeRows) > 0 {
				buildingTypeName = buildingTypeRows[0].Name
			}
			
			buildingCounts[buildingTypeName]++
			
			buildingDetails = append(buildingDetails, map[string]interface{}{
				"id": buildingTypeID,
				"name": buildingTypeName,
				"level": level,
				"nameError": nameErr,
			})
		}
		
		result["buildingQuery"] = map[string]interface{}{
			"error": err,
			"rows": buildingRows,
			"buildingDetails": buildingDetails,
			"buildingCounts": buildingCounts,
		}
		
		return c.JSON(http.StatusOK, result)
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
			costResourceType := record.GetString("cost_resource_type")
			costResourceName := ""
			
			// Get resource name if cost_resource_type is specified
			if costResourceType != "" {
				if resourceType, err := app.Dao().FindRecordById("resource_types", costResourceType); err == nil {
					costResourceName = resourceType.GetString("name")
				}
			}
			
			buildingTypesData[i] = BuildingTypeData{
				ID:               record.Id,
				Name:             record.GetString("name"),
				Cost:             record.GetInt("cost"),
				CostResourceType: costResourceType,
				CostResourceName: costResourceName,
				CostQuantity:     record.GetInt("cost_quantity"),
				Description:      record.GetString("description"),
				Category:         record.GetString("category"),
				BuildTime:        record.GetInt("build_time"),
				Icon:            record.GetString("icon"),
				MaxLevel:        record.GetInt("max_level"),
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
				BuildingType:   buildingTypeID,
				Res1Stored:     building.GetInt("res1_stored"),
				Res2Stored:     building.GetInt("res2_stored"),
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
			fleets, err = app.Dao().FindRecordsByFilter("fleets", filter, "name", 0, 0)
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

		log.Printf("DEBUG: Creating fleet order with values:")
		log.Printf("  user_id: %s", user.Id)
		log.Printf("  fleet_id: %s", fleet.Id)
		log.Printf("  destination_system_id: %s", data.ToID)
		log.Printf("  original_system_id: %s", data.FromID)
		log.Printf("  travel_time_ticks: %d", travelDurationInTicks)
		log.Printf("  execute_at_tick: %d", executeAtTick)

		if err := app.Dao().SaveRecord(order); err != nil {
			log.Printf("Error saving fleet order: %v", err)
			return apis.NewBadRequestError("Failed to create fleet move order", err)
		}

		log.Printf("DEBUG: Fleet order saved successfully with ID: %s", order.Id)

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":  true,
			"order_id": order.Id,
		})
	}
}

func sendFleetRoute(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := struct {
			FleetID   string   `json:"fleet_id"`
			RoutePath []string `json:"route_path"`
		}{}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Validate route path
		if len(data.RoutePath) < 2 {
			return apis.NewBadRequestError("Route path must have at least 2 systems", nil)
		}

		// Validate fleet ownership and availability
		fleet, err := app.Dao().FindRecordById("fleets", data.FleetID)
		if err != nil {
			return apis.NewBadRequestError("Fleet not found", err)
		}

		if fleet.GetString("owner_id") != user.Id {
			return apis.NewForbiddenError("You don't own this fleet", nil)
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

		// Ensure fleet starts at the first system in the route
		currentSystem := fleet.GetString("current_system")
		if currentSystem != data.RoutePath[0] {
			return apis.NewBadRequestError("Fleet is not at the starting system of the route", nil)
		}

		// Validate that all systems in the route exist
		for _, systemId := range data.RoutePath {
			if _, err := app.Dao().FindRecordById("systems", systemId); err != nil {
				return apis.NewBadRequestError("Invalid system in route path: "+systemId, err)
			}
		}

		// Get Fleet Orders collection
		fleetOrdersCollection, err := app.Dao().FindCollectionByNameOrId("fleet_orders")
		if err != nil {
			log.Printf("Error finding fleet_orders collection: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Fleet Orders collection not found", err)
		}

		// Calculate execute_at_tick for first hop
		currentTick := tick.GetCurrentTick(app)
		travelDurationInTicks := int64(2) // 2 ticks per hop
		executeAtTick := currentTick + travelDurationInTicks

		// Create a new fleet_order record with multi-hop route data
		order := models.NewRecord(fleetOrdersCollection)
		order.Set("user_id", user.Id)
		order.Set("fleet_id", fleet.Id)
		order.Set("type", "move")
		order.Set("status", "pending")
		order.Set("execute_at_tick", executeAtTick)
		
		// Set single hop fields (first hop)
		order.Set("destination_system_id", data.RoutePath[1])
		order.Set("original_system_id", data.RoutePath[0])
		order.Set("travel_time_ticks", travelDurationInTicks)
		
		// Set multi-hop fields
		order.Set("route_path", data.RoutePath)
		order.Set("current_hop", 0)
		order.Set("final_destination_id", data.RoutePath[len(data.RoutePath)-1])

		if err := app.Dao().SaveRecord(order); err != nil {
			log.Printf("Error saving fleet route order: %v", err)
			return apis.NewBadRequestError("Failed to create fleet route order", err)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":           true,
			"order_id":          order.Id,
			"route_path":        data.RoutePath,
			"final_destination": data.RoutePath[len(data.RoutePath)-1],
			"total_hops":        len(data.RoutePath) - 1,
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
			FleetID      string `json:"fleet_id"` // Fleet containing ships with resources
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

		// Get cost resource type and quantity from new flexible system
		costResourceType := buildingTypeRecord.GetString("cost_resource_type")
		costQuantity := buildingTypeRecord.GetInt("cost_quantity")

		if costResourceType != "" && costQuantity > 0 {
			// New flexible resource cost system - check and consume from ship cargo
			if data.FleetID == "" {
				return apis.NewBadRequestError("Fleet ID required for resource-based building costs", nil)
			}

			// Verify fleet ownership
			fleet, err := app.Dao().FindRecordById("fleets", data.FleetID)
			if err != nil {
				return apis.NewBadRequestError("Fleet not found", err)
			}
			if fleet.GetString("owner_id") != user.Id {
				return apis.NewForbiddenError("You don't own this fleet", nil)
			}

			// Check if fleet is at the same system as the planet
			planet, err := app.Dao().FindRecordById("planets", data.PlanetID)
			if err != nil {
				return apis.NewBadRequestError("Planet not found", err)
			}
			planetSystemID := planet.GetString("system_id")
			fleetSystemID := fleet.GetString("current_system")
			
			if planetSystemID != fleetSystemID {
				return apis.NewBadRequestError("Fleet must be in the same system as the planet to build", nil)
			}

			// Get ships in this fleet
			ships, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id='%s'", data.FleetID), "", 0, 0)
			if err != nil {
				return apis.NewBadRequestError("Failed to find ships in fleet", err)
			}

			// Calculate total resource quantity available in fleet cargo
			totalAvailable := 0
			var cargoRecords []*models.Record

			for _, ship := range ships {
				cargo, err := app.Dao().FindRecordsByFilter("ship_cargo", 
					fmt.Sprintf("ship_id='%s' && resource_type='%s'", ship.Id, costResourceType), "", 0, 0)
				if err != nil {
					continue
				}
				for _, cargoRecord := range cargo {
					totalAvailable += cargoRecord.GetInt("quantity")
					cargoRecords = append(cargoRecords, cargoRecord)
				}
			}

			// Check if we have enough resources
			if totalAvailable < costQuantity {
				resourceType, _ := app.Dao().FindRecordById("resource_types", costResourceType)
				resourceName := "unknown resource"
				if resourceType != nil {
					resourceName = resourceType.GetString("name")
				}
				return apis.NewBadRequestError(fmt.Sprintf("Insufficient %s. Need %d, have %d", resourceName, costQuantity, totalAvailable), nil)
			}

			// Consume resources from ship cargo
			remainingToConsume := costQuantity
			for _, cargoRecord := range cargoRecords {
				if remainingToConsume <= 0 {
					break
				}
				
				currentQuantity := cargoRecord.GetInt("quantity")
				consumeFromThis := currentQuantity
				if consumeFromThis > remainingToConsume {
					consumeFromThis = remainingToConsume
				}

				newQuantity := currentQuantity - consumeFromThis
				if newQuantity <= 0 {
					// Delete empty cargo record
					if err := app.Dao().DeleteRecord(cargoRecord); err != nil {
						return apis.NewBadRequestError("Failed to update ship cargo", err)
					}
				} else {
					// Update cargo quantity
					cargoRecord.Set("quantity", newQuantity)
					if err := app.Dao().SaveRecord(cargoRecord); err != nil {
						return apis.NewBadRequestError("Failed to update ship cargo", err)
					}
				}

				remainingToConsume -= consumeFromThis
			}
		} else {
			// Legacy credit-based system fallback
			cost := buildingTypeRecord.GetInt("cost")
			if cost > 0 {
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
			}
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

		// Prepare response with resource info
		response := map[string]interface{}{
			"success":     true,
			"building_id": building.Id,
		}

		if costResourceType != "" && costQuantity > 0 {
			// For resource-based costs, include resource info
			resourceType, _ := app.Dao().FindRecordById("resource_types", costResourceType)
			resourceName := "unknown"
			if resourceType != nil {
				resourceName = resourceType.GetString("name")
			}
			response["cost_resource"] = resourceName
			response["cost_quantity"] = costQuantity
		} else {
			// For credit-based costs, include credits info
			currentCredits, _ := credits.GetUserCredits(app, user.Id)
			response["cost"] = buildingTypeRecord.Get("cost")
			response["credits_remaining"] = currentCredits
		}

		return c.JSON(http.StatusOK, response)
	}
}

func getShipCargo(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		fleetID := c.QueryParam("fleet_id")
		if fleetID == "" {
			return apis.NewBadRequestError("fleet_id parameter required", nil)
		}

		// Verify fleet ownership
		fleet, err := app.Dao().FindRecordById("fleets", fleetID)
		if err != nil {
			return apis.NewBadRequestError("Fleet not found", err)
		}
		if fleet.GetString("owner_id") != user.Id {
			return apis.NewForbiddenError("You don't own this fleet", nil)
		}

		// Get ships in this fleet
		ships, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id='%s'", fleetID), "", 0, 0)
		if err != nil {
			return apis.NewBadRequestError("Failed to find ships in fleet", err)
		}

		// Aggregate cargo across all ships
		cargoSummary := make(map[string]interface{})
		totalCapacity := 0

		for _, ship := range ships {
			// Get ship type for capacity info
			shipType, err := app.Dao().FindRecordById("ship_types", ship.GetString("ship_type"))
			if err == nil {
				totalCapacity += shipType.GetInt("cargo_capacity")
			}

			// Get cargo for this ship
			cargo, err := app.Dao().FindRecordsByFilter("ship_cargo", fmt.Sprintf("ship_id='%s'", ship.Id), "", 0, 0)
			if err != nil {
				continue
			}

			for _, cargoRecord := range cargo {
				resourceTypeID := cargoRecord.GetString("resource_type")
				quantity := cargoRecord.GetInt("quantity")

				// Get resource type info
				resourceType, err := app.Dao().FindRecordById("resource_types", resourceTypeID)
				if err != nil {
					continue
				}

				resourceName := resourceType.GetString("name")
				if existing, exists := cargoSummary[resourceName]; exists {
					cargoSummary[resourceName] = existing.(int) + quantity
				} else {
					cargoSummary[resourceName] = quantity
				}
			}
		}

		// Calculate used capacity
		usedCapacity := 0
		for _, quantity := range cargoSummary {
			usedCapacity += quantity.(int)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"fleet_id":       fleetID,
			"cargo":          cargoSummary,
			"used_capacity":  usedCapacity,
			"total_capacity": totalCapacity,
			"available_space": totalCapacity - usedCapacity,
		})
	}
}

func transferCargo(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		data := struct {
			FleetID      string `json:"fleet_id"`
			ResourceType string `json:"resource_type"`
			Quantity     int    `json:"quantity"`
			Direction    string `json:"direction"` // "load" or "unload"
		}{}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request data", err)
		}

		// Validate fleet ownership
		fleet, err := app.Dao().FindRecordById("fleets", data.FleetID)
		if err != nil {
			return apis.NewBadRequestError("Fleet not found", err)
		}

		if fleet.GetString("owner_id") != user.Id {
			return apis.NewForbiddenError("You don't own this fleet", nil)
		}

		// Get fleet ships
		ships, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id='%s'", data.FleetID), "", 0, 0)
		if err != nil {
			return apis.NewBadRequestError("Failed to find ships", err)
		}

		// Find resource type
		resourceType, err := app.Dao().FindRecordsByFilter("resource_types", fmt.Sprintf("name='%s'", data.ResourceType), "", 1, 0)
		if err != nil || len(resourceType) == 0 {
			return apis.NewBadRequestError("Resource type not found", err)
		}
		resourceTypeID := resourceType[0].Id
	
		var remaining int

		if data.Direction == "unload" {
			// Transfer from fleet to player resources
			totalAvailable := 0
			var cargoRecords []*models.Record

			// Find all cargo of this type in the fleet
			for _, ship := range ships {
				cargo, err := app.Dao().FindRecordsByFilter("ship_cargo", 
					fmt.Sprintf("ship_id='%s' && resource_type='%s'", ship.Id, resourceTypeID), "", 0, 0)
				if err != nil {
					continue
				}
				for _, cargoRecord := range cargo {
					totalAvailable += cargoRecord.GetInt("quantity")
					cargoRecords = append(cargoRecords, cargoRecord)
				}
			}

			if totalAvailable < data.Quantity {
				return apis.NewBadRequestError(fmt.Sprintf("Not enough %s in fleet. Available: %d, Requested: %d", 
					data.ResourceType, totalAvailable, data.Quantity), nil)
			}

			// Remove from ship cargo
			remaining = data.Quantity
			for _, cargoRecord := range cargoRecords {
				if remaining <= 0 {
					break
				}

				currentQuantity := cargoRecord.GetInt("quantity")
				if currentQuantity <= remaining {
					// Remove entire record
					remaining -= currentQuantity
					if err := app.Dao().DeleteRecord(cargoRecord); err != nil {
						return apis.NewBadRequestError("Failed to update ship cargo", err)
					}
				} else {
					// Reduce quantity
					cargoRecord.Set("quantity", currentQuantity-remaining)
					remaining = 0
					if err := app.Dao().SaveRecord(cargoRecord); err != nil {
						return apis.NewBadRequestError("Failed to update ship cargo", err)
					}
				}
			}

			// Add to building storage - find suitable buildings in this system
			currentSystem := fleet.GetString("current_system")
			planets, err := app.Dao().FindRecordsByFilter("planets", 
				fmt.Sprintf("system_id='%s' && colonized_by='%s'", currentSystem, user.Id), "", 0, 0)
			if err != nil || len(planets) == 0 {
				return apis.NewBadRequestError("No colonized planets in this system to unload to", nil)
			}

			// Find buildings that can store this resource type
			var suitableBuildings []*models.Record
			for _, planet := range planets {
				buildings, err := app.Dao().FindRecordsByFilter("buildings", 
					fmt.Sprintf("planet_id='%s' && active=true", planet.Id), "", 0, 0)
				if err != nil {
					continue
				}
				
				for _, building := range buildings {
					buildingType, err := app.Dao().FindRecordById("building_types", building.GetString("building_type"))
					if err != nil {
						continue
					}
					
					// Check if this building can store the resource type
					if buildingType.GetString("res1_type") == resourceTypeID || buildingType.GetString("res2_type") == resourceTypeID {
						suitableBuildings = append(suitableBuildings, building)
					}
				}
			}

			if len(suitableBuildings) == 0 {
				return apis.NewBadRequestError("No suitable storage buildings found for "+data.ResourceType, nil)
			}

			// Distribute the cargo to buildings with available capacity
			remaining = data.Quantity
			for _, building := range suitableBuildings {
				if remaining <= 0 {
					break
				}

				buildingType, err := app.Dao().FindRecordById("building_types", building.GetString("building_type"))
				if err != nil {
					continue
				}
				
				level := building.GetInt("level")
				if level == 0 {
					level = 1
				}
				
				// Try to add to res1 storage first
				if buildingType.GetString("res1_type") == resourceTypeID {
					currentStored := building.GetInt("res1_stored")
					capacity := buildingType.GetInt("res1_capacity") * level
					availableSpace := capacity - currentStored
					
					if availableSpace > 0 {
						toAdd := remaining
						if toAdd > availableSpace {
							toAdd = availableSpace
						}
						building.Set("res1_stored", currentStored+toAdd)
						remaining -= toAdd
						if err := app.Dao().SaveRecord(building); err != nil {
							return apis.NewBadRequestError("Failed to update building storage", err)
						}
					}
				}
				
				// Try res2 storage if still have cargo and res1 didn't work
				if remaining > 0 && buildingType.GetString("res2_type") == resourceTypeID {
					currentStored := building.GetInt("res2_stored")
					capacity := buildingType.GetInt("res2_capacity") * level
					availableSpace := capacity - currentStored
					
					if availableSpace > 0 {
						toAdd := remaining
						if toAdd > availableSpace {
							toAdd = availableSpace
						}
						building.Set("res2_stored", currentStored+toAdd)
						remaining -= toAdd
						if err := app.Dao().SaveRecord(building); err != nil {
							return apis.NewBadRequestError("Failed to update building storage", err)
						}
					}
				}
			}

			if remaining > 0 {
				return apis.NewBadRequestError(fmt.Sprintf("Not enough storage capacity. Could only store %d of %d %s", 
					data.Quantity-remaining, data.Quantity, data.ResourceType), nil)
			}

			log.Printf("Transferred %d %s from fleet %s to building storage for player %s", data.Quantity, data.ResourceType, data.FleetID, user.Id)

		} else if data.Direction == "load" {
			// Transfer from building storage to fleet
			// First, find planets in the fleet's current system that the player owns
			currentSystem := fleet.GetString("current_system")
			planets, err := app.Dao().FindRecordsByFilter("planets", 
				fmt.Sprintf("system_id='%s' && colonized_by='%s'", currentSystem, user.Id), "", 0, 0)
			if err != nil {
				return apis.NewBadRequestError("Failed to find planets", err)
			}

			if len(planets) == 0 {
				return apis.NewBadRequestError("No colonized planets in this system to load from", nil)
			}

			// Find buildings with stored resources across all player planets in the system
			totalAvailable := 0
			var buildingsWithResource []*models.Record

			for _, planet := range planets {
				buildings, err := app.Dao().FindRecordsByFilter("buildings", 
					fmt.Sprintf("planet_id='%s' && active=true", planet.Id), "", 0, 0)
				if err != nil {
					continue
				}
				
				for _, building := range buildings {
					// Check both res1 and res2 storage for the requested resource type
					buildingType, err := app.Dao().FindRecordById("building_types", building.GetString("building_type"))
					if err != nil {
						continue
					}
					
					// Check res1 storage
					if buildingType.GetString("res1_type") == resourceTypeID {
						stored := building.GetInt("res1_stored")
						if stored > 0 {
							totalAvailable += stored
							buildingsWithResource = append(buildingsWithResource, building)
						}
					}
					
					// Check res2 storage
					if buildingType.GetString("res2_type") == resourceTypeID {
						stored := building.GetInt("res2_stored")
						if stored > 0 {
							totalAvailable += stored
							buildingsWithResource = append(buildingsWithResource, building)
						}
					}
				}
			}

			if totalAvailable < data.Quantity {
				return apis.NewBadRequestError(fmt.Sprintf("Not enough %s in building storage. Available: %d, Requested: %d", 
					data.ResourceType, totalAvailable, data.Quantity), nil)
			}

			// Check fleet cargo capacity
			totalCapacity := 0
			currentCargo := 0
			for _, ship := range ships {
				shipType, err := app.Dao().FindRecordById("ship_types", ship.GetString("ship_type"))
				if err == nil {
					totalCapacity += shipType.GetInt("cargo_capacity")
				}

				// Count current cargo
				cargo, err := app.Dao().FindRecordsByFilter("ship_cargo", 
					fmt.Sprintf("ship_id='%s'", ship.Id), "", 0, 0)
				if err == nil {
					for _, cargoRecord := range cargo {
						currentCargo += cargoRecord.GetInt("quantity")
					}
				}
			}

			if currentCargo + data.Quantity > totalCapacity {
				return apis.NewBadRequestError(fmt.Sprintf("Not enough fleet cargo space. Current: %d, Capacity: %d, Trying to add: %d", 
					currentCargo, totalCapacity, data.Quantity), nil)
			}

			// Remove from building storage
			remaining = data.Quantity
			for _, building := range buildingsWithResource {
				if remaining <= 0 {
					break
				}

				buildingType, err := app.Dao().FindRecordById("building_types", building.GetString("building_type"))
				if err != nil {
					continue
				}
				
				// Check res1 storage first
				if buildingType.GetString("res1_type") == resourceTypeID {
					stored := building.GetInt("res1_stored")
					if stored > 0 && remaining > 0 {
						toTake := stored
						if toTake > remaining {
							toTake = remaining
						}
						building.Set("res1_stored", stored-toTake)
						remaining -= toTake
						if err := app.Dao().SaveRecord(building); err != nil {
							return apis.NewBadRequestError("Failed to update building storage", err)
						}
					}
				}
				
				// Check res2 storage if still need more
				if buildingType.GetString("res2_type") == resourceTypeID && remaining > 0 {
					stored := building.GetInt("res2_stored")
					if stored > 0 {
						toTake := stored
						if toTake > remaining {
							toTake = remaining
						}
						building.Set("res2_stored", stored-toTake)
						remaining -= toTake
						if err := app.Dao().SaveRecord(building); err != nil {
							return apis.NewBadRequestError("Failed to update building storage", err)
						}
					}
				}
			}

			// Add to fleet cargo - find a ship with available space
			toLoad := data.Quantity
			for _, ship := range ships {
				if toLoad <= 0 {
					break
				}

				// Check existing cargo for this resource type
				existingCargo, err := app.Dao().FindRecordsByFilter("ship_cargo", 
					fmt.Sprintf("ship_id='%s' && resource_type='%s'", ship.Id, resourceTypeID), "", 1, 0)
				
				if err == nil && len(existingCargo) > 0 {
					// Add to existing cargo
					cargoRecord := existingCargo[0]
					newQuantity := cargoRecord.GetInt("quantity") + toLoad
					cargoRecord.Set("quantity", newQuantity)
					if err := app.Dao().SaveRecord(cargoRecord); err != nil {
						return apis.NewBadRequestError("Failed to update ship cargo", err)
					}
					toLoad = 0
				} else {
					// Create new cargo record
					cargoCollection, err := app.Dao().FindCollectionByNameOrId("ship_cargo")
					if err != nil {
						return apis.NewBadRequestError("Failed to find ship_cargo collection", err)
					}

					cargoRecord := models.NewRecord(cargoCollection)
					cargoRecord.Set("ship_id", ship.Id)
					cargoRecord.Set("resource_type", resourceTypeID)
					cargoRecord.Set("quantity", toLoad)

					if err := app.Dao().SaveRecord(cargoRecord); err != nil {
						return apis.NewBadRequestError("Failed to create ship cargo", err)
					}
					toLoad = 0
				}
			}

			log.Printf("Transferred %d %s from building storage to fleet %s for player %s", data.Quantity, data.ResourceType, data.FleetID, user.Id)

		} else {
			return apis.NewBadRequestError("Invalid direction. Use 'load' or 'unload'", nil)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":  true,
			"message":  fmt.Sprintf("Successfully %sed %d %s", data.Direction, data.Quantity, data.ResourceType),
			"fleet_id": data.FleetID,
		})
	}
}

func getBuildingStorage(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		systemID := c.QueryParam("system_id")
		if systemID == "" {
			return apis.NewBadRequestError("system_id parameter required", nil)
		}

		// Find user's planets in the system
		planets, err := app.Dao().FindRecordsByFilter("planets", 
			fmt.Sprintf("system_id='%s' && colonized_by='%s'", systemID, user.Id), "", 0, 0)
		if err != nil {
			return apis.NewBadRequestError("Failed to find planets", err)
		}

		if len(planets) == 0 {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"storage": map[string]int{},
				"buildings": []interface{}{},
			})
		}

		// Get all resource types for name mapping
		resourceTypes, err := app.Dao().FindRecordsByExpr("resource_types", nil, nil)
		if err != nil {
			return apis.NewBadRequestError("Failed to find resource types", err)
		}
		
		resourceTypeMap := make(map[string]string) // ID -> name
		for _, rt := range resourceTypes {
			resourceTypeMap[rt.Id] = rt.GetString("name")
		}

		// Aggregate storage across all buildings
		totalStorage := make(map[string]int)
		var buildingDetails []interface{}

		for _, planet := range planets {
			buildings, err := app.Dao().FindRecordsByFilter("buildings", 
				fmt.Sprintf("planet_id='%s' && active=true", planet.Id), "", 0, 0)
			if err != nil {
				continue
			}

			for _, building := range buildings {
				buildingType, err := app.Dao().FindRecordById("building_types", building.GetString("building_type"))
				if err != nil {
					continue
				}

				buildingName := buildingType.GetString("name")
				level := building.GetInt("level")
				if level == 0 {
					level = 1
				}

				// Check res1 storage
				res1Stored := building.GetInt("res1_stored")
				res1TypeID := buildingType.GetString("res1_type")
				res1Capacity := buildingType.GetInt("res1_capacity") * level

				if res1Stored > 0 && res1TypeID != "" {
					resourceName := resourceTypeMap[res1TypeID]
					if resourceName != "" {
						totalStorage[resourceName] += res1Stored
					}
				}

				// Check res2 storage
				res2Stored := building.GetInt("res2_stored")
				res2TypeID := buildingType.GetString("res2_type")
				res2Capacity := buildingType.GetInt("res2_capacity") * level

				if res2Stored > 0 && res2TypeID != "" {
					resourceName := resourceTypeMap[res2TypeID]
					if resourceName != "" {
						totalStorage[resourceName] += res2Stored
					}
				}

				// Add building details if it has storage
				if (res1Stored > 0 && res1TypeID != "") || (res2Stored > 0 && res2TypeID != "") {
					buildingDetail := map[string]interface{}{
						"id":           building.Id,
						"name":         buildingName,
						"level":        level,
						"planet_id":    building.GetString("planet_id"),
						"planet_name":  planet.GetString("name"),
					}

					if res1Stored > 0 && res1TypeID != "" {
						buildingDetail["res1_type"] = resourceTypeMap[res1TypeID]
						buildingDetail["res1_stored"] = res1Stored
						buildingDetail["res1_capacity"] = res1Capacity
					}

					if res2Stored > 0 && res2TypeID != "" {
						buildingDetail["res2_type"] = resourceTypeMap[res2TypeID]
						buildingDetail["res2_stored"] = res2Stored
						buildingDetail["res2_capacity"] = res2Capacity
					}

					buildingDetails = append(buildingDetails, buildingDetail)
				}
			}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"storage":   totalStorage,
			"buildings": buildingDetails,
		})
	}
}

func getIndividualShipCargo(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		shipID := c.PathParam("ship_id")
		if shipID == "" {
			return apis.NewBadRequestError("ship_id parameter required", nil)
		}

		// Get the ship and verify ownership through fleet
		ship, err := app.Dao().FindRecordById("ships", shipID)
		if err != nil {
			return apis.NewBadRequestError("Ship not found", err)
		}

		// Verify fleet ownership
		fleet, err := app.Dao().FindRecordById("fleets", ship.GetString("fleet_id"))
		if err != nil {
			return apis.NewBadRequestError("Fleet not found", err)
		}
		if fleet.GetString("owner_id") != user.Id {
			return apis.NewForbiddenError("You don't own this ship", nil)
		}

		// Get ship type for capacity info
		shipType, err := app.Dao().FindRecordById("ship_types", ship.GetString("ship_type"))
		if err != nil {
			return apis.NewBadRequestError("Ship type not found", err)
		}

		cargoCapacity := shipType.GetInt("cargo_capacity")

		// Get cargo for this specific ship
		cargo, err := app.Dao().FindRecordsByFilter("ship_cargo", fmt.Sprintf("ship_id='%s'", shipID), "", 0, 0)
		if err != nil {
			return apis.NewBadRequestError("Failed to find ship cargo", err)
		}

		cargoSummary := make(map[string]interface{})
		usedCapacity := 0

		for _, cargoRecord := range cargo {
			resourceTypeID := cargoRecord.GetString("resource_type")
			quantity := cargoRecord.GetInt("quantity")

			// Get resource type info
			resourceType, err := app.Dao().FindRecordById("resource_types", resourceTypeID)
			if err != nil {
				continue
			}

			resourceName := resourceType.GetString("name")
			cargoSummary[resourceName] = quantity
			usedCapacity += quantity
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"ship_id":        shipID,
			"cargo":          cargoSummary,
			"used_capacity":  usedCapacity,
			"total_capacity": cargoCapacity,
			"available_space": cargoCapacity - usedCapacity,
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

		// Get ore resource type
		oreResource, err := app.Dao().FindFirstRecordByFilter("resource_types", "name='ore'")
		if err != nil {
			return apis.NewBadRequestError("Ore resource type not found", err)
		}

		// Check colonization cost (30 ore)
		colonizationCost := 30

		// Get ships in this fleet
		ships, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id='%s'", fleet.Id), "", 0, 0)
		if err != nil {
			return apis.NewBadRequestError("Failed to find ships in fleet", err)
		}

		// Calculate total ore available in fleet cargo
		totalOre := 0
		var cargoRecords []*models.Record

		for _, ship := range ships {
			cargo, err := app.Dao().FindRecordsByFilter("ship_cargo", 
				fmt.Sprintf("ship_id='%s' && resource_type='%s'", ship.Id, oreResource.Id), "", 0, 0)
			if err != nil {
				continue
			}
			for _, cargoRecord := range cargo {
				totalOre += cargoRecord.GetInt("quantity")
				cargoRecords = append(cargoRecords, cargoRecord)
			}
		}

		// Check if we have enough ore
		if totalOre < colonizationCost {
			return apis.NewBadRequestError(fmt.Sprintf("Insufficient ore for colonization. Need %d ore, have %d", colonizationCost, totalOre), nil)
		}

		// Consume ore from ship cargo
		remainingToConsume := colonizationCost
		for _, cargoRecord := range cargoRecords {
			if remainingToConsume <= 0 {
				break
			}
			
			currentQuantity := cargoRecord.GetInt("quantity")
			consumeFromThis := currentQuantity
			if consumeFromThis > remainingToConsume {
				consumeFromThis = remainingToConsume
			}

			newQuantity := currentQuantity - consumeFromThis
			if newQuantity <= 0 {
				// Delete empty cargo record
				if err := app.Dao().DeleteRecord(cargoRecord); err != nil {
					return apis.NewBadRequestError("Failed to update ship cargo", err)
				}
			} else {
				// Update cargo quantity
				cargoRecord.Set("quantity", newQuantity)
				if err := app.Dao().SaveRecord(cargoRecord); err != nil {
					return apis.NewBadRequestError("Failed to update ship cargo", err)
				}
			}

			remainingToConsume -= consumeFromThis
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
			"success":     true,
			"planet_id":   planet.Id,
			"fleet_id":    fleet.Id,
			"ore_used":    colonizationCost,
			"ore_remaining": totalOre - colonizationCost,
			"message":     fmt.Sprintf("Planet colonized successfully using %d ore", colonizationCost),
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
