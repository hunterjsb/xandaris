package pkg

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tests"
	// "github.com/labstack/echo/v5" // Not directly needed if using TestApp's router
	// "github.com/pocketbase/pocketbase" // Not directly needed for app var
	// "github.com/pocketbase/pocketbase/apis"
	// "github.com/pocketbase/pocketbase/core"
)

func saveNewRecord(t *testing.T, app *tests.TestApp, collectionName string, data map[string]interface{}) *models.Record {
	t.Helper()
	collection, err := app.Dao().FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatalf("Failed to find collection %s: %v. Ensure migrations are loaded by TestApp.", collectionName, err)
	}

	record := models.NewRecord(collection)
	for key, value := range data {
		record.Set(key, value)
	}

	if idVal, ok := data["id"]; ok {
		record.SetId(idVal.(string))
	}

	if err := app.Dao().SaveRecord(record); err != nil {
		t.Fatalf("Failed to save record to %s: %v. Data: %v", collectionName, err, data)
	}
	return record
}

func TestGetMapData_BasicIntegration(t *testing.T) {
	testApp, err := tests.NewTestApp("")
	if err != nil {
		t.Fatalf("Failed to init TestApp: %v", err)
	}
	defer testApp.Cleanup()

	// Manually register routes for this test app instance.
	// RegisterAPIRoutes takes *pocketbase.PocketBase.
	// *tests.TestApp embeds *pocketbase.PocketBase, so its methods are promoted,
	// and it can be passed to functions expecting *pocketbase.PocketBase.
	RegisterAPIRoutes(testApp)


	// 1. Create a planet_types record
	planetType := saveNewRecord(t, testApp, "planet_types", map[string]interface{}{
		"id":   "pt_terrestrial",
		"name": "Terrestrial",
		"icon": "🌍",
	})

	// 2. Create a systems record
	system := saveNewRecord(t, testApp, "systems", map[string]interface{}{
		"name":     "Sol",
		"x":        10,
		"y":        20,
		"richness": 5,
	})

	// 3. Create a planets record
	planet := saveNewRecord(t, testApp, "planets", map[string]interface{}{
		"name":        "Earth",
		"system_id":   system.Id,
		"type_id":     planetType.Id,
		"population":  100,
		"size":        5,
		"colonized_at": time.Now(),
	})

	// 4. Create a buildings record on that planet
	_ = saveNewRecord(t, testApp, "buildings", map[string]interface{}{
		"planet_id":     planet.Id,
		"building_type": "farm",
		"level":         1,
		"active":        true,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/map", nil)
	rec := httptest.NewRecorder()

	testApp.Router().ServeHTTP(rec, req)


	// Assertions
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status OK; got %d, body: %s", rec.Code, rec.Body.String())
	}

	var mapDataResponse MapData
	if err := json.Unmarshal(rec.Body.Bytes(), &mapDataResponse); err != nil {
		t.Fatalf("failed to unmarshal response: %v. Body: %s", err, rec.Body.String())
	}

	if len(mapDataResponse.Systems) != 1 {
		t.Fatalf("expected 1 system, got %d", len(mapDataResponse.Systems))
	}
	if mapDataResponse.Systems[0].ID != system.Id {
		t.Errorf("expected system ID %s, got %s", system.Id, mapDataResponse.Systems[0].ID)
	}

    var rawResponse map[string]interface{}
    if err := json.Unmarshal(rec.Body.Bytes(), &rawResponse); err != nil {
        t.Fatalf("Failed to unmarshal raw response body: %v", err)
    }
    if systemsField, ok := rawResponse["systems"].([]interface{}); ok && len(systemsField) > 0 {
        if firstSystem, ok := systemsField[0].(map[string]interface{}); ok {
            if _, exists := firstSystem["pop"]; exists {
                t.Errorf("SystemData in JSON response should not have 'pop' field, but found it.")
            }
        } else {
			t.Logf("Could not cast first system to map[string]interface{} for raw check.")
		}
    } else {
		t.Logf("No systems found in raw response for detailed check, or format unexpected.")
	}


	if len(mapDataResponse.Planets) != 1 {
		t.Fatalf("expected 1 planet, got %d", len(mapDataResponse.Planets))
	}
	respPlanet := mapDataResponse.Planets[0]

	if respPlanet.ID != planet.Id {
		t.Errorf("expected planet ID %s, got %s", planet.Id, respPlanet.ID)
	}
	if respPlanet.PlanetType != "Terrestrial" {
		t.Errorf("expected PlanetType 'Terrestrial', got '%s'", respPlanet.PlanetType)
	}
	if respPlanet.Pop != 100 {
		t.Errorf("expected planet Pop 100, got %d", respPlanet.Pop)
	}

	if respPlanet.Food != 10 {
		t.Errorf("expected planet Food 10, got %d. Actual buildings: %v", respPlanet.Food, respPlanet.Buildings)
	}
	expectedBuildings := map[string]int{"farm": 1}
	if !reflect.DeepEqual(respPlanet.Buildings, expectedBuildings) {
		t.Errorf("expected planet Buildings %v, got %v", expectedBuildings, respPlanet.Buildings)
	}

	expectedResources := map[string]interface{}{
		"food":  float64(10),
		"ore":   float64(0),
		"goods": float64(0),
		"fuel":  float64(0),
	}
	if !reflect.DeepEqual(respPlanet.Resources, expectedResources) {
		t.Errorf("expected planet Resources %v, got %v", expectedResources, respPlanet.Resources)
	}
}

func TestGetMapData_Empty(t *testing.T) {
	testApp, err := tests.NewTestApp("")
	if err != nil {
		t.Fatal(err)
	}
	defer testApp.Cleanup()

	RegisterAPIRoutes(testApp) // Register routes for this test instance as well

	req := httptest.NewRequest(http.MethodGet, "/api/map", nil)
	rec := httptest.NewRecorder()

	testApp.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status OK; got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var mapDataResponse MapData
	if err := json.Unmarshal(rec.Body.Bytes(), &mapDataResponse); err != nil {
		t.Fatalf("failed to unmarshal response: %v. Body: %s", err, rec.Body.String())
	}

	if len(mapDataResponse.Systems) != 0 {
		t.Errorf("expected 0 systems, got %d", len(mapDataResponse.Systems))
	}
	if len(mapDataResponse.Planets) != 0 {
		t.Errorf("expected 0 planets, got %d", len(mapDataResponse.Planets))
	}
}
