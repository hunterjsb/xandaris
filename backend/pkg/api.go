package pkg

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
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

		// Game actions
		e.Router.POST("/api/orders/fleet", sendFleet(app))
		e.Router.POST("/api/orders/build", queueBuilding(app))
		e.Router.POST("/api/orders/trade", createTradeRoute(app))
		e.Router.POST("/api/orders/colonize", colonizePlanet(app))
		e.Router.POST("/api/diplomacy", proposeTreaty(app))

		// Status endpoint
		e.Router.GET("/api/status", getStatus(app))

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

	// Aggregated data from planets/buildings
	Pop       int                    `json:"pop"`
	Morale    int                    `json:"morale"`
	Food      int                    `json:"food"`
	Ore       int                    `json:"ore"`
	Goods     int                    `json:"goods"`
	Fuel      int                    `json:"fuel"`
	Credits   int                    `json:"credits"`
	Buildings map[string]int         `json:"buildings"`
	Resources map[string]interface{} `json:"resources"`
}

type PlanetData struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SystemID      string `json:"system_id"`
	PlanetType    string `json:"type"`
	Size          int    `json:"size"`
	Population    int    `json:"population"`
	MaxPopulation int    `json:"max_population"`
	ColonizedBy   string `json:"colonized_by"`
	ColonizedAt   string `json:"colonized_at"`
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

// getMapData returns the complete map with systems, planets, and lanes
func getMapData(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get all systems
		systems, err := app.Dao().FindRecordsByExpr("systems", nil, nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch systems"})
		}

		// Get all planets
		planets, err := app.Dao().FindRecordsByExpr("planets", nil, nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch planets"})
		}

		// Transform systems data
		systemsData := make([]SystemData, len(systems))
		for i, system := range systems {
			// Get planets for this system
			systemPlanets, _ := app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("system_id='%s'", system.Id), "", 0, 0)

			// Calculate aggregated data
			totalPop := 0
			totalFood := 0
			totalOre := 0
			totalGoods := 0
			totalFuel := 0
			totalCredits := 0
			buildingCounts := make(map[string]int)

			for _, planet := range systemPlanets {
				totalPop += planet.GetInt("population")

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
			}

			systemsData[i] = SystemData{
				ID:        system.Id,
				Name:      system.GetString("name"),
				X:         system.GetInt("x"),
				Y:         system.GetInt("y"),
				OwnerID:   system.GetString("owner_id"),
				Richness:  system.GetInt("richness"),
				Pop:       totalPop,
				Morale:    75, // Default morale
				Food:      totalFood,
				Ore:       totalOre,
				Goods:     totalGoods,
				Fuel:      totalFuel,
				Credits:   totalCredits,
				Buildings: buildingCounts,
				Resources: map[string]interface{}{
					"food":  totalFood,
					"ore":   totalOre,
					"goods": totalGoods,
					"fuel":  totalFuel,
				},
			}
		}

		// Transform planets data
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

		// Get planets in this system
		planets, _ := app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("system_id='%s'", systemID), "", 0, 0)

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
			for _, building := range buildings {
				// Get planet and system to check ownership
				planet, err := app.Dao().FindRecordById("planets", building.GetString("planet_id"))
				if err != nil {
					continue
				}
				system, err := app.Dao().FindRecordById("systems", planet.GetString("system_id"))
				if err != nil {
					continue
				}
				if system.GetString("owner_id") == userID {
					filteredBuildings = append(filteredBuildings, building)
				}
			}
			buildings = filteredBuildings
		}

		// Transform to frontend format
		buildingsData := make([]BuildingData, len(buildings))
		for i, building := range buildings {
			// Get planet and system data
			planet, _ := app.Dao().FindRecordById("planets", building.GetString("planet_id"))
			var systemID, ownerID string
			if planet != nil {
				systemID = planet.GetString("system_id")
				if system, err := app.Dao().FindRecordById("systems", systemID); err == nil {
					ownerID = system.GetString("owner_id")
				}
			}

			creditsPerTick := 0
			if building.GetString("building_type") == "bank" {
				creditsPerTick = building.GetInt("level")
			}

			buildingsData[i] = BuildingData{
				ID:             building.Id,
				PlanetID:       building.GetString("planet_id"),
				SystemID:       systemID,
				OwnerID:        ownerID,
				Type:           building.GetString("building_type"),
				Name:           fmt.Sprintf("%s Level %d", building.GetString("building_type"), building.GetInt("level")),
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

		fleetsData := make([]FleetData, len(fleets))
		for i, fleet := range fleets {
			fleetsData[i] = FleetData{
				ID:       fleet.Id,
				OwnerID:  fleet.GetString("owner_id"),
				Name:     fleet.GetString("name"),
				FromID:   fleet.GetString("from_id"),
				ToID:     fleet.GetString("to_id"),
				ETA:      fleet.GetDateTime("eta").String(),
				ETATick:  fleet.GetInt("eta_tick"),
				Strength: fleet.GetInt("strength"),
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
			Strength int    `json:"strength"`
		}{}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Create fleet record
		collection, err := app.Dao().FindCollectionByNameOrId("fleets")
		if err != nil {
			return apis.NewBadRequestError("Fleets collection not found", err)
		}

		fleet := models.NewRecord(collection)
		fleet.Set("owner_id", user.Id)
		fleet.Set("from_id", data.FromID)
		fleet.Set("to_id", data.ToID)
		fleet.Set("strength", data.Strength)
		fleet.Set("eta_tick", 12) // 2 hours = 12 ticks

		if err := app.Dao().SaveRecord(fleet); err != nil {
			return apis.NewBadRequestError("Failed to create fleet", err)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":  true,
			"fleet_id": fleet.Id,
		})
	}
}

func queueBuilding(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := struct {
			SystemID     string `json:"system_id"`
			BuildingType string `json:"building_type"`
		}{}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request data", err)
		}

		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		// Get building type to check cost
		buildingType, err := app.Dao().FindRecordById("building_types", data.BuildingType)
		if err != nil {
			return apis.NewBadRequestError(fmt.Sprintf("Building %s type not found", data.BuildingType), err)
		}

		// Check building cost
		cost := buildingType.GetInt("cost")

		// Check if user has enough credits
		userCredits := user.GetInt("credits")
		if userCredits < cost {
			return apis.NewBadRequestError(fmt.Sprintf("Insufficient credits. Need %d, have %d", cost, userCredits), nil)
		}

		// Find a planet in the system to build on
		planets, err := app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("system_id='%s'", data.SystemID), "", 1, 0)
		if err != nil || len(planets) == 0 {
			return apis.NewBadRequestError("No planets found in system", err)
		}

		// Deduct credits from user
		user.Set("credits", userCredits-cost)
		if err := app.Dao().SaveRecord(user); err != nil {
			return apis.NewBadRequestError("Failed to deduct credits", err)
		}

		// Create building record
		collection, err := app.Dao().FindCollectionByNameOrId("buildings")
		if err != nil {
			return apis.NewBadRequestError("Buildings collection not found", err)
		}

		building := models.NewRecord(collection)
		building.Set("planet_id", planets[0].Id)
		building.Set("building_type", data.BuildingType)
		building.Set("level", 1)
		building.Set("active", true)

		if err := app.Dao().SaveRecord(building); err != nil {
			return apis.NewBadRequestError("Failed to create building", err)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":           true,
			"building_id":       building.Id,
			"cost":              cost,
			"credits_remaining": userCredits - cost,
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

		// Check colonization cost
		colonizationCost := 500 // Base cost to establish a colony
		userCredits := user.GetInt("credits")
		if userCredits < colonizationCost {
			return apis.NewBadRequestError(fmt.Sprintf("Insufficient credits. Colonization costs %d, you have %d", colonizationCost, userCredits), nil)
		}

		// Deduct credits from user
		user.Set("credits", userCredits-colonizationCost)
		if err := app.Dao().SaveRecord(user); err != nil {
			return apis.NewBadRequestError("Failed to deduct credits", err)
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
			"message":   "Planet colonized successfully",
		})
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
	// Get building types
	buildingTypes, err := app.Dao().FindRecordsByFilter("building_types", "name = 'Command Center'", "", 1, 0)
	if err != nil || len(buildingTypes) == 0 {
		// If no Command Center, try to get any building type
		buildingTypes, err = app.Dao().FindRecordsByExpr("building_types", nil, nil)
		if err != nil || len(buildingTypes) == 0 {
			return fmt.Errorf("no building types found")
		}
	}

	buildingCollection, err := app.Dao().FindCollectionByNameOrId("buildings")
	if err != nil {
		return err
	}

	// Create one initial building (Command Center or first available)
	building := models.NewRecord(buildingCollection)
	building.Set("planet_id", planet.Id)
	building.Set("building_type", buildingTypes[0].Id)
	building.Set("level", 1)
	building.Set("active", true)

	return app.Dao().SaveRecord(building)
}

func getStatus(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"current_tick":     1,
			"ticks_per_minute": 6,
			"server_time":      "2025-05-31T12:00:00Z",
		})
	}
}

// Helper functions
func generateLanes(systems []SystemData) []LaneData {
	lanes := make([]LaneData, 0)
	maxDistance := 300.0

	for i, sys1 := range systems {
		for j, sys2 := range systems {
			if i >= j {
				continue
			}

			dx := float64(sys2.X - sys1.X)
			dy := float64(sys2.Y - sys1.Y)
			distance := int(dx*dx + dy*dy)

			if float64(distance) <= maxDistance*maxDistance {
				lanes = append(lanes, LaneData{
					From:     sys1.ID,
					To:       sys2.ID,
					FromX:    sys1.X,
					FromY:    sys1.Y,
					ToX:      sys2.X,
					ToY:      sys2.Y,
					Distance: int(maxDistance),
				})
			}
		}
	}

	return lanes
}

func getPlanets(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		systemID := c.QueryParam("system_id")

		var planets []*models.Record
		var err error

		if systemID != "" {
			// Get all planets and filter in Go since PocketBase relation field filtering is tricky
			allPlanets, err := app.Dao().FindRecordsByExpr("planets", nil, nil)
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
			planets, err = app.Dao().FindRecordsByExpr("planets", nil, nil)
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
