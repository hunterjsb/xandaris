package tick_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tests"
	
	"github.com/hunterjsb/xandaris/internal/tick"
)

// Helper functions (similar to api_test.go, but can be simplified or specific for tick tests)
func createTestUserTick(app tests.TestApp, id string) (*models.Record, error) {
	user := &models.Record{}
	user.Id = id // Use specific ID for easier reference if needed
	user.RefreshId() // ensure ID is set if not provided
	user.SetEmail(fmt.Sprintf("user_%s@example.com", user.Id))
	user.SetPassword("password123")
	user.SetVerified(true)
	user.Collection().Name = "_users"
	return user, app.Dao().SaveRecord(user)
}

func createTestSystemTick(app tests.TestApp, name string) (*models.Record, error) {
	collection, err := app.Dao().FindCollectionByNameOrId("systems")
	if err != nil {
		return nil, err
	}
	record := models.NewRecord(collection)
	record.Set("name", name)
	record.Set("x", 0) // Position doesn't matter much for these tests
	record.Set("y", 0)
	return record, app.Dao().SaveRecord(record)
}

func createTestPlanetTick(app tests.TestApp, name string, systemID string, ownerID string) (*models.Record, error) {
	collection, err := app.Dao().FindCollectionByNameOrId("planets")
	if err != nil {
		return nil, err
	}
	record := models.NewRecord(collection)
	record.Set("name", name)
	record.Set("system_id", systemID)
	if ownerID != "" {
		record.Set("colonized_by", ownerID)
	}
	return record, app.Dao().SaveRecord(record)
}

func createTestFleetTick(app tests.TestApp, name string, ownerID string, currentSystemID string) (*models.Record, error) {
	collection, err := app.Dao().FindCollectionByNameOrId("fleets")
	if err != nil {
		return nil, err
	}
	record := models.NewRecord(collection)
	record.Set("name", name)
	record.Set("owner_id", ownerID)
	record.Set("current_system", currentSystemID)
	record.Set("destination_system", "")
	record.Set("eta", nil)
	record.Set("next_stop", "")
	return record, app.Dao().SaveRecord(record)
}

func createTestBuildingTypeTick(app tests.TestApp, name string, buildTime int) (*models.Record, error) {
	collection, err := app.Dao().FindCollectionByNameOrId("building_types")
	if err != nil {
		return nil, err
	}
	record := models.NewRecord(collection)
	record.Set("name", name)
	record.Set("cost", 10) // Cost doesn't matter for ProcessPendingOrders tests
	record.Set("build_time", buildTime) // in seconds
	return record, app.Dao().SaveRecord(record)
}

// createTestFleetOrder creates a record in the "fleet_orders" collection.
func createTestFleetOrder(app tests.TestApp, userID, fleetID, originalSystemID, destSystemID string, orderType string, status string, executeAtTick int64, travelTimeTicks int64) (*models.Record, error) {
	collection, err := app.Dao().FindCollectionByNameOrId("fleet_orders")
	if err != nil {
		return nil, fmt.Errorf("failed to find 'fleet_orders' collection: %w", err)
	}
	order := models.NewRecord(collection)
	order.Set("user_id", userID)
	order.Set("fleet_id", fleetID)
	order.Set("type", orderType) // Should be "move"
	order.Set("status", status)
	order.Set("execute_at_tick", executeAtTick)
	order.Set("original_system_id", originalSystemID)
	order.Set("destination_system_id", destSystemID)
	order.Set("travel_time_ticks", travelTimeTicks)
	
	return order, app.Dao().SaveRecord(order)
}


func TestProcessPendingFleetOrders(t *testing.T) { // Renamed
	app, cleanup := tests.NewTestApp(t)
	defer cleanup()
	
	user, _ := createTestUserTick(app, "user1")
	fromSystem, _ := createTestSystemTick(app, "Sol")
	toSystem, _ := createTestSystemTick(app, "Sirius")
	fleet, _ := createTestFleetTick(app, "Explorer", user.Id, fromSystem.Id)

	t.Run("process due fleet_move order", func(t *testing.T) {
		currentTestTick := int64(10)
		// Use createTestFleetOrder, type is "move", provide travel_time_ticks
		order, err := createTestFleetOrder(app, user.Id, fleet.Id, fromSystem.Id, toSystem.Id, "move", "pending", currentTestTick, 12)
		if err != nil {
			t.Fatalf("Failed to create test fleet order: %v", err)
		}

		err = tick.ProcessPendingFleetOrders(app, currentTestTick) // Call renamed function
		if err != nil {
			t.Fatalf("ProcessPendingFleetOrders failed: %v", err)
		}

		updatedOrder, _ := app.Dao().FindRecordById("fleet_orders", order.Id) // Check fleet_orders
		if updatedOrder.GetString("status") != "completed" {
			t.Errorf("Expected order status 'completed', got '%s'", updatedOrder.GetString("status"))
		}

		updatedFleet, _ := app.Dao().FindRecordById("fleets", fleet.Id)
		// Verify fleet moved to destination system
		if updatedFleet.GetString("current_system") != toSystem.Id { 
			t.Errorf("Expected fleet at system '%s', got '%s'", toSystem.Id, updatedFleet.GetString("current_system"))
		}
		// Legacy movement fields have been removed from fleets table
		// Fleet movement is now managed entirely by fleet_orders
		
		// Cleanup: Reset fleet location for other tests if necessary, or delete order
		updatedFleet.Set("current_system", fromSystem.Id) 
		app.Dao().SaveRecord(updatedFleet)
		app.Dao().DeleteRecord(updatedOrder)
	})

	t.Run("fleet_move order for non-existent fleet", func(t *testing.T) {
		currentTestTick := int64(15)
		order, _ := createTestFleetOrder(app, user.Id, "nonexistentfleet", fromSystem.Id, toSystem.Id, "move", "pending", currentTestTick, 12)
		
		tick.ProcessPendingFleetOrders(app, currentTestTick)

		updatedOrder, _ := app.Dao().FindRecordById("fleet_orders", order.Id)
		if updatedOrder.GetString("status") != "failed" {
			t.Errorf("Expected order status 'failed', got '%s'", updatedOrder.GetString("status"))
		}
		// Error details are now in logs, not stored in data field
		// Just verify the order failed
		app.Dao().DeleteRecord(updatedOrder)
	})

	t.Run("fleet_move order not yet due", func(t *testing.T) {
		currentTestTick := int64(5)
		futureTick := int64(25)
		order, _ := createTestFleetOrder(app, user.Id, fleet.Id, fromSystem.Id, toSystem.Id, "move", "pending", futureTick, 12)

		tick.ProcessPendingFleetOrders(app, currentTestTick) 

		updatedOrder, _ := app.Dao().FindRecordById("fleet_orders", order.Id)
		if updatedOrder.GetString("status") != "pending" {
			t.Errorf("Expected order status 'pending', got '%s'", updatedOrder.GetString("status"))
		}

		updatedFleet, _ := app.Dao().FindRecordById("fleets", fleet.Id)
		if updatedFleet.GetString("current_system") != fromSystem.Id { 
			t.Errorf("Expected fleet to be at fromSystem '%s', got '%s'", fromSystem.Id, updatedFleet.GetString("current_system"))
		}
		app.Dao().DeleteRecord(updatedOrder)
	})
	
	t.Run("fleet_move order already processing", func(t *testing.T) {
		currentTestTick := int64(30)
		order, _ := createTestFleetOrder(app, user.Id, fleet.Id, fromSystem.Id, toSystem.Id, "move", "processing", currentTestTick, 12)
		
		originalFleet, _ := app.Dao().FindRecordById("fleets", fleet.Id) 

		tick.ProcessPendingFleetOrders(app, currentTestTick)

		updatedOrder, _ := app.Dao().FindRecordById("fleet_orders", order.Id)
		if updatedOrder.GetString("status") != "processing" { 
			t.Errorf("Expected order status 'processing', got '%s'", updatedOrder.GetString("status"))
		}
		
		currentFleetState, _ := app.Dao().FindRecordById("fleets", fleet.Id)
		if currentFleetState.GetString("current_system") != originalFleet.GetString("current_system") {
			t.Errorf("Fleet location should not change for an order already in 'processing' by this function alone.")
		}
		app.Dao().DeleteRecord(updatedOrder)
	})

	t.Run("fleet_move order with invalid data field (missing destination_system_id)", func(t *testing.T) {
		currentTestTick := int64(35)
		// Create an order with a data field that's missing destination_system_id
		orderCollection, _ := app.Dao().FindCollectionByNameOrId("fleet_orders")
		order := models.NewRecord(orderCollection)
		order.Set("user_id", user.Id)
		order.Set("fleet_id", fleet.Id)
		order.Set("type", "move")
		order.Set("status", "pending")
		order.Set("execute_at_tick", currentTestTick)
		order.Set("original_system_id", fromSystem.Id)
		order.Set("travel_time_ticks", 12)
		// Missing destination_system_id field
		app.Dao().SaveRecord(order)

		tick.ProcessPendingFleetOrders(app, currentTestTick)

		updatedOrder, _ := app.Dao().FindRecordById("fleet_orders", order.Id)
		if updatedOrder.GetString("status") != "failed" {
			t.Errorf("Expected order status 'failed' for invalid data, got '%s'", updatedOrder.GetString("status"))
		}
		// Error details are now in logs, not stored in data field
		// Just verify the order failed
		app.Dao().DeleteRecord(updatedOrder)
	})
}

// Note: TestProcessPendingOrdersBuildingConstruct has been removed as building orders are deferred.

// Note: These tests assume that collections (systems, planets, fleets, building_types, fleet_orders)
// are properly defined in the schema and accessible via the TestApp's DAO.
// If migrations are not automatically run by tests.NewTestApp(), they would need to be
// applied in a TestMain or per-test setup. PocketBase TestApp typically uses an in-memory DB
// and applies migrations found in the `pb_migrations` directory by default.
// Ensure your project structure and test setup align with this.
// The global tick counter in the `tick` package is a slight concern for fully isolated parallel tests in the future,
// but for sequential execution or simple cases, it should be manageable.
// The `currentTestTick` variable passed to `ProcessPendingOrders` is the key driver for "due" logic.
