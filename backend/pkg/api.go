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
	"github.com/hunterjsb/xandaris/internal/mapgen" // Added import
	"github.com/hunterjsb/xandaris/internal/player" // Added import for player package
	"github.com/hunterjsb/xandaris/internal/resources" // Added import for resources utils
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
		e.Router.POST("/api/orders/colonize", colonizePlanet(app), apis.ActivityLogger(app), apis.RequireAdminOrRecordAuth()) // Legacy, ore-based
		e.Router.POST("/api/orders/colonize-ship", colonizeWithShip(app), apis.ActivityLogger(app), apis.RequireAdminOrRecordAuth()) // New, ship-based
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
		
		// Debug endpoints
		e.Router.POST("/api/debug/spawn_starter_ship", spawnStarterShip(app), apis.RequireAdminOrRecordAuth())

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

var buildingTypeCache map[string]*models.Record // Cache for building types
var buildingTypeCacheMutex sync.Mutex          // Mutex for buildingTypeCache

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
			log.Printf("ERROR: Failed to fetch systems: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch systems"})
		}

		// Get Null planet type ID once
		nullPlanetTypes, err := app.Dao().FindRecordsByFilter("planet_types", "name = 'Null'", "", 1, 0)
		if err != nil {
			log.Printf("ERROR: Failed to fetch null planet type: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch planet types"})
		}
		
		var nullPlanetTypeID string
		if len(nullPlanetTypes) > 0 {
			nullPlanetTypeID = nullPlanetTypes[0].Id
		}

		// Get all non-Null planets efficiently
		var planets []*models.Record
		if nullPlanetTypeID != "" {
			planets, err = app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("planet_type != '%s'", nullPlanetTypeID), "", 0, 0)
		} else {
			planets, err = app.Dao().FindRecordsByExpr("planets", nil, nil)
		}
		if err != nil {
			log.Printf("ERROR: Failed to fetch planets: %v", err)
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

		// Pre-fetch all building types for efficiency
		buildingTypeCacheMutex.Lock()
		if buildingTypeCache == nil {
			buildingTypeCache = make(map[string]*models.Record)
			allBuildingTypes, err := app.Dao().FindRecordsByExpr("building_types", nil)
			if err != nil {
				buildingTypeCacheMutex.Unlock()
				log.Printf("ERROR: getMapData - Failed to pre-fetch building types: %v", err)
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load building type data"})
			}
			for _, bt := range allBuildingTypes {
				buildingTypeCache[bt.Id] = bt
			}
			log.Printf("DEBUG: getMapData - Successfully cached %d building types", len(buildingTypeCache))
		}
		buildingTypeCacheMutex.Unlock()


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

			// Get population for this planet using ORM
			populationRecords, err := app.Dao().FindRecordsByFilter("populations", fmt.Sprintf("planet_id = '%s'", planet.Id), nil, 0, 0, nil)
			if err != nil {
				log.Printf("ERROR: Error querying populations for planet %s with ORM: %v", planet.Id, err)
			} else {
				log.Printf("Found %d population records for planet %s with ORM", len(populationRecords), planet.Id)
				for _, popRecord := range populationRecords {
					totalPop += popRecord.GetInt("count")
				}
			}

			// Get buildings for this planet using ORM
			buildingRecords, err := app.Dao().FindRecordsByFilter("buildings", fmt.Sprintf("planet_id = '%s'", planet.Id), nil, 0, 0, nil)
			if err != nil {
				log.Printf("ERROR: Error querying buildings for planet %s with ORM: %v", planet.Id, err)
			} else {
				log.Printf("Found %d buildings for planet %s with ORM", len(buildingRecords), planet.Id)
				for _, building := range buildingRecords {
					buildingTypeID := building.GetString("building_type")
					level := building.GetInt("level")
					if level == 0 { level = 1}


					buildingTypeName := buildingTypeID // Fallback
					buildingTypeCacheMutex.Lock() // Lock for reading cache
					if btInfo, ok := buildingTypeCache[buildingTypeID]; ok {
						buildingTypeName = btInfo.GetString("name")
					} else {
						log.Printf("WARN: Building type ID %s for building %s not found in cache. Manual fetch attempt.", buildingTypeID, building.Id)
						// Attempt a manual fetch if not in cache (should ideally not happen if cache is populated correctly)
						btManual, errManual := app.Dao().FindRecordById("building_types", buildingTypeID)
						if errManual == nil && btManual != nil {
							buildingTypeName = btManual.GetString("name")
							buildingTypeCache[buildingTypeID] = btManual // Add to cache
						} else {
							log.Printf("ERROR: Failed to manually fetch building type %s: %v", buildingTypeID, errManual)
						}
					}
					buildingTypeCacheMutex.Unlock() // Unlock after reading
				
					buildingCounts[buildingTypeName]++
					// log.Printf("Planet %s: Added building %s (total: %d)", planet.Id, buildingTypeName, buildingCounts[buildingTypeName]) // Too verbose for normal operation

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

			// log.Printf("Planet %s final data: totalPop=%d, buildingCounts=%v", planet.Id, totalPop, buildingCounts) // Too verbose for normal operation

			// Calculate MaxPopulation
			calculatedMaxPop := 0 // Default to 0 if type not found or error
			planetTypeID := planet.GetString("planet_type")
			if planetTypeID != "" {
				planetTypeRecord, err := app.Dao().FindRecordById("planet_types", planetTypeID)
				if err != nil {
					log.Printf("WARN: Failed to fetch planet_type %s for planet %s in getMapData: %v. MaxPopulation will be 0.", planetTypeID, planet.Id, err)
				} else {
					planetSize := planet.GetInt("size")
					if planetSize == 0 {
						planetSize = 1 // Ensure size is at least 1 for calculation
					}
					baseMaxPop := planetTypeRecord.GetInt("base_max_population")
					habitability := planetTypeRecord.GetFloat("habitability")
					calculatedMaxPop = int(float64(baseMaxPop) * float64(planetSize) * habitability)
					log.Printf("DEBUG: Planet %s (%s), Size: %d, BaseMaxPop: %d, Habitability: %f, CalculatedMaxPop: %d", planet.Id, planet.GetString("name"), planetSize, baseMaxPop, habitability, calculatedMaxPop)
				}
			} else {
				log.Printf("WARN: Planet %s (%s) has no planet_type set. MaxPopulation will be 0.", planet.Id, planet.GetString("name"))
			}


			planetsData[i] = PlanetData{
				ID:            planet.Id,
				Name:          planet.GetString("name"),
				SystemID:      planet.GetString("system_id"),
				PlanetType:    planetTypeID, // Use the fetched ID
				Size:          planet.GetInt("size"),
				Population:    totalPop, // This is base population, totalPop includes this.
				MaxPopulation: calculatedMaxPop,
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
		// Transform systemsData from []pkg.SystemData to []map[string]interface{} for mapgen.GenerateLanes
		systemsForMapgen := make([]map[string]interface{}, len(systemsData))
		for i, sysData := range systemsData {
			systemsForMapgen[i] = map[string]interface{}{
				"id": sysData.ID,
				"x":  sysData.X,
				"y":  sysData.Y,
				// mapgen.GenerateLanes doesn't strictly need name, owner, richness but passing them doesn't hurt
				"name":     sysData.Name,
				"owner_id": sysData.OwnerID,
				"richness": sysData.Richness,
			}
		}
		log.Printf("DEBUG: Calling mapgen.GenerateLanes with %d systems", len(systemsForMapgen))
		lanesMapData := mapgen.GenerateLanes(systemsForMapgen)
		log.Printf("DEBUG: mapgen.GenerateLanes returned %d lanes", len(lanesMapData))

		// Transform lanesMapData from []map[string]interface{} to []pkg.LaneData
		lanesData := make([]LaneData, len(lanesMapData))
		for i, laneMap := range lanesMapData {
			// Type assertions with checks for safety, though mapgen.GenerateLanes should be consistent
			from, okF := laneMap["from"].(string)
			to, okT := laneMap["to"].(string)
			fromX, okFX := laneMap["fromX"].(int)
			fromY, okFY := laneMap["fromY"].(int)
			toX, okTX := laneMap["toX"].(int)
			toY, okTY := laneMap["toY"].(int)
			distance, okD := laneMap["distance"].(int)

			if !okF || !okT || !okFX || !okFY || !okTX || !okTY || !okD {
				log.Printf("WARN: Skipping lane due to unexpected type in mapgen.GenerateLanes output: %+v", laneMap)
				continue
			}
			lanesData[i] = LaneData{
				From:     from,
				To:       to,
				FromX:    fromX,
				FromY:    fromY,
				ToX:      toX,
				ToY:      toY,
				Distance: distance,
			}
		}

		mapData := MapData{
			Systems: systemsData,
			Planets: planetsData,
			Lanes:   lanesData,
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
		
		// Fetch population records using ORM
		populationRecords, err := app.Dao().FindRecordsByFilter("populations", fmt.Sprintf("planet_id = '%s'", planetId), nil, 0, 0, nil)
		if err != nil {
			log.Printf("ERROR: Failed to query populations in debugBuildings for planet %s: %v", planetId, err)
			result["populationQuery"] = map[string]interface{}{"error": err.Error(), "records": nil}
		} else {
			result["populationQuery"] = map[string]interface{}{"error": nil, "records": populationRecords}
		}

		// Pre-fetch all building types for efficiency
		buildingTypeCacheMutex.Lock()
		if buildingTypeCache == nil {
			buildingTypeCache = make(map[string]*models.Record)
			allBuildingTypesRecords, btErr := app.Dao().FindRecordsByExpr("building_types", nil)
			if btErr != nil {
				buildingTypeCacheMutex.Unlock()
				log.Printf("ERROR: debugBuildings - Failed to pre-fetch building types: %v", btErr)
				// Not returning error for the whole endpoint, just logging and proceeding
			} else {
				for _, bt := range allBuildingTypesRecords {
					buildingTypeCache[bt.Id] = bt
				}
				log.Printf("DEBUG: debugBuildings - Successfully cached %d building types", len(buildingTypeCache))
			}
		}
		localBuildingTypeCache := make(map[string]*models.Record)
		for k, v := range buildingTypeCache { // Create a local copy for thread-safe iteration if needed, or ensure map is not modified elsewhere
			localBuildingTypeCache[k] = v
		}
		buildingTypeCacheMutex.Unlock()


		// Fetch building records using ORM
		buildingRecords, err := app.Dao().FindRecordsByFilter("buildings", fmt.Sprintf("planet_id = '%s'", planetId), nil, 0, 0, nil)
		
		buildingCounts := make(map[string]int)
		buildingDetails := []map[string]interface{}{}

		if err != nil {
			log.Printf("ERROR: Failed to query buildings in debugBuildings for planet %s: %v", planetId, err)
			result["buildingQuery"] = map[string]interface{}{"error": err.Error(), "records": nil}
		} else {
			for _, building := range buildingRecords {
				buildingTypeID := building.GetString("building_type")
				level := building.GetInt("level")
				nameErr := "N/A" // Placeholder for name error, as direct query is removed

				buildingTypeName := buildingTypeID // Fallback
				if btInfo, ok := localBuildingTypeCache[buildingTypeID]; ok {
					buildingTypeName = btInfo.GetString("name")
					nameErr = "" // No error if found in cache
				} else {
					log.Printf("WARN: Building type ID %s for building %s not found in cache during debugBuildings. Manual fetch attempt.", buildingTypeID, building.Id)
					btManual, errManual := app.Dao().FindRecordById("building_types", buildingTypeID)
					if errManual == nil && btManual != nil {
						buildingTypeName = btManual.GetString("name")
						nameErr = ""
						// Optionally add to shared cache (with mutex) if desired, though debug might not need to update global cache
					} else {
						log.Printf("ERROR: Failed to manually fetch building type %s in debugBuildings: %v", buildingTypeID, errManual)
						if errManual != nil {nameErr = errManual.Error()}
					}
				}
			
				buildingCounts[buildingTypeName]++
			
				buildingDetails = append(buildingDetails, map[string]interface{}{
					"id":          building.Id, // Changed from buildingTypeID to actual building ID
					"building_type_id": buildingTypeID,
					"name":        buildingTypeName,
					"level":       level,
					"nameError":   nameErr,
					"record_data": building.PublicExport(), // Export the full building record for debugging
				})
			}
			result["buildingQuery"] = map[string]interface{}{
				"error":           nil,
				"orm_records":     buildingRecords, // Show ORM records
				"processed_details": buildingDetails,
				"building_counts": buildingCounts,
			}
		}
		
		return c.JSON(http.StatusOK, result)
	}
}

// getBuildingTypes returns all building types
func getBuildingTypes(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		records, err := app.Dao().FindRecordsByExpr("building_types", nil, nil)
		if err != nil {
			log.Printf("ERROR: Failed to fetch building_types: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "Failed to fetch building types",
				"details": err.Error(),
			})
		}

		resourceTypeMap, err := resources.GetResourceTypeMap(app)
		if err != nil {
			log.Printf("ERROR: getBuildingTypes - Failed to get resource type map: %v", err)
			// Decide if this is fatal. For now, proceed, names might be missing.
			// return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load resource type data"})
		}

		buildingTypesData := make([]BuildingTypeData, len(records))
		for i, record := range records {
			costResourceTypeID := record.GetString("cost_resource_type")
			costResourceName := ""
			
			if costResourceTypeID != "" && resourceTypeMap != nil {
				if name, ok := resourceTypeMap[costResourceTypeID]; ok {
					costResourceName = name
				} else {
					log.Printf("WARN: getBuildingTypes - Resource type ID %s not found in map for building type %s", costResourceTypeID, record.Id)
				}
			}
			
			buildingTypesData[i] = BuildingTypeData{
				ID:               record.Id,
				Name:             record.GetString("name"),
				Cost:             record.GetInt("cost"),
				CostResourceType: costResourceTypeID,
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
			log.Printf("ERROR: Failed to fetch resource_types: %v", err)
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
			log.Printf("ERROR: Failed to fetch systems: %v (userID: %s)", err, userID)
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
			log.Printf("ERROR: System not found with ID %s: %v", systemID, err)
			return c.JSON(http.StatusNotFound, map[string]string{"error": "System not found"})
		}

		// Get Null planet type ID once
		nullPlanetTypes, err := app.Dao().FindRecordsByFilter("planet_types", "name = 'Null'", "", 1, 0)
		if err != nil {
			log.Printf("WARN: Failed to fetch null planet type in getSystem: %v", err)
			// Continue without it, planet query will fetch all types
		}
		
		// Get non-Null planets in this system efficiently
		var planets []*models.Record
		if len(nullPlanetTypes) > 0 {
			nullPlanetTypeID := nullPlanetTypes[0].Id
			planets, err = app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("system_id='%s' AND planet_type != '%s'", systemID, nullPlanetTypeID), "", 0, 0)
		} else {
			planets, err = app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("system_id='%s'", systemID), "", 0, 0)
		}
		if err != nil {
			log.Printf("WARN: Failed to fetch planets for system %s: %v. Returning system data without planets.", systemID, err)
			planets = []*models.Record{} // Ensure planetsData is empty if query fails
		}

		planetsData := make([]PlanetData, len(planets))
		for i, planet := range planets {
			// Calculate MaxPopulation
			calculatedMaxPop := 0 // Default to 0 if type not found or error
			planetTypeID := planet.GetString("planet_type")
			if planetTypeID != "" {
				planetTypeRecord, err := app.Dao().FindRecordById("planet_types", planetTypeID)
				if err != nil {
					log.Printf("WARN: Failed to fetch planet_type %s for planet %s in getSystem: %v. MaxPopulation will be 0.", planetTypeID, planet.Id, err)
				} else {
					planetSize := planet.GetInt("size")
					if planetSize == 0 {
						planetSize = 1 // Ensure size is at least 1
					}
					baseMaxPop := planetTypeRecord.GetInt("base_max_population")
					habitability := planetTypeRecord.GetFloat("habitability")
					calculatedMaxPop = int(float64(baseMaxPop) * float64(planetSize) * habitability)
					log.Printf("DEBUG: Planet %s (%s) in getSystem, Size: %d, BaseMaxPop: %d, Habitability: %f, CalculatedMaxPop: %d", planet.Id, planet.GetString("name"), planetSize, baseMaxPop, habitability, calculatedMaxPop)
				}
			} else {
				log.Printf("WARN: Planet %s (%s) in getSystem has no planet_type set. MaxPopulation will be 0.", planet.Id, planet.GetString("name"))
			}

			planetsData[i] = PlanetData{
				ID:            planet.Id,
				Name:          planet.GetString("name"),
				SystemID:      planet.GetString("system_id"),
				PlanetType:    planetTypeID,
				Size:          planet.GetInt("size"),
				Population:    planet.GetInt("population"), // This is the direct field, not aggregated like in getMapData
				MaxPopulation: calculatedMaxPop,
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
			log.Printf("ERROR: Failed to fetch buildings: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch buildings"})
		}

		// Filter by user if needed
		if userID != "" {
			var filteredBuildings []*models.Record
			for _, buildingRecord := range buildings { // Renamed 'building' to 'buildingRecord'
				planet, err := app.Dao().FindRecordById("planets", buildingRecord.GetString("planet_id"))
				if err != nil {
					log.Printf("WARN: Planet %s for building %s not found during filtering: %v", buildingRecord.GetString("planet_id"), buildingRecord.Id, err)
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
					// If planet is not colonized, owner might be derived from system
					systemRecord, errSystem := app.Dao().FindRecordById("systems", systemID)
					if errSystem != nil {
						log.Printf("WARN: System %s for planet %s not found when determining building owner: %v", systemID, planet.Id, errSystem)
					} else if systemRecord != nil {
						ownerID = systemRecord.GetString("owner_id")
					}
				}
				// Get system name
				systemRecord, errSystem := app.Dao().FindRecordById("systems", systemID)
				if errSystem != nil {
					log.Printf("WARN: System %s for planet %s not found when getting system name: %v", systemID, planet.Id, errSystem)
				} else if systemRecord != nil {
					systemName = systemRecord.GetString("name")
				}
			} else if errPlanet != nil {
				log.Printf("WARN: Planet %s for building %s not found when creating BuildingData: %v", building.GetString("planet_id"), building.Id, errPlanet)
			}

			buildingTypeID := building.GetString("building_type")
			buildingTypeName := buildingTypeID // Fallback to ID if name not found
			isBank := false

			bt, errBT := app.Dao().FindRecordById("building_types", buildingTypeID)
			if errBT != nil {
				log.Printf("WARN: Building type %s (ID: %s) for building %s not found when creating BuildingData: %v", buildingTypeName, buildingTypeID, building.Id, errBT)
			} else if bt != nil {
				buildingTypeName = bt.GetString("name")
				// Assuming "Bank" is the exact name in the 'building_types' collection for bank buildings
				if buildingTypeName == "Bank" {
					isBank = true
				}
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
			log.Printf("ERROR: Failed to fetch fleets (userID: %s): %v", userID, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch fleets"})
		}

		fleetsData := make([]map[string]interface{}, len(fleets))
		for i, fleet := range fleets {
			// Get ships in this fleet
			ships, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id='%s'", fleet.Id), "", 0, 0)
			if err != nil {
				log.Printf("WARN: Failed to fetch ships for fleet %s: %v. Fleet will be shown without ships.", fleet.Id, err)
				ships = []*models.Record{} // Empty if error
			}

			// Process ship data
			shipData := make([]map[string]interface{}, len(ships))
			for j, ship := range ships {
				// Get ship type details
				shipType, err := app.Dao().FindRecordById("ship_types", ship.GetString("ship_type"))
				var shipTypeName string
				if err != nil {
					log.Printf("WARN: Failed to fetch ship_type %s for ship %s: %v. Type will be 'unknown'.", ship.GetString("ship_type"), ship.Id, err)
					shipTypeName = "unknown"
				} else if shipType != nil {
					shipTypeName = shipType.GetString("name")
				} else {
					shipTypeName = "unknown" // Should not happen if err is nil, but defensive
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
			log.Printf("ERROR: Failed to fetch trade_routes (userID: %s): %v", userID, err)
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
			log.Printf("ERROR: Failed to fetch treaties (userID: %s): %v", userID, err)
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
			log.Printf("ERROR: Invalid request data for sendFleet: %v", err)
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			log.Printf("ERROR: Authentication required for sendFleet (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		var fleet *models.Record
		var err error

		if data.FleetID != "" {
			// Use specific fleet if provided
			fleet, err = app.Dao().FindRecordById("fleets", data.FleetID)
			if err != nil {
				log.Printf("ERROR: Fleet not found with ID %s for sendFleet: %v", data.FleetID, err)
				return apis.NewBadRequestError("Fleet not found", err)
			}
			
			// Verify ownership
			if fleet.GetString("owner_id") != user.Id {
				log.Printf("ERROR: User %s attempted to send fleet %s owned by %s", user.Id, data.FleetID, fleet.GetString("owner_id"))
				return apis.NewForbiddenError("You don't own this fleet", nil)
			}
			
			// Verify fleet is at source system
			if fleet.GetString("current_system") != data.FromID {
				log.Printf("ERROR: Fleet %s is at %s, not at source system %s for sendFleet", data.FleetID, fleet.GetString("current_system"), data.FromID)
				return apis.NewBadRequestError("Fleet is not at source system", nil)
			}
		} else {
			// Find an existing fleet at the source system
			fleetFilter := fmt.Sprintf("owner_id='%s' && current_system='%s'", user.Id, data.FromID)
			log.Printf("DEBUG: sendFleet Fleet Filter: %s", fleetFilter)
			fleets, err := app.Dao().FindRecordsByFilter("fleets", fleetFilter, "", 1, 0)
			if err != nil {
				log.Printf("ERROR: Failed to find fleets with filter %s for sendFleet: %v", fleetFilter, err)
				return apis.NewBadRequestError("Failed to find fleets", err)
			}
			log.Printf("DEBUG: sendFleet Found %d fleets", len(fleets))

			if len(fleets) == 0 {
				log.Printf("INFO: No available fleets at source system %s for user %s for sendFleet", data.FromID, user.Id)
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
		if err != nil {
			log.Printf("ERROR: Failed to check existing fleet orders for fleet %s: %v", fleet.Id, err)
			// Decide if this is a critical error or if we can proceed
			return apis.NewApiError(http.StatusInternalServerError, "Failed to check existing fleet orders", err)
		}
		if len(existingOrders) > 0 {
			log.Printf("INFO: Fleet %s already has pending orders, sendFleet aborted.", fleet.Id)
			return apis.NewBadRequestError("Fleet already has pending orders", nil)
		}

		// Validate hyperlane range (same as navigation system - 800 units max)
		fromSystem, err := app.Dao().FindRecordById("systems", data.FromID)
		if err != nil {
			log.Printf("ERROR: Source system %s not found for sendFleet: %v", data.FromID, err)
			return apis.NewBadRequestError("Source system not found", err)
		}
		toSystem, err := app.Dao().FindRecordById("systems", data.ToID)
		if err != nil {
			log.Printf("ERROR: Target system %s not found for sendFleet: %v", data.ToID, err)
			return apis.NewBadRequestError("Target system not found", err)
		}

		deltaX := toSystem.GetFloat("x") - fromSystem.GetFloat("x")
		deltaY := toSystem.GetFloat("y") - fromSystem.GetFloat("y")
		distance := math.Sqrt(deltaX*deltaX + deltaY*deltaY)

		if distance > 800 {
			log.Printf("INFO: Target system %s too far from %s (distance: %f) for sendFleet", data.ToID, data.FromID, distance)
			return apis.NewBadRequestError("Target system too far - outside hyperlane range", nil)
		}

		// Get Fleet Orders collection
		fleetOrdersCollection, err := app.Dao().FindCollectionByNameOrId("fleet_orders")
		if err != nil {
			log.Printf("ERROR: Error finding fleet_orders collection: %v", err)
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

		log.Printf("DEBUG: Creating fleet order with values: user_id=%s, fleet_id=%s, dest_system_id=%s, orig_system_id=%s, travel_time_ticks=%d, execute_at_tick=%d",
			user.Id, fleet.Id, data.ToID, data.FromID, travelDurationInTicks, executeAtTick)

		if err := app.Dao().SaveRecord(order); err != nil {
			log.Printf("ERROR: Error saving fleet order: %v", err)
			return apis.NewBadRequestError("Failed to create fleet move order", err)
		}

		log.Printf("INFO: Fleet order %s saved successfully for user %s, fleet %s", order.Id, user.Id, fleet.Id)

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
			log.Printf("ERROR: Invalid request data for sendFleetRoute: %v", err)
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			log.Printf("ERROR: Authentication required for sendFleetRoute (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Validate route path
		if len(data.RoutePath) < 2 {
			log.Printf("INFO: Route path must have at least 2 systems for sendFleetRoute, got %d", len(data.RoutePath))
			return apis.NewBadRequestError("Route path must have at least 2 systems", nil)
		}

		// Validate fleet ownership and availability
		fleet, err := app.Dao().FindRecordById("fleets", data.FleetID)
		if err != nil {
			log.Printf("ERROR: Fleet not found with ID %s for sendFleetRoute: %v", data.FleetID, err)
			return apis.NewBadRequestError("Fleet not found", err)
		}

		if fleet.GetString("owner_id") != user.Id {
			log.Printf("ERROR: User %s attempted to send fleet route for fleet %s owned by %s", user.Id, data.FleetID, fleet.GetString("owner_id"))
			return apis.NewForbiddenError("You don't own this fleet", nil)
		}

		// Check if fleet already has pending orders
		existingOrders, err := app.Dao().FindRecordsByFilter(
			"fleet_orders",
			fmt.Sprintf("fleet_id='%s' && (status='pending' || status='processing')", fleet.Id),
			"", 1, 0,
		)
		if err != nil {
			log.Printf("ERROR: Failed to check existing fleet orders for fleet %s in sendFleetRoute: %v", fleet.Id, err)
			return apis.NewApiError(http.StatusInternalServerError, "Failed to check existing fleet orders", err)
		}
		if len(existingOrders) > 0 {
			log.Printf("INFO: Fleet %s already has pending orders, sendFleetRoute aborted.", fleet.Id)
			return apis.NewBadRequestError("Fleet already has pending orders", nil)
		}

		// Ensure fleet starts at the first system in the route
		currentSystem := fleet.GetString("current_system")
		if currentSystem != data.RoutePath[0] {
			log.Printf("INFO: Fleet %s is at %s, not at the starting system %s of the route for sendFleetRoute", data.FleetID, currentSystem, data.RoutePath[0])
			return apis.NewBadRequestError("Fleet is not at the starting system of the route", nil)
		}

		// Validate that all systems in the route exist
		for i, systemId := range data.RoutePath {
			if _, err := app.Dao().FindRecordById("systems", systemId); err != nil {
				log.Printf("ERROR: Invalid system ID %s at index %d in route path for sendFleetRoute: %v", systemId, i, err)
				return apis.NewBadRequestError("Invalid system in route path: "+systemId, err)
			}
		}

		// Get Fleet Orders collection
		fleetOrdersCollection, err := app.Dao().FindCollectionByNameOrId("fleet_orders")
		if err != nil {
			log.Printf("ERROR: Error finding fleet_orders collection in sendFleetRoute: %v", err)
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

		log.Printf("DEBUG: Creating fleet route order: user=%s, fleet=%s, path_len=%d, final_dest=%s", user.Id, fleet.Id, len(data.RoutePath), data.RoutePath[len(data.RoutePath)-1])
		if err := app.Dao().SaveRecord(order); err != nil {
			log.Printf("ERROR: Error saving fleet route order: %v", err)
			return apis.NewBadRequestError("Failed to create fleet route order", err)
		}

		log.Printf("INFO: Fleet route order %s saved for user %s, fleet %s", order.Id, user.Id, fleet.Id)
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
			log.Printf("ERROR: Failed to fetch hyperlanes: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Failed to fetch hyperlanes", err)
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
			log.Printf("ERROR: Invalid request data for queueBuilding: %v", err)
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			log.Printf("ERROR: Authentication required for queueBuilding (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Verify Planet Ownership/Validity
		targetPlanet, err := app.Dao().FindRecordById("planets", data.PlanetID)
		if err != nil {
			log.Printf("ERROR: Planet not found with ID %s for queueBuilding: %v", data.PlanetID, err)
			return apis.NewNotFoundError("Planet not found.", err)
		}
		if targetPlanet.GetString("colonized_by") != user.Id {
			log.Printf("ERROR: User %s attempted to build on planet %s not owned by them (owned by %s)", user.Id, data.PlanetID, targetPlanet.GetString("colonized_by"))
			return apis.NewForbiddenError("You do not own this planet and cannot build on it.", nil)
		}

		// Get building type to check cost
		buildingTypeRecord, err := app.Dao().FindRecordById("building_types", data.BuildingType)
		if err != nil {
			log.Printf("ERROR: Building type %s not found for queueBuilding: %v", data.BuildingType, err)
			return apis.NewBadRequestError(fmt.Sprintf("Building type %s not found", data.BuildingType), err)
		}

		// Get cost resource type and quantity from new flexible system
		costResourceType := buildingTypeRecord.GetString("cost_resource_type")
		costQuantity := buildingTypeRecord.GetInt("cost_quantity")

		if costResourceType != "" && costQuantity > 0 {
			// New flexible resource cost system - check and consume from ship cargo
			if data.FleetID == "" {
				log.Printf("INFO: Fleet ID required for resource-based building costs for type %s, but not provided.", data.BuildingType)
				return apis.NewBadRequestError("Fleet ID required for resource-based building costs", nil)
			}

			// Verify fleet ownership
			fleet, err := app.Dao().FindRecordById("fleets", data.FleetID)
			if err != nil {
				log.Printf("ERROR: Fleet not found with ID %s for queueBuilding: %v", data.FleetID, err)
				return apis.NewBadRequestError("Fleet not found", err)
			}
			if fleet.GetString("owner_id") != user.Id {
				log.Printf("ERROR: User %s attempted to use fleet %s not owned by them (owned by %s) for queueBuilding", user.Id, data.FleetID, fleet.GetString("owner_id"))
				return apis.NewForbiddenError("You don't own this fleet", nil)
			}

			// Check if fleet is at the same system as the planet
			// We already have targetPlanet, no need to fetch planet again by data.PlanetID
			planetSystemID := targetPlanet.GetString("system_id")
			fleetSystemID := fleet.GetString("current_system")
			
			if planetSystemID != fleetSystemID {
				log.Printf("INFO: Fleet %s must be in system %s to build on planet %s, but is in %s.", data.FleetID, planetSystemID, data.PlanetID, fleetSystemID)
				return apis.NewBadRequestError("Fleet must be in the same system as the planet to build", nil)
			}

			// Get ships in this fleet
			ships, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id='%s'", data.FleetID), "", 0, 0)
			if err != nil {
				log.Printf("ERROR: Failed to find ships in fleet %s for queueBuilding: %v", data.FleetID, err)
				return apis.NewBadRequestError("Failed to find ships in fleet", err)
			}

			// Calculate total resource quantity available in fleet cargo
			totalAvailable := 0
			var cargoRecords []*models.Record

			for _, ship := range ships {
				cargo, err := app.Dao().FindRecordsByFilter("ship_cargo",
					fmt.Sprintf("ship_id='%s' && resource_type='%s'", ship.Id, costResourceType), "", 0, 0)
				if err != nil {
					log.Printf("WARN: Failed to query ship_cargo for ship %s, resource %s in queueBuilding: %v", ship.Id, costResourceType, err)
					continue
				}
				for _, cargoRecord := range cargo {
					totalAvailable += cargoRecord.GetInt("quantity")
					cargoRecords = append(cargoRecords, cargoRecord) // These are the specific cargo records to be debited
				}
			}

			// Check if we have enough resources
			if totalAvailable < costQuantity {
				resourceTypeName := costResourceType // Fallback name is ID
				// Attempt to get the actual name for a nicer message
				// No need to fetch the full map if only one name is potentially needed here.
				// A direct fetch is fine, or a helper GetResourceNameFromId if this pattern is common.
				resourceTypeRecord, err := app.Dao().FindRecordById("resource_types", costResourceType)
				if err == nil && resourceTypeRecord != nil {
					resourceTypeName = resourceTypeRecord.GetString("name")
				} else {
					log.Printf("WARN: queueBuilding - Failed to get resource type name for ID %s for error message: %v", costResourceType, err)
				}
				log.Printf("INFO: Insufficient %s for user %s to build %s. Need %d, have %d", resourceTypeName, user.Id, data.BuildingType, costQuantity, totalAvailable)
				return apis.NewBadRequestError(fmt.Sprintf("Insufficient %s. Need %d, have %d", resourceTypeName, costQuantity, totalAvailable), nil)
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
						log.Printf("ERROR: Failed to delete ship_cargo record %s: %v", cargoRecord.Id, err)
						return apis.NewBadRequestError("Failed to update ship cargo", err)
					}
				} else {
					// Update cargo quantity
					cargoRecord.Set("quantity", newQuantity)
					if err := app.Dao().SaveRecord(cargoRecord); err != nil {
						log.Printf("ERROR: Failed to update ship_cargo record %s quantity: %v", cargoRecord.Id, err)
						return apis.NewBadRequestError("Failed to update ship cargo", err)
					}
				}

				remainingToConsume -= consumeFromThis
			}
			log.Printf("INFO: Consumed %d of resource type %s from fleet %s for building %s", costQuantity, costResourceType, data.FleetID, data.BuildingType)
		} else {
			// Legacy credit-based system fallback
			cost := buildingTypeRecord.GetInt("cost")
			if cost > 0 {
				hasCredits, err := credits.HasSufficientCredits(app, user.Id, cost)
				if err != nil {
					log.Printf("ERROR: Failed to check credits for user %s: %v", user.Id, err)
					return apis.NewBadRequestError("Failed to check credits", err)
				}
				if !hasCredits {
					userCredits, creditsErr := credits.GetUserCredits(app, user.Id)
					if creditsErr != nil {
						log.Printf("WARN: Failed to get user credits for user %s in error message: %v", user.Id, creditsErr)
					}
					log.Printf("INFO: Insufficient credits for user %s to build %s. Need %d, have %d", user.Id, data.BuildingType, cost, userCredits)
					return apis.NewBadRequestError(fmt.Sprintf("Insufficient credits. Need %d, have %d", cost, userCredits), nil)
				}
				if err := credits.DeductUserCredits(app, user.Id, cost); err != nil {
					log.Printf("ERROR: Failed to deduct %d credits from user %s: %v", cost, user.Id, err)
					return apis.NewBadRequestError("Failed to deduct credits", err)
				}
				log.Printf("INFO: Deducted %d credits from user %s for building %s", cost, user.Id, data.BuildingType)
			}
		}

		// Create building record directly
		buildingsCollection, err := app.Dao().FindCollectionByNameOrId("buildings")
		if err != nil {
			log.Printf("ERROR: Error finding buildings collection: %v", err)
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
			log.Printf("ERROR: Error saving new building: %v", err)
			return apis.NewBadRequestError("Failed to create building", err)
		}
		log.Printf("INFO: Building %s created successfully on planet %s for user %s", building.Id, data.PlanetID, user.Id)

		// Prepare response with resource info
		response := map[string]interface{}{
			"success":     true,
			"building_id": building.Id,
		}

		if costResourceType != "" && costQuantity > 0 {
			// For resource-based costs, include resource info
			resourceTypeName := costResourceType // Fallback name is ID
			resourceTypeRecord, err := app.Dao().FindRecordById("resource_types", costResourceType)
			if err == nil && resourceTypeRecord != nil {
				resourceTypeName = resourceTypeRecord.GetString("name")
			} else {
				log.Printf("WARN: queueBuilding - Failed to get resource type name for ID %s for response: %v", costResourceType, err)
			}
			response["cost_resource"] = resourceTypeName
			response["cost_quantity"] = costQuantity
		} else {
			// For credit-based costs, include credits info
			currentCredits, err := credits.GetUserCredits(app, user.Id)
			if err != nil {
				log.Printf("WARN: Failed to get current credits for user %s for response: %v", user.Id, err)
				// response will not contain credits_remaining if this fails
			} else {
				response["credits_remaining"] = currentCredits
			}
			response["cost"] = buildingTypeRecord.GetInt("cost")
		}

		return c.JSON(http.StatusOK, response)
	}
}

func getShipCargo(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			log.Printf("ERROR: Authentication required for getShipCargo (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		fleetID := c.QueryParam("fleet_id")
		if fleetID == "" {
			log.Printf("INFO: fleet_id parameter required for getShipCargo")
			return apis.NewBadRequestError("fleet_id parameter required", nil)
		}

		// Verify fleet ownership
		fleet, err := app.Dao().FindRecordById("fleets", fleetID)
		if err != nil {
			log.Printf("ERROR: Fleet not found with ID %s for getShipCargo: %v", fleetID, err)
			return apis.NewBadRequestError("Fleet not found", err)
		}
		if fleet.GetString("owner_id") != user.Id {
			log.Printf("ERROR: User %s attempted to get cargo for fleet %s not owned by them (owned by %s)", user.Id, fleetID, fleet.GetString("owner_id"))
			return apis.NewForbiddenError("You don't own this fleet", nil)
		}

		// Get ships in this fleet
		ships, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id='%s'", fleetID), "", 0, 0)
		if err != nil {
			log.Printf("ERROR: Failed to find ships in fleet %s for getShipCargo: %v", fleetID, err)
			return apis.NewBadRequestError("Failed to find ships in fleet", err)
		}

		// Aggregate cargo across all ships
		cargoSummary := make(map[string]interface{})
		totalCapacity := 0

		// Get resource type map once
		resourceTypeMap, err := resources.GetResourceTypeMap(app)
		if err != nil {
			log.Printf("ERROR: getShipCargo - Failed to get resource type map: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Failed to load resource type data", err)
		}

		for _, ship := range ships {
			// Get ship type for capacity info
			shipType, err := app.Dao().FindRecordById("ship_types", ship.GetString("ship_type"))
			if err != nil {
				log.Printf("WARN: Failed to get ship_type %s for ship %s in getShipCargo: %v. Cargo capacity for this ship type won't be added.", ship.GetString("ship_type"), ship.Id, err)
			} else if shipType != nil {
				totalCapacity += shipType.GetInt("cargo_capacity")
			}

			// Get cargo for this ship
			cargo, err := app.Dao().FindRecordsByFilter("ship_cargo", fmt.Sprintf("ship_id='%s'", ship.Id), "", 0, 0)
			if err != nil {
				log.Printf("WARN: Failed to get ship_cargo for ship %s in getShipCargo: %v. Cargo for this ship will be skipped.", ship.Id, err)
				continue
			}

			for _, cargoRecord := range cargo {
				resourceTypeID := cargoRecord.GetString("resource_type")
				quantity := cargoRecord.GetInt("quantity")

				resourceName, ok := resourceTypeMap[resourceTypeID]
				if !ok {
					log.Printf("WARN: getShipCargo - Resource type ID %s from cargo record %s not found in map. This cargo item will be skipped.", resourceTypeID, cargoRecord.Id)
					continue
				}

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
			log.Printf("ERROR: Authentication required for transferCargo (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		data := struct {
			FleetID      string `json:"fleet_id"`
			ResourceType string `json:"resource_type"`
			Quantity     int    `json:"quantity"`
			Direction    string `json:"direction"` // "load" or "unload"
		}{}

		if err := c.Bind(&data); err != nil {
			log.Printf("ERROR: Invalid request data for transferCargo: %v", err)
			return apis.NewBadRequestError("Invalid request data", err)
		}

		// Validate fleet ownership
		fleet, err := app.Dao().FindRecordById("fleets", data.FleetID)
		if err != nil {
			log.Printf("ERROR: Fleet not found with ID %s for transferCargo: %v", data.FleetID, err)
			return apis.NewBadRequestError("Fleet not found", err)
		}

		if fleet.GetString("owner_id") != user.Id {
			log.Printf("ERROR: User %s attempted to transfer cargo for fleet %s not owned by them (owned by %s)", user.Id, data.FleetID, fleet.GetString("owner_id"))
			return apis.NewForbiddenError("You don't own this fleet", nil)
		}

		// Get fleet ships
		ships, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id='%s'", data.FleetID), "", 0, 0)
		if err != nil {
			log.Printf("ERROR: Failed to find ships in fleet %s for transferCargo: %v", data.FleetID, err)
			return apis.NewBadRequestError("Failed to find ships", err)
		}

		// Find resource type ID from its name
		resourceTypeID, err := resources.GetResourceTypeIdFromName(app, data.ResourceType)
		if err != nil {
			// GetResourceTypeIdFromName already logs the detailed error
			return apis.NewApiError(http.StatusInternalServerError, fmt.Sprintf("Error looking up resource type '%s'", data.ResourceType), err)
		}
		if resourceTypeID == "" { // Not found
			log.Printf("INFO: transferCargo - Resource type '%s' not found by name.", data.ResourceType)
			return apis.NewBadRequestError(fmt.Sprintf("Resource type '%s' not found", data.ResourceType), nil)
		}
		log.Printf("DEBUG: transferCargo - Found resource type ID %s for name %s", resourceTypeID, data.ResourceType)
	
		var remaining int

		if data.Direction == "unload" {
			// Transfer from fleet to player resources
			totalAvailable := 0
			var cargoRecordsToUpdate []*models.Record // Use a different name to avoid confusion

			// Find all cargo of this type in the fleet
			for _, ship := range ships {
				cargo, err := app.Dao().FindRecordsByFilter("ship_cargo",
					fmt.Sprintf("ship_id='%s' && resource_type='%s'", ship.Id, resourceTypeID), "", 0, 0)
				if err != nil {
					log.Printf("WARN: Failed to query ship_cargo for ship %s, resource %s in transferCargo (unload): %v", ship.Id, resourceTypeID, err)
					continue
				}
				for _, cargoRecord := range cargo {
					totalAvailable += cargoRecord.GetInt("quantity")
					cargoRecordsToUpdate = append(cargoRecordsToUpdate, cargoRecord)
				}
			}

			if totalAvailable < data.Quantity {
				log.Printf("INFO: Not enough %s in fleet %s for user %s to unload. Available: %d, Requested: %d", data.ResourceType, data.FleetID, user.Id, totalAvailable, data.Quantity)
				return apis.NewBadRequestError(fmt.Sprintf("Not enough %s in fleet. Available: %d, Requested: %d",
					data.ResourceType, totalAvailable, data.Quantity), nil)
			}

			// Remove from ship cargo
			remaining = data.Quantity
			for _, cargoRecord := range cargoRecordsToUpdate {
				if remaining <= 0 {
					break
				}

				currentQuantity := cargoRecord.GetInt("quantity")
				if currentQuantity <= remaining {
					// Remove entire record
					remaining -= currentQuantity
					if err := app.Dao().DeleteRecord(cargoRecord); err != nil {
						log.Printf("ERROR: Failed to delete ship_cargo record %s during unload: %v", cargoRecord.Id, err)
						return apis.NewBadRequestError("Failed to update ship cargo", err)
					}
				} else {
					// Reduce quantity
					cargoRecord.Set("quantity", currentQuantity-remaining)
					remaining = 0
					if err := app.Dao().SaveRecord(cargoRecord); err != nil {
						log.Printf("ERROR: Failed to update quantity for ship_cargo record %s during unload: %v", cargoRecord.Id, err)
						return apis.NewBadRequestError("Failed to update ship cargo", err)
					}
				}
			}

			// Add to building storage - find suitable buildings in this system
			currentSystem := fleet.GetString("current_system")
			planets, err := app.Dao().FindRecordsByFilter("planets",
				fmt.Sprintf("system_id='%s' && colonized_by='%s'", currentSystem, user.Id), "", 0, 0)
			if err != nil {
				log.Printf("ERROR: Failed to query planets in system %s for user %s during unload: %v", currentSystem, user.Id, err)
				return apis.NewApiError(http.StatusInternalServerError, "Failed to find planets for unloading", err)
			}
			if len(planets) == 0 {
				log.Printf("INFO: No colonized planets in system %s for user %s to unload to.", currentSystem, user.Id)
				return apis.NewBadRequestError("No colonized planets in this system to unload to", nil)
			}

			// Find buildings that can store this resource type
			var suitableBuildings []*models.Record
			for _, planet := range planets {
				buildings, err := app.Dao().FindRecordsByFilter("buildings",
					fmt.Sprintf("planet_id='%s' && active=true", planet.Id), "", 0, 0)
				if err != nil {
					log.Printf("WARN: Failed to query buildings for planet %s during unload: %v", planet.Id, err)
					continue
				}
				
				for _, building := range buildings {
					buildingType, err := app.Dao().FindRecordById("building_types", building.GetString("building_type"))
					if err != nil {
						log.Printf("WARN: Failed to get building_type %s for building %s during unload: %v", building.GetString("building_type"), building.Id, err)
						continue
					}
					
					// Check if this building can store the resource type
					if buildingType.GetString("res1_type") == resourceTypeID || buildingType.GetString("res2_type") == resourceTypeID {
						suitableBuildings = append(suitableBuildings, building)
					}
				}
			}

			if len(suitableBuildings) == 0 {
				log.Printf("INFO: No suitable storage buildings found for %s in system %s for user %s.", data.ResourceType, currentSystem, user.Id)
				return apis.NewBadRequestError("No suitable storage buildings found for "+data.ResourceType, nil)
			}

			// Distribute the cargo to buildings with available capacity
			originalQuantityToUnload := data.Quantity // Store original amount for logging
			remainingToStoreInBuildings := data.Quantity // This is the amount successfully taken from fleet

			for _, building := range suitableBuildings {
				if remainingToStoreInBuildings <= 0 {
					break
				}

				buildingType, err := app.Dao().FindRecordById("building_types", building.GetString("building_type"))
				if err != nil {
					log.Printf("WARN: Failed to get building_type %s for building %s during storage distribution: %v", building.GetString("building_type"), building.Id, err)
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
						toAdd := remainingToStoreInBuildings
						if toAdd > availableSpace {
							toAdd = availableSpace
						}
						building.Set("res1_stored", currentStored+toAdd)
						remainingToStoreInBuildings -= toAdd
						if err := app.Dao().SaveRecord(building); err != nil {
							log.Printf("ERROR: Failed to update building %s res1_stored: %v", building.Id, err)
							return apis.NewBadRequestError("Failed to update building storage", err)
						}
					}
				}
				
				// Try res2 storage if still have cargo and res1 didn't work
				if remainingToStoreInBuildings > 0 && buildingType.GetString("res2_type") == resourceTypeID {
					currentStored := building.GetInt("res2_stored")
					capacity := buildingType.GetInt("res2_capacity") * level
					availableSpace := capacity - currentStored
					
					if availableSpace > 0 {
						toAdd := remainingToStoreInBuildings
						if toAdd > availableSpace {
							toAdd = availableSpace
						}
						building.Set("res2_stored", currentStored+toAdd)
						remainingToStoreInBuildings -= toAdd
						if err := app.Dao().SaveRecord(building); err != nil {
							log.Printf("ERROR: Failed to update building %s res2_stored: %v", building.Id, err)
							return apis.NewBadRequestError("Failed to update building storage", err)
						}
					}
				}
			}

			successfullyStored := originalQuantityToUnload - remainingToStoreInBuildings
			if remainingToStoreInBuildings > 0 {
				log.Printf("WARN: Not enough storage capacity for user %s. Could only store %d of %d %s", user.Id, successfullyStored, originalQuantityToUnload, data.ResourceType)
				// Return an error but also indicate partial success if any was stored.
				// The fleet cargo was already debited, so this is a bit tricky.
				// For now, let the error indicate that not all items could be stored.
				// A more robust solution might involve trying to "return" unstorable items to the fleet, or a temporary holding.
				return apis.NewBadRequestError(fmt.Sprintf("Not enough storage capacity. Could only store %d of %d %s",
					successfullyStored, originalQuantityToUnload, data.ResourceType), nil)
			}

			log.Printf("INFO: Transferred %d %s from fleet %s to building storage for player %s", successfullyStored, data.ResourceType, data.FleetID, user.Id)

		} else if data.Direction == "load" {
			// Transfer from building storage to fleet
			// First, find planets in the fleet's current system that the player owns
			currentSystem := fleet.GetString("current_system")
			planets, err := app.Dao().FindRecordsByFilter("planets",
				fmt.Sprintf("system_id='%s' && colonized_by='%s'", currentSystem, user.Id), "", 0, 0)
			if err != nil {
				log.Printf("ERROR: Failed to find planets for user %s in system %s during load: %v", user.Id, currentSystem, err)
				return apis.NewApiError(http.StatusInternalServerError, "Failed to find planets for loading", err)
			}

			if len(planets) == 0 {
				log.Printf("INFO: No colonized planets in system %s for user %s to load from.", currentSystem, user.Id)
				return apis.NewBadRequestError("No colonized planets in this system to load from", nil)
			}

			// Find buildings with stored resources across all player planets in the system
			totalAvailable := 0
			var buildingsWithResource []*models.Record

			for _, planet := range planets {
				buildings, err := app.Dao().FindRecordsByFilter("buildings",
					fmt.Sprintf("planet_id='%s' && active=true", planet.Id), "", 0, 0)
				if err != nil {
					log.Printf("WARN: Failed to query buildings for planet %s during load: %v", planet.Id, err)
					continue
				}
				
				for _, building := range buildings {
					// Check both res1 and res2 storage for the requested resource type
					buildingType, err := app.Dao().FindRecordById("building_types", building.GetString("building_type"))
					if err != nil {
						log.Printf("WARN: Failed to get building_type %s for building %s during load: %v", building.GetString("building_type"), building.Id, err)
						continue
					}
					
					// Check res1 storage
					if buildingType.GetString("res1_type") == resourceTypeID {
						stored := building.GetInt("res1_stored")
						if stored > 0 {
							totalAvailable += stored
							// Only add building if not already added (e.g. if both res1 and res2 match)
							found := false
							for _, b := range buildingsWithResource {
								if b.Id == building.Id {
									found = true
									break
								}
							}
							if !found {
								buildingsWithResource = append(buildingsWithResource, building)
							}
						}
					}
					
					// Check res2 storage
					if buildingType.GetString("res2_type") == resourceTypeID {
						stored := building.GetInt("res2_stored")
						if stored > 0 {
							totalAvailable += stored
							found := false
							for _, b := range buildingsWithResource {
								if b.Id == building.Id {
									found = true
									break
								}
							}
							if !found {
								buildingsWithResource = append(buildingsWithResource, building)
							}
						}
					}
				}
			}

			if totalAvailable < data.Quantity {
				log.Printf("INFO: Not enough %s in building storage for user %s in system %s to load. Available: %d, Requested: %d", data.ResourceType, user.Id, currentSystem, totalAvailable, data.Quantity)
				return apis.NewBadRequestError(fmt.Sprintf("Not enough %s in building storage. Available: %d, Requested: %d",
					data.ResourceType, totalAvailable, data.Quantity), nil)
			}

			// Check fleet cargo capacity
			totalFleetCapacity := 0
			currentFleetCargo := 0
			for _, ship := range ships {
				shipType, err := app.Dao().FindRecordById("ship_types", ship.GetString("ship_type"))
				if err != nil {
					log.Printf("WARN: Failed to get ship_type %s for ship %s during capacity check: %v", ship.GetString("ship_type"), ship.Id, err)
				} else if shipType != nil {
					totalFleetCapacity += shipType.GetInt("cargo_capacity")
				}

				// Count current cargo for this ship
				cargoItems, err := app.Dao().FindRecordsByFilter("ship_cargo",
					fmt.Sprintf("ship_id='%s'", ship.Id), "", 0, 0)
				if err != nil {
					log.Printf("WARN: Failed to get current cargo for ship %s during capacity check: %v", ship.Id, err)
				} else {
					for _, cargoItem := range cargoItems {
						currentFleetCargo += cargoItem.GetInt("quantity")
					}
				}
			}

			if currentFleetCargo+data.Quantity > totalFleetCapacity {
				log.Printf("INFO: Not enough fleet cargo space for user %s to load %d %s. Current: %d, Capacity: %d", user.Id, data.Quantity, data.ResourceType, currentFleetCargo, totalFleetCapacity)
				return apis.NewBadRequestError(fmt.Sprintf("Not enough fleet cargo space. Current: %d, Capacity: %d, Trying to add: %d",
					currentFleetCargo, totalFleetCapacity, data.Quantity), nil)
			}

			// Remove from building storage
			remainingToLoadFromBuildings := data.Quantity
			for _, building := range buildingsWithResource {
				if remainingToLoadFromBuildings <= 0 {
					break
				}

				buildingType, err := app.Dao().FindRecordById("building_types", building.GetString("building_type"))
				if err != nil { // Should have been caught earlier, but defensive
					log.Printf("WARN: Failed to get building_type %s for building %s during resource removal: %v", building.GetString("building_type"), building.Id, err)
					continue
				}
				
				// Check res1 storage first
				if buildingType.GetString("res1_type") == resourceTypeID {
					stored := building.GetInt("res1_stored")
					if stored > 0 && remainingToLoadFromBuildings > 0 {
						toTake := stored
						if toTake > remainingToLoadFromBuildings {
							toTake = remainingToLoadFromBuildings
						}
						building.Set("res1_stored", stored-toTake)
						remainingToLoadFromBuildings -= toTake
						if err := app.Dao().SaveRecord(building); err != nil {
							log.Printf("ERROR: Failed to update building %s res1_stored during load: %v", building.Id, err)
							return apis.NewBadRequestError("Failed to update building storage", err)
						}
					}
				}
				
				// Check res2 storage if still need more
				if buildingType.GetString("res2_type") == resourceTypeID && remainingToLoadFromBuildings > 0 {
					stored := building.GetInt("res2_stored")
					if stored > 0 {
						toTake := stored
						if toTake > remainingToLoadFromBuildings {
							toTake = remainingToLoadFromBuildings
						}
						building.Set("res2_stored", stored-toTake)
						remainingToLoadFromBuildings -= toTake
						if err := app.Dao().SaveRecord(building); err != nil {
							log.Printf("ERROR: Failed to update building %s res2_stored during load: %v", building.Id, err)
							return apis.NewBadRequestError("Failed to update building storage", err)
						}
					}
				}
			}

			// Add to fleet cargo - distribute among ships, prioritizing ships that already have the resource or have space
			quantitySuccessfullyLoadedToFleet := data.Quantity - remainingToLoadFromBuildings
			remainingToDistributeToShips := quantitySuccessfullyLoadedToFleet

			for _, ship := range ships {
				if remainingToDistributeToShips <= 0 {
					break
				}

				// Calculate available space on this ship
				shipType, _ := app.Dao().FindRecordById("ship_types", ship.GetString("ship_type")) // Error already logged
				shipCapacity := 0
				if shipType != nil {
					shipCapacity = shipType.GetInt("cargo_capacity")
				}
				currentShipCargoQuantity := 0
				shipCargoItems, _ := app.Dao().FindRecordsByFilter("ship_cargo", fmt.Sprintf("ship_id='%s'", ship.Id), "", 0,0) // Error already logged
				for _, item := range shipCargoItems {
					currentShipCargoQuantity += item.GetInt("quantity")
				}
				shipAvailableSpace := shipCapacity - currentShipCargoQuantity


				if shipAvailableSpace <= 0 {
					continue // No space on this ship
				}

				amountToLoadToThisShip := remainingToDistributeToShips
				if amountToLoadToThisShip > shipAvailableSpace {
					amountToLoadToThisShip = shipAvailableSpace
				}

				// Check existing cargo for this resource type on this ship
				existingCargo, err := app.Dao().FindRecordsByFilter("ship_cargo",
					fmt.Sprintf("ship_id='%s' && resource_type='%s'", ship.Id, resourceTypeID), "", 1, 0)
				if err != nil {
					log.Printf("WARN: Failed to query existing ship_cargo for ship %s, resource %s during load: %v", ship.Id, resourceTypeID, err)
					// Try to create new record if possible
				}
				
				if err == nil && len(existingCargo) > 0 {
					// Add to existing cargo
					cargoRecord := existingCargo[0]
					newQuantity := cargoRecord.GetInt("quantity") + amountToLoadToThisShip
					cargoRecord.Set("quantity", newQuantity)
					if err := app.Dao().SaveRecord(cargoRecord); err != nil {
						log.Printf("ERROR: Failed to update ship_cargo %s for ship %s during load: %v", cargoRecord.Id, ship.Id, err)
						// If this fails, the resource amount might be "lost" from building but not added to ship.
						// Consider how to handle this inconsistency, for now, we continue and some resources might not be loaded.
						continue
					}
				} else {
					// Create new cargo record
					cargoCollection, err := app.Dao().FindCollectionByNameOrId("ship_cargo")
					if err != nil {
						log.Printf("ERROR: Failed to find ship_cargo collection during load: %v", err)
						continue // Cannot create new cargo record
					}

					cargoRecord := models.NewRecord(cargoCollection)
					cargoRecord.Set("ship_id", ship.Id)
					cargoRecord.Set("resource_type", resourceTypeID)
					cargoRecord.Set("quantity", amountToLoadToThisShip)

					if err := app.Dao().SaveRecord(cargoRecord); err != nil {
						log.Printf("ERROR: Failed to create new ship_cargo for ship %s during load: %v", ship.Id, err)
						continue
					}
				}
				remainingToDistributeToShips -= amountToLoadToThisShip
			}

			if remainingToDistributeToShips > 0 {
				// This means not all resources taken from buildings could be loaded onto ships,
				// which implies an issue with capacity calculation or distribution logic.
				log.Printf("WARN: Could not load all %d %s onto fleet %s. %d remaining. This may indicate a cargo capacity calculation issue.", quantitySuccessfullyLoadedToFleet, data.ResourceType, data.FleetID, remainingToDistributeToShips)
				// Consider how to reconcile this - for now, the resources are removed from buildings but not fully loaded.
			}

			log.Printf("INFO: Transferred %d %s from building storage to fleet %s for player %s", (quantitySuccessfullyLoadedToFleet - remainingToDistributeToShips), data.ResourceType, data.FleetID, user.Id)

		} else {
			log.Printf("ERROR: Invalid direction '%s' for transferCargo", data.Direction)
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
			log.Printf("ERROR: Authentication required for getBuildingStorage (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		systemID := c.QueryParam("system_id")
		if systemID == "" {
			log.Printf("INFO: system_id parameter required for getBuildingStorage")
			return apis.NewBadRequestError("system_id parameter required", nil)
		}

		// Find user's planets in the system
		planets, err := app.Dao().FindRecordsByFilter("planets",
			fmt.Sprintf("system_id='%s' && colonized_by='%s'", systemID, user.Id), "", 0, 0)
		if err != nil {
			log.Printf("ERROR: Failed to find planets in system %s for user %s (getBuildingStorage): %v", systemID, user.Id, err)
			return apis.NewBadRequestError("Failed to find planets", err)
		}

		if len(planets) == 0 {
			log.Printf("INFO: No planets found in system %s for user %s (getBuildingStorage)", systemID, user.Id)
			return c.JSON(http.StatusOK, map[string]interface{}{
				"storage":   map[string]int{},
				"buildings": []interface{}{},
			})
		}

		// Get all resource types for name mapping
		resourceTypeMap, err := resources.GetResourceTypeMap(app)
		if err != nil {
			log.Printf("ERROR: getBuildingStorage - Failed to get resource type map: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Failed to load resource type data", err)
		}
		// No need to manually build the map here anymore

		// Aggregate storage across all buildings
		totalStorage := make(map[string]int)
		var buildingDetails []interface{}

		for _, planet := range planets {
			buildings, err := app.Dao().FindRecordsByFilter("buildings",
				fmt.Sprintf("planet_id='%s' && active=true", planet.Id), "", 0, 0)
			if err != nil {
				log.Printf("WARN: Failed to query buildings for planet %s in getBuildingStorage: %v", planet.Id, err)
				continue
			}

			for _, building := range buildings {
				buildingType, err := app.Dao().FindRecordById("building_types", building.GetString("building_type"))
				if err != nil {
					log.Printf("WARN: Failed to get building_type %s for building %s in getBuildingStorage: %v", building.GetString("building_type"), building.Id, err)
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
			log.Printf("ERROR: Authentication required for getIndividualShipCargo (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		shipID := c.PathParam("ship_id")
		if shipID == "" {
			// This case should ideally be caught by Echo's routing if ship_id is a path param
			log.Printf("ERROR: ship_id parameter is empty for getIndividualShipCargo")
			return apis.NewBadRequestError("ship_id parameter required", nil)
		}

		// Get the ship and verify ownership through fleet
		ship, err := app.Dao().FindRecordById("ships", shipID)
		if err != nil {
			log.Printf("ERROR: Ship not found with ID %s for getIndividualShipCargo: %v", shipID, err)
			return apis.NewBadRequestError("Ship not found", err)
		}

		// Verify fleet ownership
		fleet, err := app.Dao().FindRecordById("fleets", ship.GetString("fleet_id"))
		if err != nil {
			log.Printf("ERROR: Fleet not found with ID %s (for ship %s) for getIndividualShipCargo: %v", ship.GetString("fleet_id"), shipID, err)
			return apis.NewBadRequestError("Fleet not found", err)
		}
		if fleet.GetString("owner_id") != user.Id {
			log.Printf("ERROR: User %s attempted to get cargo for ship %s in fleet %s not owned by them (owned by %s)", user.Id, shipID, fleet.Id, fleet.GetString("owner_id"))
			return apis.NewForbiddenError("You don't own this ship", nil)
		}

		// Get ship type for capacity info
		shipType, err := app.Dao().FindRecordById("ship_types", ship.GetString("ship_type"))
		if err != nil {
			log.Printf("ERROR: Ship type %s not found for ship %s in getIndividualShipCargo: %v", ship.GetString("ship_type"), shipID, err)
			return apis.NewBadRequestError("Ship type not found", err)
		}

		cargoCapacity := shipType.GetInt("cargo_capacity")

		// Get cargo for this specific ship
		cargo, err := app.Dao().FindRecordsByFilter("ship_cargo", fmt.Sprintf("ship_id='%s'", shipID), "", 0, 0)
		if err != nil {
			log.Printf("ERROR: Failed to find ship_cargo for ship %s in getIndividualShipCargo: %v", shipID, err)
			return apis.NewBadRequestError("Failed to find ship cargo", err)
		}

		cargoSummary := make(map[string]interface{})
		usedCapacity := 0

		// Get resource type map once
		resourceTypeMap, err := resources.GetResourceTypeMap(app)
		if err != nil {
			log.Printf("ERROR: getIndividualShipCargo - Failed to get resource type map: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Failed to load resource type data", err)
		}

		for _, cargoRecord := range cargo {
			resourceTypeID := cargoRecord.GetString("resource_type")
			quantity := cargoRecord.GetInt("quantity")

			resourceName, ok := resourceTypeMap[resourceTypeID]
			if !ok {
				log.Printf("WARN: getIndividualShipCargo - Resource type ID %s from cargo record %s not found in map. This cargo item will be skipped.", resourceTypeID, cargoRecord.Id)
				continue
			}

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

func spawnStarterShip(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			log.Printf("ERROR: Authentication required for spawnStarterShip (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Find a system for the user (preferably one they own, or any system)
		var targetSystem *models.Record
		userSystems, err := app.Dao().FindRecordsByExpr("systems", dbx.HashExp{"discovered_by": user.Id}, nil)
		if err != nil {
			log.Printf("WARN: Failed to query user-discovered systems for spawnStarterShip (user %s): %v. Will try fallback.", user.Id, err)
		}

		if err == nil && len(userSystems) > 0 {
			targetSystem = userSystems[0]
		} else {
			allSystems, err := app.Dao().FindRecordsByExpr("systems", nil, nil)
			if err != nil {
				log.Printf("ERROR: Failed to query any systems for spawnStarterShip: %v", err)
				return apis.NewApiError(http.StatusInternalServerError, "No systems available to spawn starter ship", err)
			}
			if len(allSystems) == 0 {
				log.Printf("ERROR: No systems found in the database for spawnStarterShip")
				return apis.NewApiError(http.StatusInternalServerError, "No systems available in the galaxy", nil)
			}
			targetSystem = allSystems[0]
		}

		log.Printf("INFO: User %s invoking spawnStarterShip in system %s (%s)", user.Id, targetSystem.Id, targetSystem.GetString("name"))

		fleet, ship, err := player.CreateUserStarterFleet(app, user.Id, targetSystem.Id)
		if err != nil {
			// CreateUserStarterFleet already logs detailed errors
			log.Printf("ERROR: Failed to create starter fleet via utility for user %s in system %s: %v", user.Id, targetSystem.Id, err)
			// Determine appropriate API error based on the error from CreateUserStarterFleet
			// For simplicity, returning a generic bad request or internal server error.
			// A more granular error handling could inspect `err` further.
			if strings.Contains(err.Error(), "not found") {
				return apis.NewBadRequestError("Failed to create starter fleet due to missing definitions (e.g., ship type).", err)
			}
			return apis.NewApiError(http.StatusInternalServerError, "Failed to spawn starter ship.", err)
		}

		// Starter cargo details are not directly returned by CreateUserStarterFleet in this example,
		// but could be if the utility function was designed to do so.
		// For now, just confirm success.
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":    true,
			"message":    "Starter ship spawned successfully.",
			"fleet_id":   fleet.Id,
			"ship_id":    ship.Id,
			"system_id":  targetSystem.Id,
			// "cargo": starterCargo, // This would require CreateUserStarterFleet to return cargo details
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
			log.Printf("ERROR: Invalid request data for createTradeRoute: %v", err)
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			log.Printf("ERROR: Authentication required for createTradeRoute (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Create trade route record
		collection, err := app.Dao().FindCollectionByNameOrId("trade_routes")
		if err != nil {
			log.Printf("ERROR: Failed to find trade_routes collection in createTradeRoute: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Trade routes collection not found", err)
		}

		route := models.NewRecord(collection)
		route.Set("owner_id", user.Id)
		route.Set("from_id", data.FromID)
		route.Set("to_id", data.ToID)
		route.Set("cargo", data.Cargo)
		route.Set("capacity", data.Capacity)
		route.Set("eta_tick", 6) // 1 hour = 6 ticks

		if err := app.Dao().SaveRecord(route); err != nil {
			log.Printf("ERROR: Failed to create trade route for user %s (from %s to %s): %v", user.Id, data.FromID, data.ToID, err)
			return apis.NewBadRequestError("Failed to create trade route", err)
		}
		log.Printf("INFO: Trade route %s created successfully for user %s from %s to %s", route.Id, user.Id, data.FromID, data.ToID)
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
			log.Printf("ERROR: Invalid request data for proposeTreaty: %v", err)
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			log.Printf("ERROR: Authentication required for proposeTreaty (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Create treaty record
		collection, err := app.Dao().FindCollectionByNameOrId("treaties")
		if err != nil {
			log.Printf("ERROR: Failed to find treaties collection in proposeTreaty: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Treaties collection not found", err)
		}

		treaty := models.NewRecord(collection)
		treaty.Set("type", data.Type)
		treaty.Set("a_id", user.Id)
		treaty.Set("b_id", data.PlayerID)
		treaty.Set("status", "proposed")

		if err := app.Dao().SaveRecord(treaty); err != nil {
			log.Printf("ERROR: Failed to create treaty between user %s and player %s (type %s): %v", user.Id, data.PlayerID, data.Type, err)
			return apis.NewBadRequestError("Failed to create treaty", err)
		}
		log.Printf("INFO: Treaty %s proposed successfully between user %s and player %s (type %s)", treaty.Id, user.Id, data.PlayerID, data.Type)
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
			log.Printf("ERROR: Invalid request data for colonizePlanet: %v", err)
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			log.Printf("ERROR: Authentication required for colonizePlanet (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Get the planet
		planet, err := app.Dao().FindRecordById("planets", data.PlanetID)
		if err != nil {
			log.Printf("ERROR: Planet not found with ID %s for colonizePlanet: %v", data.PlanetID, err)
			return apis.NewBadRequestError("Planet not found", err)
		}

		// Check if planet is already colonized
		if planet.GetString("colonized_by") != "" {
			log.Printf("INFO: Planet %s is already colonized by %s. Colonization attempt by %s failed.", data.PlanetID, planet.GetString("colonized_by"), user.Id)
			return apis.NewBadRequestError("Planet is already colonized", nil)
		}

		// Get the fleet
		fleet, err := app.Dao().FindRecordById("fleets", data.FleetID)
		if err != nil {
			log.Printf("ERROR: Fleet not found with ID %s for colonizePlanet: %v", data.FleetID, err)
			return apis.NewBadRequestError("Fleet not found", err)
		}

		// Verify fleet ownership
		if fleet.GetString("owner_id") != user.Id {
			log.Printf("ERROR: User %s attempted to colonize with fleet %s not owned by them (owned by %s)", user.Id, data.FleetID, fleet.GetString("owner_id"))
			return apis.NewForbiddenError("You don't own this fleet", nil) // Changed from Unauthorized
		}

		// Get the system the planet is in
		system, err := app.Dao().FindRecordById("systems", planet.GetString("system_id"))
		if err != nil {
			log.Printf("ERROR: System not found with ID %s (for planet %s) for colonizePlanet: %v", planet.GetString("system_id"), data.PlanetID, err)
			return apis.NewBadRequestError("System not found", err)
		}

		// Check if fleet is at the same system as the planet
		if fleet.GetString("current_system") != system.Id {
			log.Printf("INFO: Fleet %s must be in system %s to colonize planet %s, but is in %s.", data.FleetID, system.Id, data.PlanetID, fleet.GetString("current_system"))
			return apis.NewBadRequestError("Fleet must be at the same system as the planet", nil)
		}

		// Get ore resource type ID
		oreResourceID, err := resources.GetResourceTypeIdFromName(app, "ore")
		if err != nil {
			log.Printf("ERROR: colonizePlanet - Failed to query for 'ore' resource type ID: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Error determining ore resource type", err)
		}
		if oreResourceID == "" {
			log.Printf("ERROR: colonizePlanet - 'ore' resource type not found by name.")
			return apis.NewApiError(http.StatusInternalServerError, "Ore resource type definition missing", nil)
		}
		log.Printf("DEBUG: colonizePlanet - Ore resource type ID: %s", oreResourceID)

		// Check colonization cost (30 ore)
		colonizationCost := 30

		// Get ships in this fleet
		ships, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id='%s'", fleet.Id), "", 0, 0)
		if err != nil {
			log.Printf("ERROR: Failed to find ships in fleet %s for colonizePlanet: %v", data.FleetID, err)
			return apis.NewBadRequestError("Failed to find ships in fleet", err)
		}

		// Calculate total ore available in fleet cargo
		totalOre := 0
		var cargoRecordsToUpdate []*models.Record // Renamed to avoid confusion

		for _, ship := range ships {
			cargo, err := app.Dao().FindRecordsByFilter("ship_cargo",
				fmt.Sprintf("ship_id='%s' && resource_type='%s'", ship.Id, oreResourceID), "", 0, 0)
			if err != nil {
				log.Printf("WARN: Failed to query ship_cargo for ship %s, resource 'ore' (ID: %s) in colonizePlanet: %v", ship.Id, oreResourceID, err)
				continue
			}
			for _, cargoRecord := range cargo {
				totalOre += cargoRecord.GetInt("quantity")
				cargoRecordsToUpdate = append(cargoRecordsToUpdate, cargoRecord)
			}
		}

		// Check if we have enough ore
		if totalOre < colonizationCost {
			log.Printf("INFO: Insufficient ore for user %s to colonize planet %s. Need %d, have %d", user.Id, data.PlanetID, colonizationCost, totalOre)
			return apis.NewBadRequestError(fmt.Sprintf("Insufficient ore for colonization. Need %d ore, have %d", colonizationCost, totalOre), nil)
		}

		// Consume ore from ship cargo
		remainingToConsume := colonizationCost
		for _, cargoRecord := range cargoRecordsToUpdate { // Use the renamed slice
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
					log.Printf("ERROR: Failed to delete ship_cargo record %s during colonization ore consumption: %v", cargoRecord.Id, err)
					return apis.NewBadRequestError("Failed to update ship cargo", err)
				}
			} else {
				// Update cargo quantity
				cargoRecord.Set("quantity", newQuantity)
				if err := app.Dao().SaveRecord(cargoRecord); err != nil {
					log.Printf("ERROR: Failed to update quantity for ship_cargo record %s during colonization ore consumption: %v", cargoRecord.Id, err)
					return apis.NewBadRequestError("Failed to update ship cargo", err)
				}
			}

			remainingToConsume -= consumeFromThis
		}
		log.Printf("INFO: Consumed %d ore from fleet %s for user %s to colonize planet %s", colonizationCost, data.FleetID, user.Id, data.PlanetID)


		// Set colonization data
		planet.Set("colonized_by", user.Id)
		planet.Set("colonized_at", time.Now().UTC().Format(time.RFC3339Nano)) // Use UTC and standard format

		if err := app.Dao().SaveRecord(planet); err != nil {
			log.Printf("ERROR: Failed to save planet %s colonization data for user %s: %v", data.PlanetID, user.Id, err)
			return apis.NewBadRequestError("Failed to colonize planet", err)
		}

		// Create initial population
		if err := createInitialPopulation(app, planet, user.Id); err != nil {
			log.Printf("ERROR: Failed to create initial population on planet %s for user %s after colonization: %v", data.PlanetID, user.Id, err)
			// This is a significant error, might want to roll back colonization or mark planet as needing attention
			return apis.NewBadRequestError("Failed to create initial population", err)
		}

		// Create initial buildings (optional)
		if err := createInitialBuildings(app, planet); err != nil {
			// Don't fail colonization if buildings fail, just log it
			log.Printf("WARN: Failed to create initial buildings for planet %s after colonization by user %s: %v", planet.Id, user.Id, err)
		}
		log.Printf("INFO: Planet %s colonized successfully by user %s using fleet %s.", data.PlanetID, user.Id, data.FleetID)
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
			log.Printf("ERROR: Authentication required for colonizeWithShip (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		log.Printf("INFO: colonizeWithShip called by user %s with data: PlanetID=%s, FleetID=%s", user.Id, data.PlanetID, data.FleetID)

		// 1. Fetch the planet
		planet, err := app.Dao().FindRecordById("planets", data.PlanetID)
		if err != nil {
			log.Printf("ERROR: Planet not found with ID %s for colonizeWithShip: %v", data.PlanetID, err)
			return apis.NewBadRequestError("Planet not found", err)
		}

		// 2. Check if planet is already colonized
		if planet.GetString("colonized_by") != "" {
			log.Printf("INFO: Planet %s is already colonized by %s. colonizeWithShip attempt by %s failed.", data.PlanetID, planet.GetString("colonized_by"), user.Id)
			return apis.NewBadRequestError("Planet is already colonized", nil)
		}

		// 3. Fetch the fleet
		fleet, err := app.Dao().FindRecordById("fleets", data.FleetID)
		if err != nil {
			log.Printf("ERROR: Fleet not found with ID %s for colonizeWithShip: %v", data.FleetID, err)
			return apis.NewBadRequestError("Fleet not found", err)
		}

		// 4. Verify fleet ownership
		if fleet.GetString("owner_id") != user.Id {
			log.Printf("ERROR: User %s attempted to colonizeWithShip with fleet %s not owned by them (owned by %s)", user.Id, data.FleetID, fleet.GetString("owner_id"))
			return apis.NewForbiddenError("You don't own this fleet", nil)
		}

		// 5. Verify fleet is in the same system as the planet
		planetSystem, err := app.Dao().FindRecordById("systems", planet.GetString("system_id"))
		if err != nil {
			log.Printf("ERROR: System not found with ID %s (for planet %s) for colonizeWithShip: %v", planet.GetString("system_id"), data.PlanetID, err)
			return apis.NewBadRequestError("System for planet not found", err)
		}
		if fleet.GetString("current_system") != planetSystem.Id {
			log.Printf("INFO: Fleet %s must be in system %s to colonizeWithShip on planet %s, but is in %s.", data.FleetID, planetSystem.Id, data.PlanetID, fleet.GetString("current_system"))
			return apis.NewBadRequestError("Fleet must be at the same system as the planet", nil)
		}

		// 6. Find the "settler" ship type.
		//    For this implementation, we'll hardcode "settler". A more flexible approach might involve a "colonizer_ship" flag on ship_types.
		settlerShipType, err := app.Dao().FindFirstRecordByFilter("ship_types", "name = 'settler'")
		if err != nil {
			log.Printf("ERROR: Failed to query for 'settler' ship type in colonizeWithShip: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Could not find settler ship type definition", err)
		}
		if settlerShipType == nil {
			log.Printf("ERROR: 'settler' ship type definition not found in database for colonizeWithShip.")
			return apis.NewApiError(http.StatusInternalServerError, "Settler ship type definition missing", nil)
		}

		// 7. Find a settler ship in the fleet
		settlerShips, err := app.Dao().FindRecordsByFilter(
			"ships",
			fmt.Sprintf("fleet_id = '%s' && ship_type = '%s'", fleet.Id, settlerShipType.Id),
			"created", // get the oldest one first if multiple stacks
			1, 0,
		)
		if err != nil {
			log.Printf("ERROR: Failed to query for settler ships in fleet %s for colonizeWithShip: %v", fleet.Id, err)
			return apis.NewApiError(http.StatusInternalServerError, "Failed to check for settler ship", err)
		}
		if len(settlerShips) == 0 {
			log.Printf("INFO: No settler ships found in fleet %s for user %s for colonizeWithShip.", fleet.Id, user.Id)
			return apis.NewBadRequestError("No settler ship in the fleet.", nil)
		}
		colonizingShip := settlerShips[0]

		// 8. Colonization Logic
		// Update planet
		planet.Set("colonized_by", user.Id)
		planet.Set("colonized_at", time.Now().UTC().Format(time.RFC3339Nano))
		if err := app.Dao().SaveRecord(planet); err != nil {
			log.Printf("ERROR: Failed to save planet %s colonization data for colonizeWithShip by user %s: %v", data.PlanetID, user.Id, err)
			return apis.NewApiError(http.StatusInternalServerError, "Failed to update planet data", err)
		}
		log.Printf("INFO: Planet %s updated to colonized_by %s at %s", planet.Id, user.Id, planet.GetDateTime("colonized_at").String())

		// Remove one settler ship
		currentShipCount := colonizingShip.GetInt("count")
		if currentShipCount > 1 {
			colonizingShip.Set("count", currentShipCount-1)
			if err := app.Dao().SaveRecord(colonizingShip); err != nil {
				log.Printf("ERROR: Failed to decrement count for ship %s (settler) in colonizeWithShip: %v", colonizingShip.Id, err)
				// Continue colonization, but log error. Planet is colonized, ship count is inconsistent.
			} else {
				log.Printf("INFO: Decremented count of settler ship %s to %d", colonizingShip.Id, currentShipCount-1)
			}
		} else {
			if err := app.Dao().DeleteRecord(colonizingShip); err != nil {
				log.Printf("ERROR: Failed to delete ship %s (settler, count 0) in colonizeWithShip: %v", colonizingShip.Id, err)
				// Continue colonization, but log error.
			} else {
				log.Printf("INFO: Deleted settler ship %s as count reached 0", colonizingShip.Id)
			}
		}

		// Check if fleet is empty and delete if so
		remainingShips, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id = '%s'", fleet.Id), "", 1, 0)
		if err != nil {
			log.Printf("WARN: Failed to check remaining ships in fleet %s after colonization: %v. Fleet may not be auto-deleted if empty.", fleet.Id, err)
		} else if len(remainingShips) == 0 {
			if err := app.Dao().DeleteRecord(fleet); err != nil {
				log.Printf("WARN: Failed to delete empty fleet %s after colonization: %v", fleet.Id, err)
			} else {
				log.Printf("INFO: Deleted empty fleet %s after colonization ship was consumed.", fleet.Id)
			}
		}

		// Create initial population & buildings
		if err := createInitialPopulation(app, planet, user.Id); err != nil {
			log.Printf("ERROR: Failed to create initial population on planet %s for user %s after colonizeWithShip: %v", data.PlanetID, user.Id, err)
			// Planet is colonized, but no population. This is a partial success/failure.
			return apis.NewApiError(http.StatusInternalServerError, "Planet colonized, but failed to create initial population.", err)
		}
		if err := createInitialBuildings(app, planet); err != nil {
			log.Printf("WARN: Failed to create initial buildings for planet %s after colonizeWithShip by user %s: %v", planet.Id, user.Id, err)
			// This is a warning, colonization is still considered mostly successful.
		}

		log.Printf("INFO: Planet %s colonized successfully by user %s using settler ship from fleet %s.", data.PlanetID, user.Id, data.FleetID)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":   true,
			"planet_id": planet.Id,
			"message":   fmt.Sprintf("Planet %s colonized successfully with a settler ship.", planet.GetString("name")),
		})
	}
}

func createInitialPopulation(app *pocketbase.PocketBase, planet *models.Record, ownerID string) error {
	populationCollection, err := app.Dao().FindCollectionByNameOrId("populations")
	if err != nil {
		log.Printf("ERROR: Failed to find 'populations' collection in createInitialPopulation for planet %s: %v", planet.Id, err)
		return fmt.Errorf("failed to find populations collection: %w", err)
	}

	population := models.NewRecord(populationCollection)
	population.Set("owner_id", ownerID)
	population.Set("planet_id", planet.Id)
	population.Set("count", 100)    // Start with 100 population
	population.Set("happiness", 80) // Start with 80% happiness

	err = app.Dao().SaveRecord(population)
	if err != nil {
		log.Printf("ERROR: Failed to save initial population for planet %s, owner %s: %v", planet.Id, ownerID, err)
		return fmt.Errorf("failed to save initial population: %w", err)
	}
	log.Printf("INFO: Initial population created for planet %s, owner %s", planet.Id, ownerID)
	return nil
}

func createInitialBuildings(app *pocketbase.PocketBase, planet *models.Record) error {
	ownerID := planet.GetString("colonized_by")
	log.Printf("INFO: Creating initial buildings for planet %s, owner %s", planet.Id, ownerID)

	// Check if this is the user's first colony by checking if they have any other colonies
	existingColonies, err := app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("colonized_by = '%s' AND id != '%s'", ownerID, planet.Id), "", 1, 0) // Exclude current planet, limit 1
	if err != nil {
		log.Printf("WARN: Failed to check existing colonies for owner %s (planet %s) in createInitialBuildings: %v", ownerID, planet.Id, err)
		// Proceed assuming it might be the first, or default to non-first colony buildings.
		// Or return error if this check is critical. For now, log and continue.
	}

	isFirstColony := (err == nil && len(existingColonies) == 0)
	log.Printf("INFO: Planet %s is determined to be first colony: %t for owner %s", planet.Id, isFirstColony, ownerID)

	buildingCollection, err := app.Dao().FindCollectionByNameOrId("buildings")
	if err != nil {
		log.Printf("ERROR: Failed to find 'buildings' collection in createInitialBuildings for planet %s: %v", planet.Id, err)
		return fmt.Errorf("failed to find buildings collection: %w", err)
	}

	var buildingTypeRecord *models.Record
	var buildingTypeName string

	if isFirstColony {
		buildingTypeName = "crypto_server"
		// For first colony, create a starter crypto_server with credits
		buildingTypeRecord, err = app.Dao().FindFirstRecordByFilter("building_types", "name = 'crypto_server'")
		if err != nil {
			log.Printf("ERROR: Failed to find 'crypto_server' building type for planet %s: %w", planet.Id, err)
			return fmt.Errorf("crypto_server building type not found: %w", err)
		}
	} else {
		buildingTypeName = "base"
		// For subsequent colonies, create a basic base building
		buildingTypeRecord, err = app.Dao().FindFirstRecordByFilter("building_types", "name = 'base'")
		if err != nil {
			log.Printf("ERROR: Failed to find 'base' building type for planet %s: %w", planet.Id, err)
			return fmt.Errorf("base building type not found: %w", err)
		}
	}

	if buildingTypeRecord == nil { // Should be caught by err != nil, but defensive
		log.Printf("ERROR: Building type record for '%s' is nil for planet %s", buildingTypeName, planet.Id)
		return fmt.Errorf("building type '%s' record not found", buildingTypeName)
	}

	building := models.NewRecord(buildingCollection)
	building.Set("planet_id", planet.Id)
	building.Set("building_type", buildingTypeRecord.Id)
	building.Set("level", 1)
	building.Set("active", true)

	// Crypto servers start empty - they generate credits over time

	err = app.Dao().SaveRecord(building)
	if err != nil {
		log.Printf("ERROR: Failed to save initial building (%s) for planet %s: %v", buildingTypeName, planet.Id, err)
		return fmt.Errorf("failed to save initial %s building: %w", buildingTypeName, err)
	}
	log.Printf("INFO: Initial building %s (%s) created for planet %s", building.Id, buildingTypeName, planet.Id)
	return nil
}

func getStatus(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		currentTickVal := tick.GetCurrentTick(app)
		tickRateVal := tick.GetTickRate()
		serverTimeVal := time.Now().Format(time.RFC3339)
		log.Printf("INFO: getStatus called. Tick: %d, Rate: %d/min, Time: %s", currentTickVal, tickRateVal, serverTimeVal)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"current_tick":     currentTickVal,
			"ticks_per_minute": tickRateVal,
			"server_time":      serverTimeVal,
		})
	}
}

func getUserResources(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			log.Printf("ERROR: Authentication required for getUserResources (user is nil)")
			return apis.NewUnauthorizedError("Authentication required", nil)
		}
		log.Printf("INFO: getUserResources called for user %s", user.Id)

		// Get credits from crypto_server buildings
		userCredits, err := credits.GetUserCredits(app, user.Id)
		if err != nil {
			log.Printf("ERROR: Failed to get credits for user %s in getUserResources: %v", user.Id, err)
			// Potentially return an error response if credits are critical and failed to load
			// For now, defaulting to 0 and continuing to load other resources.
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
// func generateLanes(systems []SystemData) []LaneData { ... } // This function is now removed

func getPlanets(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		systemID := c.QueryParam("system_id")

		var planets []*models.Record

		log.Printf("INFO: getPlanets called. SystemID: '%s'", systemID)
		// Get Null planet type ID once
		nullPlanetTypes, err := app.Dao().FindRecordsByFilter("planet_types", "name = 'Null'", "", 1, 0)
		if err != nil {
			log.Printf("WARN: Failed to fetch null planet type in getPlanets: %v. Proceeding without filtering null planets.", err)
			// nullPlanetTypeID will remain empty, and filtering will not occur / might be incorrect if 'Null' planets exist with empty type string
		}
		
		var nullPlanetTypeID string
		if len(nullPlanetTypes) > 0 {
			nullPlanetTypeID = nullPlanetTypes[0].Id
		}

		if systemID != "" {
			// Get all non-Null planets and filter by system
			var allPlanets []*models.Record
			filter := "id != ''" // Base filter to get all planets if nullPlanetTypeID is empty
			if nullPlanetTypeID != "" {
				filter = fmt.Sprintf("planet_type != '%s'", nullPlanetTypeID)
			}
			allPlanets, err = app.Dao().FindRecordsByFilter("planets", filter, "", 0, 0)

			if err != nil {
				log.Printf("ERROR: Failed to fetch all planets (for system filtering) in getPlanets: %v", err)
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch planets"})
			}

			// Filter planets that belong to this system
			var filteredPlanets []*models.Record
			for _, planet := range allPlanets {
				// Get system_id as string slice (relation field is stored as JSON array)
				// PocketBase stores relation IDs as plain strings if it's a single relation,
				// or a JSON array of strings if it's a multiple relation.
				// Assuming system_id on planets is a single relation field to systems collection.
				planetSystemID := planet.GetString("system_id") // If it's a single relation
				if planetSystemID == systemID {
					filteredPlanets = append(filteredPlanets, planet)
				}
				// If system_id were a multiple relation field:
				// systemIDs := planet.GetStringSlice("system_id")
				// for _, id := range systemIDs {
				// 	if id == systemID {
				// 		filteredPlanets = append(filteredPlanets, planet)
				// 		break
				// 	}
				// }
			}
			planets = filteredPlanets
		} else {
			// Get all non-Null planets if no systemID is specified
			filter := "id != ''"
			if nullPlanetTypeID != "" {
				filter = fmt.Sprintf("planet_type != '%s'", nullPlanetTypeID)
			}
			planets, err = app.Dao().FindRecordsByFilter("planets", filter, "", 0, 0)
			if err != nil {
				log.Printf("ERROR: Failed to fetch all non-null planets in getPlanets: %v", err)
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch planets"})
			}
		}

		planetsData := make([]PlanetData, len(planets))
		for i, planet := range planets {
			// Assuming system_id on a planet is a single relation field.
			planetSystemID := planet.GetString("system_id")
			// If it were a multiple relation:
			// systemIDs := planet.GetStringSlice("system_id")
			// var planetSystemID string
			// if len(systemIDs) > 0 {
			// 	planetSystemID = systemIDs[0] // Or handle multiple systems if necessary
			// }

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
