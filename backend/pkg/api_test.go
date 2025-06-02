package pkg_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/pocketbase/pocketbase/tools/store"
	"github.com/pocketbase/pocketbase/tests"
	
	"github.com/hunterjsb/xandaris/internal/tick" // For tick related calculations
	"github.com/hunterjsb/xandaris/pkg" // api handlers are here
)

func createTestUser(app tests.TestApp, email, password string) (*models.Record, error) {
	user := &models.Record{}
	user.RefreshId()
	user.SetEmail(email)
	user.SetPassword(password)
	user.SetVerified(true)
	user.Collection().Name = "_users" // Important: ensure collection is set for DAO operations

	return user, app.Dao().SaveRecord(user)
}

func createTestSystem(app tests.TestApp, name string, x, y int) (*models.Record, error) {
	collection, err := app.Dao().FindCollectionByNameOrId("systems")
	if err != nil {
		return nil, err
	}
	record := models.NewRecord(collection)
	record.Set("name", name)
	record.Set("x", x)
	record.Set("y", y)
	return record, app.Dao().SaveRecord(record)
}

func createTestPlanet(app tests.TestApp, name string, systemID string, ownerID string) (*models.Record, error) {
	collection, err := app.Dao().FindCollectionByNameOrId("planets")
	if err != nil {
		return nil, err
	}
	record := models.NewRecord(collection)
	record.Set("name", name)
	record.Set("system_id", systemID)
	if ownerID != "" {
		record.Set("colonized_by", ownerID)
		record.Set("colonized_at", time.Now())
	}
	return record, app.Dao().SaveRecord(record)
}

func createTestFleet(app tests.TestApp, name string, ownerID string, currentSystemID string) (*models.Record, error) {
	collection, err := app.Dao().FindCollectionByNameOrId("fleets")
	if err != nil {
		return nil, err
	}
	record := models.NewRecord(collection)
	record.Set("name", name)
	record.Set("owner_id", ownerID)
	record.Set("current_system", currentSystemID)
	record.Set("destination_system", "") // Ensure it's stationary
	record.Set("eta", nil)
	return record, app.Dao().SaveRecord(record)
}

func createTestBuildingType(app tests.TestApp, name string, cost int, buildTime int) (*models.Record, error) {
	collection, err := app.Dao().FindCollectionByNameOrId("building_types")
	if err != nil {
		return nil, err
	}
	record := models.NewRecord(collection)
	record.Set("name", name)
	record.Set("cost", cost) // Assuming cost is just an integer for simplicity in tests
	record.Set("build_time", buildTime) // in seconds
	return record, app.Dao().SaveRecord(record)
}


func TestSendFleetAPI(t *testing.T) {
	app, cleanup := tests.NewTestApp(t)
	defer cleanup()

	// Setup Router
	pkg.RegisterAPIRoutes(app) // Register the API routes

	// Create user
	user, err := createTestUser(app, "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create systems
	fromSystem, err := createTestSystem(app, "Sol", 0, 0)
	if err != nil {
		t.Fatalf("Failed to create fromSystem: %v", err)
	}
	toSystem, err := createTestSystem(app, "Alpha Centauri", 10, 10)
	if err != nil {
		t.Fatalf("Failed to create toSystem: %v", err)
	}

	t.Run("successful fleet_move order", func(t *testing.T) {
		// Create fleet
		fleet, err := createTestFleet(app, "Test Fleet", user.Id, fromSystem.Id)
		if err != nil {
			t.Fatalf("Failed to create fleet: %v", err)
		}

		// Authenticate user
		token, err := app.Dao().NewRecordAuthToken(user, nil)
		if err != nil {
			t.Fatalf("Failed to create auth token: %v", err)
		}

		payload := map[string]interface{}{
			"from_id": fromSystem.Id,
			"to_id":   toSystem.Id,
			// fleet_id is not sent; handler picks available fleet
		}
		jsonData, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/orders/fleet", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", token) 
		
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		orderID, ok := response["order_id"].(string)
		if !ok || orderID == "" {
			t.Fatalf("Response does not contain valid order_id: %v", response)
		}

		order, err := app.Dao().FindRecordById("fleet_orders", orderID) // Check fleet_orders
		if err != nil {
			t.Fatalf("Failed to fetch created fleet_order: %v", err)
		}

		if order.GetString("user_id") != user.Id {
			t.Errorf("Expected order.user_id %s, got %s", user.Id, order.GetString("user_id"))
		}
		if order.GetString("type") != "move" { // Type is "move"
			t.Errorf("Expected order.type 'move', got %s", order.GetString("type"))
		}
		if order.GetString("fleet_id") != fleet.Id {
			t.Errorf("Expected order.fleet_id %s, got %s", fleet.Id, order.GetString("fleet_id"))
		}
		if order.GetString("status") != "pending" {
			t.Errorf("Expected order.status 'pending', got %s", order.GetString("status"))
		}

		// Verify data field
		orderData, ok := order.Get("data").(map[string]interface{})
		if !ok {
			t.Fatalf("Order data field is not a map[string]interface{} or is missing")
		}
		if orderData["destination_system_id"] != toSystem.Id {
			t.Errorf("Expected data.destination_system_id %s, got %v", toSystem.Id, orderData["destination_system_id"])
		}
		if orderData["original_system_id"] != fromSystem.Id {
			t.Errorf("Expected data.original_system_id %s, got %v", fromSystem.Id, orderData["original_system_id"])
		}
		if orderData["travel_time_ticks"] != int64(12) { // Note: JSON numbers might be float64, cast to int64 for comparison if needed
			// Handle potential float64 from JSON unmarshalling
			travelTime, ok := orderData["travel_time_ticks"].(float64)
			if !ok || int64(travelTime) != int64(12) {
				t.Errorf("Expected data.travel_time_ticks 12, got %v (type %T)", orderData["travel_time_ticks"], orderData["travel_time_ticks"])
			}
		}
		
		executeAtTick := order.GetInt("execute_at_tick")
		// Assuming current tick is small (e.g., 1-3) for a new test app instance.
		// Travel duration is 12 ticks.
		expectedMinTick := int64(1 + 12) 
		expectedMaxTick := int64(5 + 12) // Allow some leeway for initial ticks if any
		if executeAtTick < expectedMinTick || executeAtTick > expectedMaxTick { 
			t.Errorf("Expected execute_at_tick to be current_tick + 12 (approx %d-%d), got %d", expectedMinTick, expectedMaxTick, executeAtTick)
		}
		
		// Cleanup: Delete the created fleet to not interfere with the next test case
		if err := app.Dao().DeleteRecord(fleet); err != nil {
			t.Logf("Warning: Failed to delete fleet %s after test: %v", fleet.Id, err)
		}
	})

	t.Run("sendFleet with no available fleet", func(t *testing.T) {
		// Ensure no fleets exist for the user at fromSystem by this point.
		// Fleets created in other subtests should be cleaned up or specific to them.

		token, err := app.Dao().NewRecordAuthToken(user, nil)
		if err != nil {
			t.Fatalf("Failed to create auth token: %v", err)
		}
		
		payload := map[string]interface{}{
			"from_id": fromSystem.Id,
			"to_id":   toSystem.Id,
		}
		jsonData, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/orders/fleet", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", token)
		
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest { // Expecting 400
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}
		
		message, _ := response["message"].(string)
		if message == "" && response["data"] != nil { // PocketBase often nests error under data.message
			dataMap, _ := response["data"].(map[string]interface{})
			message, _ = dataMap["message"].(string)
		}

		expectedErrorMsg := "No available fleets at source system" 
		// This might vary based on exact error message in api.go
		// For now, checking if it's non-empty or contains a keyword.
		if message == "" || message != expectedErrorMsg { // Simplified check
			t.Errorf("Expected error message containing '%s', got '%s'", expectedErrorMsg, message)
		}
	})
}


func TestQueueBuildingAPI(t *testing.T) {
	app, cleanup := tests.NewTestApp(t)
	defer cleanup()

	pkg.RegisterAPIRoutes(app)

	user, err := createTestUser(app, "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	
	// Give user some starting credits (directly manipulate user record for test setup)
	user.Set("credits", 1000) 
	if err := app.Dao().SaveRecord(user); err != nil {
		t.Fatalf("Failed to set user credits: %v", err)
	}


	system, err := createTestSystem(app, "Sol", 0, 0)
	if err != nil {
		t.Fatalf("Failed to create system: %v", err)
	}
	planet, err := createTestPlanet(app, "Earth", system.Id, user.Id) // Planet owned by user
	if err != nil {
		t.Fatalf("Failed to create planet: %v", err)
	}
	buildingType, err := createTestBuildingType(app, "Test Mine", 100, 60) // Cost 100, Build time 60s
	if err != nil {
		t.Fatalf("Failed to create building type: %v", err)
	}
	
	token, err := app.Dao().NewRecordAuthToken(user, nil)
	if err != nil {
		t.Fatalf("Failed to create auth token: %v", err)
	}

	t.Run("successful building_construct order", func(t *testing.T) {
		payload := map[string]interface{}{
			"planet_id":     planet.Id,
			"building_type": buildingType.Id,
		}
		jsonData, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/orders/build", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", token)
		
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)
		buildingID, _ := response["building_id"].(string)
		if buildingID == "" {
			t.Fatalf("Response does not contain building_id: %v", response)
		}

		// Verify building was created directly
		building, err := app.Dao().FindRecordById("buildings", buildingID)
		if err != nil {
			t.Fatalf("Failed to fetch created building: %v", err)
		}
		if building.GetString("planet_id") != planet.Id {
			t.Errorf("Expected building.planet_id %s, got %s", planet.Id, building.GetString("planet_id"))
		}
		if building.GetString("building_type") != buildingType.Id { // building_type stores the ID
			t.Errorf("Expected building.building_type %s, got %s", buildingType.Id, building.GetString("building_type"))
		}
		if building.GetInt("level") != 1 {
			t.Errorf("Expected building.level 1, got %d", building.GetInt("level"))
		}
		if !building.GetBool("active") {
			t.Error("Expected building.active to be true")
		}
		
		// Verify credits deducted
		updatedUser, _ := app.Dao().FindRecordById("_users", user.Id)
		initialCredits := 1000
		buildingCost := 100 
		expectedCredits := initialCredits - buildingCost
		if updatedUser.GetInt("credits") != expectedCredits { 
			t.Errorf("Expected user credits to be %d, got %d", expectedCredits, updatedUser.GetInt("credits"))
		}
		// Restore credits for next test case
		updatedUser.Set("credits", 1000)
		app.Dao().SaveRecord(updatedUser)
	})

	t.Run("queueBuilding with insufficient resources", func(t *testing.T) {
		// Set user credits to be less than building cost
		currentUser, _ := app.Dao().FindRecordById("_users", user.Id)
		currentUser.Set("credits", 50) // Cost is 100
		app.Dao().SaveRecord(currentUser)

		payload := map[string]interface{}{
			"planet_id":     planet.Id,
			"building_type": buildingType.Id,
		}
		jsonData, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/orders/build", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", token)
		
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest { // Expecting 400
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
		}
		
		// Restore credits
		currentUser.Set("credits", 1000)
		app.Dao().SaveRecord(currentUser)
	})
}

// TODO: Add more tests for edge cases, invalid inputs, etc.
// TODO: Mock tick.GetCurrentTick and tick.GetTickRate for more precise execute_at_tick assertions if possible.
// For now, the tick package uses global vars, which makes direct mocking from here harder without changing its structure.
// We are relying on the test app's behavior which should have a low current tick initially.

// Helper to initialize collections - PocketBase test app doesn't auto-create them based on schema files
// We must ensure collections exist before tests run, or create them in a setup step.
// For this test, assuming collections from `1712858400_create_orders_collection.go` and others exist.
// If not, a setup function would be needed:
// func setupCollections(app *tests.TestApp, t *testing.T) { ... app.Dao().SaveCollection(...) ... }
// and call it in TestMain or at the start of test functions.
// For now, these tests assume collections are present as per existing migrations.

// TestMain could be used for global setup if needed
// func TestMain(m *testing.M) {
// 	app, cleanup, err := tests.NewDisposableTestApp()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	// Global setup: e.g., run migrations or create collections directly
// 	// For example, to ensure "orders" collection exists:
// 	// ordersCollection := &models.Collection{}
// 	// ordersCollection.Name = "orders"
// 	// ... (define schema based on migration) ...
// 	// if err := app.Dao().SaveCollection(ordersCollection); err != nil { ... }
// 	
// 	exitCode := m.Run()
// 	cleanup()
// 	os.Exit(exitCode)
// }

// Note on pocketbase/dbx.Expression:
// The `tick.ProcessPendingOrders` uses `app.Dao().FindRecordsByFilter` which takes a filter string.
// The sorting `execute_at_tick ASC, created ASC` is also a string.
// PocketBase's Go SDK should handle this syntax for SQLite.
// Example: `filter := "status='pending' && execute_at_tick <= 10"`, `sort := "execute_at_tick asc, created asc"`
// This seems correct based on PocketBase docs.
// `tick.GetTickRate()` is a global var in the tick package, so it will be 6 unless changed.
// `tick.GetCurrentTick(app)` reads a global var `currentTick` from the tick package.
// In a test environment, this `currentTick` starts at 1 and increments if `tick.ProcessTick` is called.
// The API handlers call `tick.GetCurrentTick(app)` so they will get the current value from the tick package.
// For `execute_at_tick` calculation:
// - Fleet move: `currentTick + 12` (since 2 min * 6 ticks/min = 12 ticks)
// - Building construct: `currentTick + (buildingType.GetInt("build_time") / (60 / tick.GetTickRate()))`
//   If build_time = 60s, tickRate = 6 (10s/tick) -> 60s / 10s/tick = 6 ticks.
// The assertions for `execute_at_tick` are kept a bit loose (e.g. `> 12 && < 15`) because the exact `currentTick`
// at the moment of API call inside the test app's webserver isn't precisely known without deeper mocking.
// It should be very small, though (1, 2, or 3 typically).
