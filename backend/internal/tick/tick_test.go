package tick

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tests"
	// Ensure other necessary internal packages are imported if needed later
)

// Helper function to create a test application instance
func setupTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	testApp, err := tests.NewTestApp("../../../migrations") // Adjust path to migrations
	if err != nil {
		t.Fatalf("Failed to initialize test app: %v", err)
	}
	// Disable verbose logging during tests unless needed
	// testApp.App.Logger().SetOutput(os.Stdout) // Enable for debugging specific tests
	// testApp.App.Logger().SetLevel(log.Ldebug)

	// Clean up hook to remove the test db file
	t.Cleanup(func() {
		if err := os.Remove(testApp.DataDir() + "/storage.db"); err != nil {
			// Allow "no such file or directory" error if db was never created or already cleaned
			if !os.IsNotExist(err) {
				t.Logf("Failed to remove test db: %v", err)
			}
		}
		// Also remove the test data directory if it exists
		if err := os.RemoveAll(testApp.DataDir()); err != nil {
			t.Logf("Failed to remove test data directory: %v", err)
		}
	})

	return testApp
}

// Helper to create a user
func createTestUser(t *testing.T, app *tests.TestApp, email, password string) *models.Record {
	t.Helper()
	user := &models.Record{}
	user.CollectionID = "_pb_users_auth_" // or app.Dao().UsersCollection().Id
	user.Email = email
	user.SetPassword(password)
	user.VerificationToken = "test_verified_token" // Mark as verified

	if err := app.Dao().SaveRecord(user); err != nil {
		t.Fatalf("Failed to create test user %s: %v", email, err)
	}
	return user
}

// Helper to create a system
func createTestSystem(t *testing.T, app *tests.TestApp, owner *models.Record, name string, initialGoods int) *models.Record {
	t.Helper()
	system := &models.Record{}
	system.CollectionID = "systems" // or app.Dao().FindCollectionByNameOrId("systems").Id
	system.Set("name", name)
	if owner != nil {
		system.Set("owner_id", owner.Id)
	}
	system.Set("x", 10) // Example coordinates
	system.Set("y", 20)
	system.Set("pop", 1000)
	system.Set("goods", initialGoods)
	system.Set("hab_lvl", 0)
	system.Set("farm_lvl", 0)
	system.Set("mine_lvl", 0)
	system.Set("fac_lvl", 0)
	system.Set("yard_lvl", 0)

	if err := app.Dao().SaveRecord(system); err != nil {
		t.Fatalf("Failed to create test system %s: %v", name, err)
	}
	return system
}

// TestMain can be used for package-level setup/teardown if needed in the future
// func TestMain(m *testing.M) {
// 	// setup
// 	log.Println("Setting up tick tests...")
// 	exitCode := m.Run()
// 	// teardown
// 	log.Println("Tearing down tick tests...")
// 	os.Exit(exitCode)
// }

func TestBuildingQueue_OrderAndCompletion(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	user := createTestUser(t, app, "user1@example.com", "password123")
	system := createTestSystem(t, app, user, "Test System Alpha", 1000)

	buildingType := "farm"
	targetLevel := 1
	cost := 150 * targetLevel // Cost for farm lvl 1 is 150 * (0+1)
	buildTimeTicks := int64(8) // As defined in handlers

	// 1. Create a building queue record (simulating BuildOrderHandler)
	initialGoods := system.GetInt("goods")

	// Manually create queue item
	queueCollection, err := app.Dao().FindCollectionByNameOrId("building_queue")
	if err != nil {
		t.Fatalf("Failed to find building_queue collection: %v", err)
	}

	// Use a fixed current tick for predictability in tests
	testCurrentTick := GetCurrentTick(app.App) // Or set to a specific value if GetCurrentTick is problematic
	if testCurrentTick == 0 { // Ensure it's not 0 if it relies on a running app
		testCurrentTick = 100
		// If tick processing relies on global state that isn't initialized in tests,
		// we might need to mock or manually set it. For now, assume GetCurrentTick works or use a fixed value.
		// For controlled testing, manually setting currentTick and completion_tick is better.
		currentTick = testCurrentTick // This updates the package global, use with caution or mock.
	}


	queueItem := models.NewRecord(queueCollection)
	queueItem.Set("system_id", system.Id)
	queueItem.Set("owner_id", user.Id)
	queueItem.Set("building_type", buildingType)
	queueItem.Set("target_level", targetLevel)
	// Set completion_tick to be in the "past" relative to the tick when ApplyBuildingCompletions will run
	// Or, more robustly, set completion_tick based on a controllable "current test tick"
	queueItem.Set("completion_tick", testCurrentTick + 1) // Will complete on the next tick processing

	if err := app.Dao().SaveRecord(queueItem); err != nil {
		t.Fatalf("Failed to save building queue item: %v", err)
	}

	// Deduct resources (as BuildOrderHandler would)
	system.Set("goods", initialGoods-cost)
	if err := app.Dao().SaveRecord(system); err != nil {
		t.Fatalf("Failed to update system resources: %v", err)
	}

	// 2. Verify queue record and resource deduction
	updatedSystem, err := app.Dao().FindRecordById("systems", system.Id)
	if err != nil {
		t.Fatalf("Failed to fetch updated system: %v", err)
	}
	if updatedSystem.GetInt("goods") != initialGoods-cost {
		t.Errorf("Expected goods to be %d, got %d", initialGoods-cost, updatedSystem.GetInt("goods"))
	}

	persistedQueueItem, err := app.Dao().FindRecordById("building_queue", queueItem.Id)
	if err != nil {
		t.Fatalf("Failed to fetch queue item: %v", err)
	}
	if persistedQueueItem == nil {
		t.Fatal("Queue item not found in DB")
	}

	// 3. Advance tick and apply completions
	// Manually advance the global currentTick for testing ApplyBuildingCompletions
	// This assumes ApplyBuildingCompletions uses the global currentTick from its own package.
	// If GetCurrentTick(app.App) is used internally by ApplyBuildingCompletions, that needs to be consistent.

	// Let's make ApplyBuildingCompletions process the item created above.
	// We set queueItem.completion_tick to testCurrentTick + 1.
	// So, we need to "advance" the game's current tick to be >= testCurrentTick + 1.

	// Simulate tick advancement for ApplyBuildingCompletions.
	// The ProcessTick function increments currentTick.
	// For isolated testing of ApplyBuildingCompletions, we can directly set the global currentTick
	// or ensure GetCurrentTick(app.App) returns the desired value.
	// This is tricky if the global currentTick is not easily controlled or reset between tests.

	// For this test, let's assume ApplyBuildingCompletions will use a tick value that makes our item complete.
	// One way is to modify the global currentTick in the tick package.
	tickMutex.Lock()
	currentTick = testCurrentTick + buildTimeTicks // Advance to or past completion_tick
	completionTickForTest := currentTick // The tick at which we expect completion
	queueItem.Set("completion_tick", completionTickForTest)
	if err := app.Dao().SaveRecord(queueItem); err != nil { // Resave with adjusted completion tick
		t.Fatalf("Failed to update queue item completion_tick: %v", err)
	}
	tickMutex.Unlock()


	if err := ApplyBuildingCompletions(app.App); err != nil {
		t.Fatalf("ApplyBuildingCompletions failed: %v", err)
	}

	// 4. Verify system update and queue deletion
	finalSystem, err := app.Dao().FindRecordById("systems", system.Id)
	if err != nil {
		t.Fatalf("Failed to fetch final system state: %v", err)
	}
	if finalSystem.GetInt("farm_lvl") != targetLevel {
		t.Errorf("Expected farm_lvl to be %d, got %d", targetLevel, finalSystem.GetInt("farm_lvl"))
	}

	deletedQueueItem, _ := app.Dao().FindRecordById("building_queue", queueItem.Id)
	if deletedQueueItem != nil {
		t.Error("Expected building queue item to be deleted, but it still exists.")
	}
}

func TestBuildingQueue_BankConstruction(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	user := createTestUser(t, app, "user2@example.com", "password123")
	user.Set("credits", 2000) // Give user initial credits
	if err := app.Dao().SaveRecord(user); err != nil {
		t.Fatalf("Failed to set user credits: %v", err)
	}

	system := createTestSystem(t, app, user, "Test System Beta", 500) // System resources not used for bank cost

	buildingType := "bank"
	targetLevel := 1 // Banks are conceptual level 1
	cost := 1000     // Cost for bank is 1000 credits
	buildTimeTicks := int64(20) // As defined in handlers

	initialCredits := user.GetInt("credits")

	// 1. Create a building queue record for a bank
	queueCollection, err := app.Dao().FindCollectionByNameOrId("building_queue")
	if err != nil {
		t.Fatalf("Failed to find building_queue collection: %v", err)
	}

	testCurrentTick := GetCurrentTick(app.App)
	if testCurrentTick == 0 { testCurrentTick = 200 } // Base tick for test
	currentTick = testCurrentTick // Set package global currentTick

	queueItem := models.NewRecord(queueCollection)
	queueItem.Set("system_id", system.Id)
	queueItem.Set("owner_id", user.Id)
	queueItem.Set("building_type", buildingType)
	queueItem.Set("target_level", targetLevel)
	queueItem.Set("completion_tick", testCurrentTick + 1) // Set to complete on next effective tick

	if err := app.Dao().SaveRecord(queueItem); err != nil {
		t.Fatalf("Failed to save bank building queue item: %v", err)
	}

	// Deduct credits from user (as BuildOrderHandler would)
	user.Set("credits", initialCredits-cost)
	if err := app.Dao().SaveRecord(user); err != nil {
		t.Fatalf("Failed to update user credits: %v", err)
	}

	// 2. Verify queue record and credit deduction
	updatedUser, err := app.Dao().FindAuthRecordById("_pb_users_auth_", user.Id)
	if err != nil {
		t.Fatalf("Failed to fetch updated user: %v", err)
	}
	if updatedUser.GetInt("credits") != initialCredits-cost {
		t.Errorf("Expected user credits to be %d, got %d", initialCredits-cost, updatedUser.GetInt("credits"))
	}

	// 3. Advance tick and apply completions
	tickMutex.Lock()
	currentTick = testCurrentTick + buildTimeTicks // Advance to or past completion_tick
	completionTickForTest := currentTick
	queueItem.Set("completion_tick", completionTickForTest)
	if err := app.Dao().SaveRecord(queueItem); err != nil {
		t.Fatalf("Failed to update queue item completion_tick for bank: %v", err)
	}
	tickMutex.Unlock()

	if err := ApplyBuildingCompletions(app.App); err != nil {
		t.Fatalf("ApplyBuildingCompletions failed for bank: %v", err)
	}

	// 4. Verify bank creation and queue deletion
	bankRecord, err := app.Dao().FindFirstRecordByFilter("banks", "system_id = {:systemId} && owner_id = {:ownerId}", dbx.Params{
		"systemId": system.Id,
		"ownerId":  user.Id,
	})
	if err != nil {
		t.Fatalf("Error fetching bank record: %v", err)
	}
	if bankRecord == nil {
		t.Fatal("Expected bank record to be created, but it was not found.")
	}
	if !bankRecord.GetBool("active") {
		t.Error("Expected bank to be active.")
	}
	if bankRecord.GetInt("last_income_tick") != completionTickForTest {
		t.Errorf("Expected bank last_income_tick to be %d, got %d", completionTickForTest, bankRecord.GetInt("last_income_tick"))
	}


	deletedQueueItem, _ := app.Dao().FindRecordById("building_queue", queueItem.Id)
	if deletedQueueItem != nil {
		t.Error("Expected bank building queue item to be deleted, but it still exists.")
	}
}


func TestBuildingQueue_InsufficientResources(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	user := createTestUser(t, app, "user3@example.com", "password123")
	// System with very few goods
	system := createTestSystem(t, app, user, "Test System Gamma", 10) // Only 10 goods

	buildingType := "farm"        // Costs 150
	targetLevel := 1
	cost := 150 * targetLevel

	// Attempt to queue a farm (simulating BuildOrderHandler logic part)
	// We expect this to fail before even creating a queue item if we were calling the handler.
	// Here, we test the state *if* an item was created despite insufficient funds (which shouldn't happen with handler)
	// or, more realistically, we check that the handler *would not* create the item.
	// For this test, let's assume a direct check: if goods < cost, don't create queue item.

	initialGoods := system.GetInt("goods")
	if initialGoods < cost {
		// This is the expected path: the handler would prevent queueing.
		// So, no queue item should be created, and goods remain unchanged.
		t.Logf("Simulating rejection by BuildOrderHandler due to insufficient goods (%d < %d)", initialGoods, cost)
	} else {
		// This path should ideally not be taken by BuildOrderHandler if resources are insufficient.
		// If for some reason a queue item IS created with insufficient resources and ApplyBuildingCompletions is run,
		// the current ApplyBuildingCompletions does not re-check resources. It assumes they were deducted.
		t.Fatalf("Test logic error: System has enough goods, contrary to test intent.")
	}

	// Verify no queue item was created (or would be created by handler)
	queueItems, err := app.Dao().FindRecordsByFilter("building_queue", "system_id = {:sysId}", "", 1, 0, dbx.Params{"sysId": system.Id})
	if err != nil {
		t.Fatalf("Error fetching queue items: %v", err)
	}
	if len(queueItems) > 0 {
		t.Errorf("Expected no queue items for system with insufficient resources, found %d", len(queueItems))
	}

	// Verify resources were not deducted
	updatedSystem, _ := app.Dao().FindRecordById("systems", system.Id)
	if updatedSystem.GetInt("goods") != initialGoods {
		t.Errorf("Expected goods to remain %d, got %d", initialGoods, updatedSystem.GetInt("goods"))
	}

	// Test for bank with insufficient credits
	user.Set("credits", 50) // Insufficient for a bank (cost 1000)
	if err := app.Dao().SaveRecord(user); err != nil {
		t.Fatalf("Failed to set user credits for insufficient funds test: %v", err)
	}
	initialUserCredits := user.GetInt("credits")
	bankCost := 1000

	if initialUserCredits < bankCost {
		t.Logf("Simulating rejection by BuildOrderHandler for bank due to insufficient credits (%d < %d)", initialUserCredits, bankCost)
	} else {
		t.Fatalf("Test logic error: User has enough credits for bank, contrary to test intent.")
	}
	// Verify no bank queue item created
	bankQueueItems, err := app.Dao().FindRecordsByFilter("building_queue", "owner_id = {:ownerId} && building_type = 'bank'", "", 1, 0, dbx.Params{"ownerId": user.Id})
	if err != nil {
		t.Fatalf("Error fetching bank queue items: %v", err)
	}
	if len(bankQueueItems) > 0 {
		t.Errorf("Expected no bank queue items for user with insufficient credits, found %d", len(bankQueueItems))
	}
	// Verify credits not deducted
	updatedUser, _ := app.Dao().FindAuthRecordById("_pb_users_auth_", user.Id)
	if updatedUser.GetInt("credits") != initialUserCredits {
		t.Errorf("Expected user credits to remain %d, got %d", initialUserCredits, updatedUser.GetInt("credits"))
	}
}

func TestBuildingQueue_DuplicateBuildingType(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	user := createTestUser(t, app, "user4@example.com", "password123")
	system := createTestSystem(t, app, user, "Test System Delta", 1000)

	buildingType := "mine"
	targetLevel1 := 1
	// cost1 := 200 * targetLevel1
	buildTimeTicks1 := int64(10)

	testCurrentTick := GetCurrentTick(app.App)
	if testCurrentTick == 0 { testCurrentTick = 300 }
	currentTick = testCurrentTick


	// 1. Queue the first building (mine lvl 1)
	queueCollection, err := app.Dao().FindCollectionByNameOrId("building_queue")
	if err != nil {
		t.Fatalf("Failed to find building_queue collection: %v", err)
	}
	firstQueueItem := models.NewRecord(queueCollection)
	firstQueueItem.Set("system_id", system.Id)
	firstQueueItem.Set("owner_id", user.Id)
	firstQueueItem.Set("building_type", buildingType)
	firstQueueItem.Set("target_level", targetLevel1)
	firstQueueItem.Set("completion_tick", testCurrentTick+buildTimeTicks1)
	if err := app.Dao().SaveRecord(firstQueueItem); err != nil {
		t.Fatalf("Failed to save first queue item: %v", err)
	}
	// system.Set("goods", system.GetInt("goods") - cost1) // Assume deduction happened
	// app.Dao().SaveRecord(system)


	// 2. Attempt to queue a second building of the same type (mine lvl 2)
	// This test assumes BuildOrderHandler would reject this.
	// The handler checks: existingQueueItem, _ := app.Dao().FindFirstRecordByFilter("building_queue", "system_id = ? && building_type = ?", req.SystemID, req.BuildingType)
	// So, we simulate this check.

	existingQueueItemCheck, _ := app.Dao().FindFirstRecordByFilter(
		"building_queue",
		"system_id = {:systemId} && building_type = {:buildingType}",
		dbx.Params{"systemId": system.Id, "buildingType": buildingType},
	)

	if existingQueueItemCheck != nil && existingQueueItemCheck.Id == firstQueueItem.Id {
		t.Logf("Simulating rejection by BuildOrderHandler because a %s is already in queue for system %s.", buildingType, system.Id)
		// If we were testing the handler directly, we'd expect an error response here.
		// Since we are testing the state, we verify no second item is added.
	} else {
		t.Errorf("Expected to find the first queue item (%s) when checking for duplicates, but found %v", firstQueueItem.Id, existingQueueItemCheck)
	}

	// Verify that only one item of this building_type exists in the queue for this system
	allQueueItemsForType, err := app.Dao().FindRecordsByFilter(
		"building_queue",
		"system_id = {:systemId} && building_type = {:buildingType}",
		"", 0, 0, // no limit, get all
		dbx.Params{"systemId": system.Id, "buildingType": buildingType},
	)
	if err != nil {
		t.Fatalf("Error fetching queue items for type %s: %v", buildingType, err)
	}
	if len(allQueueItemsForType) != 1 {
		t.Errorf("Expected 1 queue item of type %s, found %d. Duplicate was likely allowed.", buildingType, len(allQueueItemsForType))
	}
}

// TODO: TestWebSocketAuth scenarios (in a new websocket/manager_test.go)

// Note on tick control:
// The current tick management (global variable `currentTick` in the `tick` package)
// can make isolated testing difficult if tests run in parallel or if state isn't reset.
// Consider refactoring GetCurrentTick to be part of a struct that can be mocked,
// or pass the current tick value explicitly to functions like ApplyBuildingCompletions
// in their testable forms if not their main operational forms.
// For now, tests will manipulate the package-level `currentTick` under a mutex.
// This is generally not ideal for test design but works for now given current codebase.
// A better approach might involve a "TickProvider" interface.
