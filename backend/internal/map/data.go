package mapgen

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

type DatabaseMapData struct {
	Systems      []DatabaseSystem      `json:"systems"`
	Planets      []DatabasePlanet      `json:"planets"`
	ResourceNodes []DatabaseResourceNode `json:"resource_nodes"`
}

type DatabaseSystem struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	DiscoveredBy string  `json:"discovered_by"`
}

type DatabasePlanet struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SystemID    string `json:"system_id"`
	PlanetType  string `json:"planet_type"`
	Size        int    `json:"size"`
	ColonizedBy string `json:"colonized_by"`
	ColonizedAt string `json:"colonized_at"`
}

type DatabaseResourceNode struct {
	ID           string `json:"id"`
	PlanetID     string `json:"planet_id"`
	ResourceType string `json:"resource_type"`
	Richness     int    `json:"richness"`
	Exhausted    bool   `json:"exhausted"`
}

// GetDatabaseMapData returns all systems, planets, and resource nodes for the galaxy map
func GetDatabaseMapData(app *pocketbase.PocketBase) (*DatabaseMapData, error) {
	mapData := &DatabaseMapData{
		Systems:       []DatabaseSystem{},
		Planets:       []DatabasePlanet{},
		ResourceNodes: []DatabaseResourceNode{},
	}

	// Get systems
	systems, err := app.Dao().FindRecordsByExpr("systems", nil, nil)
	if err != nil {
		return nil, err
	}

	for _, system := range systems {
		mapData.Systems = append(mapData.Systems, DatabaseSystem{
			ID:           system.Id,
			Name:         system.GetString("name"),
			X:            system.GetFloat("x"),
			Y:            system.GetFloat("y"),
			DiscoveredBy: system.GetString("discovered_by"),
		})
	}

	// Get planets
	planets, err := app.Dao().FindRecordsByExpr("planets", nil, nil)
	if err != nil {
		return nil, err
	}

	for _, planet := range planets {
		mapData.Planets = append(mapData.Planets, DatabasePlanet{
			ID:          planet.Id,
			Name:        planet.GetString("name"),
			SystemID:    planet.GetString("system_id"),
			PlanetType:  planet.GetString("planet_type"),
			Size:        planet.GetInt("size"),
			ColonizedBy: planet.GetString("colonized_by"),
			ColonizedAt: planet.GetString("colonized_at"),
		})
	}

	// Get resource nodes
	resourceNodes, err := app.Dao().FindRecordsByExpr("resource_nodes", nil, nil)
	if err != nil {
		return nil, err
	}

	for _, node := range resourceNodes {
		mapData.ResourceNodes = append(mapData.ResourceNodes, DatabaseResourceNode{
			ID:           node.Id,
			PlanetID:     node.GetString("planet_id"),
			ResourceType: node.GetString("resource_type"),
			Richness:     node.GetInt("richness"),
			Exhausted:    node.GetBool("exhausted"),
		})
	}

	return mapData, nil
}

// GetSystemsWithAggregatedData returns systems with planet counts and colony information
func GetSystemsWithAggregatedData(app *pocketbase.PocketBase) ([]map[string]interface{}, error) {
	systems, err := app.Dao().FindRecordsByExpr("systems", nil, nil)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(systems))

	for _, system := range systems {
		// Get planets in this system
		planets, err := app.Dao().FindRecordsByFilter("planets", "system_id = '"+system.Id+"'", "", 0, 0)
		if err != nil {
			continue
		}

		// Count colonized planets
		colonizedCount := 0
		totalPlanets := len(planets)
		
		for _, planet := range planets {
			if planet.GetString("colonized_by") != "" {
				colonizedCount++
			}
		}

		systemData := map[string]interface{}{
			"id":               system.Id,
			"name":             system.GetString("name"),
			"x":                system.GetFloat("x"),
			"y":                system.GetFloat("y"),
			"discovered_by":    system.GetString("discovered_by"),
			"planet_count":     totalPlanets,
			"colonized_count":  colonizedCount,
			"has_colonies":     colonizedCount > 0,
		}

		result = append(result, systemData)
	}

	return result, nil
}

// GetUserColonies returns all colonies (planets) owned by a specific user
func GetUserColonies(app *pocketbase.PocketBase, userID string) ([]*models.Record, error) {
	return app.Dao().FindRecordsByFilter("planets", "colonized_by = '"+userID+"'", "", 0, 0)
}

// GetPlanetWithDetails returns a planet with its system, type, buildings, and populations
func GetPlanetWithDetails(app *pocketbase.PocketBase, planetID string) (map[string]interface{}, error) {
	planet, err := app.Dao().FindRecordById("planets", planetID)
	if err != nil {
		return nil, err
	}

	// Get system
	system, err := app.Dao().FindRecordById("systems", planet.GetString("system_id"))
	if err != nil {
		return nil, err
	}

	// Get planet type
	planetType, err := app.Dao().FindRecordById("planet_types", planet.GetString("planet_type"))
	if err != nil {
		return nil, err
	}

	// Get buildings
	buildings, err := app.Dao().FindRecordsByFilter("buildings", "planet_id = '"+planetID+"'", "", 0, 0)
	if err != nil {
		return nil, err
	}

	// Get populations
	populations, err := app.Dao().FindRecordsByFilter("populations", "planet_id = '"+planetID+"'", "", 0, 0)
	if err != nil {
		return nil, err
	}

	// Get resource nodes
	resourceNodes, err := app.Dao().FindRecordsByFilter("resource_nodes", "planet_id = '"+planetID+"'", "", 0, 0)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"planet":         planet,
		"system":         system,
		"planet_type":    planetType,
		"buildings":      buildings,
		"populations":    populations,
		"resource_nodes": resourceNodes,
	}

	return result, nil
}