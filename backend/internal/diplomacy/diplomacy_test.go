package diplomacy

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

var testUsersCollection *models.Collection // To store the ID

// Helper to create collections for testing
func setupDiplomacyCollections(t *testing.T, app *tests.TestApp) {
	// Users Collection (Auth type)
	users := &models.Collection{
		Name: "users",
		Type: models.CollectionTypeAuth,
		Schema: schema.NewSchema(
			&schema.SchemaField{Name: "username", Type: schema.FieldTypeText, Required: true, System: false, Unique: true},
			&schema.SchemaField{Name: "email", Type: schema.FieldTypeEmail, Required: true, System: false, Unique: true},
			&schema.SchemaField{Name: "emailVisibility", Type: schema.FieldTypeBool, System: false},
			&schema.SchemaField{Name: "verified", Type: schema.FieldTypeBool, System: false},
			// avatar, name etc. can be added if needed by specific tests
		),
	}
	// For auth collections, you might need to set specific options if not using defaults
	users.AuthOptions = models.CollectionAuthOptions{
		ManageRule: nil, // allow all for tests
		AllowOAuth2Auth: true,
		AllowUsernameAuth: true,
		AllowEmailAuth: true,
		MinPasswordLength: 5,
	}

	if err := app.Dao().SaveCollection(users); err != nil {
		t.Fatalf("Failed to create users collection: %v", err)
	}
	testUsersCollection = users // Store for later use, e.g. getting its ID

	// Diplomatic Relations Collection
	relations := &models.Collection{
		Name: "diplomatic_relations",
		Type: models.CollectionTypeBase,
		Schema: schema.NewSchema(
			&schema.SchemaField{
				Name:     "player1_id",
				Type:     schema.FieldTypeRelation,
				Required: true,
				Options:  &schema.RelationOptions{CollectionId: testUsersCollection.Id, MinSelect: types.Pointer(1), MaxSelect: types.Pointer(1)},
			},
			&schema.SchemaField{
				Name:     "player2_id",
				Type:     schema.FieldTypeRelation,
				Required: true,
				Options:  &schema.RelationOptions{CollectionId: testUsersCollection.Id, MinSelect: types.Pointer(1), MaxSelect: types.Pointer(1)},
			},
			&schema.SchemaField{Name: "status", Type: schema.FieldTypeText, Required: true},
			&schema.SchemaField{Name: "start_date", Type: schema.FieldTypeDate, Required: true},
			&schema.SchemaField{Name: "duration_ticks", Type: schema.FieldTypeNumber, Required: false},
			&schema.SchemaField{Name: "end_date", Type: schema.FieldTypeDate, Required: false},
		),
		Indexes: types.JsonArray[string]{"CREATE UNIQUE INDEX idx_diplomatic_relations_players ON {{.Name}} (player1_id, player2_id)"},
	}
	if err := app.Dao().SaveCollection(relations); err != nil {
		t.Fatalf("Failed to create diplomatic_relations collection: %v", err)
	}

	// Diplomatic Proposals Collection
	proposals := &models.Collection{
		Name: "diplomatic_proposals",
		Type: models.CollectionTypeBase,
		Schema: schema.NewSchema(
			&schema.SchemaField{
				Name:     "proposer_id",
				Type:     schema.FieldTypeRelation,
				Required: true,
				Options:  &schema.RelationOptions{CollectionId: testUsersCollection.Id, MinSelect: types.Pointer(1), MaxSelect: types.Pointer(1)},
			},
			&schema.SchemaField{
				Name:     "receiver_id",
				Type:     schema.FieldTypeRelation,
				Required: true,
				Options:  &schema.RelationOptions{CollectionId: testUsersCollection.Id, MinSelect: types.Pointer(1), MaxSelect: types.Pointer(1)},
			},
			&schema.SchemaField{Name: "type", Type: schema.FieldTypeText, Required: true},
			&schema.SchemaField{Name: "terms", Type: schema.FieldTypeJson, Required: false},
			&schema.SchemaField{Name: "status", Type: schema.FieldTypeText, Required: true},
			&schema.SchemaField{Name: "proposed_date", Type: schema.FieldTypeDate, Required: true},
			&schema.SchemaField{Name: "expiration_date", Type: schema.FieldTypeDate, Required: true},
			&schema.SchemaField{Name: "duration_ticks", Type: schema.FieldTypeNumber, Required: false},
		),
	}
	if err := app.Dao().SaveCollection(proposals); err != nil {
		t.Fatalf("Failed to create diplomatic_proposals collection: %v", err)
	}
}

// Helper to create a user for testing
func createTestUser(t *testing.T, app *tests.TestApp, usernameSuffix string) *models.Record {
	// For auth collections, Dao().SaveRecord expects a real password to hash
	// or use NewUser and then Save. For simplicity if not testing auth itself,
	// one might change users collection to Base for tests.
	// Sticking to Auth:
	user := models.NewUser()
	user.Username = "user_" + usernameSuffix
	user.Email = "user_" + usernameSuffix + "@example.com"
	user.SetPassword("password123") // PocketBase will hash this
	user.Verified = true

	if err := app.Dao().SaveUser(user); err != nil {
		t.Fatalf("Failed to create user %s: %v", user.Username, err)
	}
	return user
}

func TestMain(m *testing.M) {
	// Optional: Setup that runs once for all tests in the package
	// For example, initializing a global TestApp instance if tests don't need full isolation.
	// However, for DB state, it's often better to init app per test or per group.
	exitCode := m.Run()
	os.Exit(exitCode)
}


func TestProposeTreaty(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	setupDiplomacyCollections(t, app)

	user1 := createTestUser(t, app, "1")
	user2 := createTestUser(t, app, "2")

	t.Run("valid proposal", func(t *testing.T) {
		terms := `{"gold": 100, "peace_turns": 10}`
		duration := 20 // example game ticks

		proposalRecord, err := ProposeTreaty(app, user1.Id, user2.Id, "alliance_offer", terms, duration)
		if err != nil {
			t.Fatalf("ProposeTreaty failed: %v", err)
		}
		if proposalRecord == nil {
			t.Fatal("Proposal record is nil")
		}

		// Verify fields
		if proposalRecord.GetString("proposer_id") != user1.Id {
			t.Errorf("Expected proposer_id %s, got %s", user1.Id, proposalRecord.GetString("proposer_id"))
		}
		if proposalRecord.GetString("receiver_id") != user2.Id {
			t.Errorf("Expected receiver_id %s, got %s", user2.Id, proposalRecord.GetString("receiver_id"))
		}
		if proposalRecord.GetString("type") != "alliance_offer" {
			t.Errorf("Expected type 'alliance_offer', got %s", proposalRecord.GetString("type"))
		}
		if proposalRecord.GetString("terms") != terms {
			t.Errorf("Expected terms '%s', got '%s'", terms, proposalRecord.GetString("terms"))
		}
		if proposalRecord.GetString("status") != "pending" {
			t.Errorf("Expected status 'pending', got %s", proposalRecord.GetString("status"))
		}
		if proposalRecord.GetInt("duration_ticks") != duration {
			t.Errorf("Expected duration_ticks %d, got %d", duration, proposalRecord.GetInt("duration_ticks"))
		}
		// Check dates are set (rough check, exact time is tricky)
		if proposalRecord.GetDateTime("proposed_date").IsZero() {
			t.Error("proposed_date is zero")
		}
		if proposalRecord.GetDateTime("expiration_date").IsZero() {
			t.Error("expiration_date is zero")
		}
		// Check expiration is after proposed
		if !proposalRecord.GetDateTime("expiration_date").Time().After(proposalRecord.GetDateTime("proposed_date").Time()) {
			t.Error("expiration_date is not after proposed_date")
		}
	})

	t.Run("propose to oneself", func(t *testing.T) {
		_, err := ProposeTreaty(app, user1.Id, user1.Id, "peace_treaty", "", 0)
		if err == nil {
			t.Error("Expected error when proposing to oneself, got nil")
		} else if !strings.Contains(err.Error(), "cannot be the same") {
			t.Errorf("Expected error message about same proposer/receiver, got: %v", err)
		}
	})

	t.Run("empty proposer or receiver ID", func(t *testing.T) {
		_, err := ProposeTreaty(app, "", user2.Id, "peace_treaty", "", 0)
		if err == nil {
			t.Error("Expected error for empty proposerID, got nil")
		}
		_, err = ProposeTreaty(app, user1.Id, "", "peace_treaty", "", 0)
		if err == nil {
			t.Error("Expected error for empty receiverID, got nil")
		}
	})
}

func TestAcceptProposal(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	setupDiplomacyCollections(t, app)

	user1 := createTestUser(t, app, "acc1")
	user2 := createTestUser(t, app, "acc2")
	user3 := createTestUser(t, app, "acc3_other")


	// Scenario: Valid acceptance of an alliance
	t.Run("valid alliance acceptance", func(t *testing.T) {
		proposal, _ := ProposeTreaty(app, user1.Id, user2.Id, "alliance_offer", `{"details":"test"}`, 50)

		err := AcceptProposal(app, proposal.Id, user2.Id)
		if err != nil {
			t.Fatalf("AcceptProposal failed: %v", err)
		}

		// Check proposal status
		updatedProposal, _ := app.Dao().FindRecordById("diplomatic_proposals", proposal.Id)
		if updatedProposal.GetString("status") != "accepted" {
			t.Errorf("Expected proposal status 'accepted', got '%s'", updatedProposal.GetString("status"))
		}

		// Check diplomatic relation
		p1ID, p2ID := orderedPlayerIDs(user1.Id, user2.Id)
		relation, _ := app.Dao().FindFirstRecordByFilter("diplomatic_relations",
			"player1_id={:p1} AND player2_id={:p2}",
			map[string]any{"p1": p1ID, "p2": p2ID})

		if relation == nil {
			t.Fatal("Diplomatic relation not created/found")
		}
		if relation.GetString("status") != "alliance" {
			t.Errorf("Expected relation status 'alliance', got '%s'", relation.GetString("status"))
		}
		if relation.GetInt("duration_ticks") != 50 {
			t.Errorf("Expected duration_ticks 50, got %d", relation.GetInt("duration_ticks"))
		}
		if relation.GetDateTime("end_date").IsZero() { // Assuming 50 ticks means end_date is set
			t.Error("Expected end_date to be set for non-zero duration_ticks")
		}
		if !relation.GetDateTime("start_date").Time().Before(relation.GetDateTime("end_date").Time()) && relation.GetInt("duration_ticks") > 0 {
             t.Errorf("Relation start_date (%v) should be before end_date (%v)", relation.GetDateTime("start_date"), relation.GetDateTime("end_date"))
        }
	})

	// Scenario: Valid acceptance of a peace treaty (should result in "peace" status)
	t.Run("valid peace_treaty acceptance", func(t *testing.T) {
		proposal, _ := ProposeTreaty(app, user1.Id, user2.Id, "peace_treaty", `{}`, 0) // Indefinite peace

		err := AcceptProposal(app, proposal.Id, user2.Id)
		if err != nil {
			t.Fatalf("AcceptProposal for peace_treaty failed: %v", err)
		}
		updatedProposal, _ := app.Dao().FindRecordById("diplomatic_proposals", proposal.Id)
		if updatedProposal.GetString("status") != "accepted" {
			t.Errorf("Expected proposal status 'accepted', got '%s'", updatedProposal.GetString("status"))
		}
		p1ID, p2ID := orderedPlayerIDs(user1.Id, user2.Id)
		relation, _ := app.Dao().FindFirstRecordByFilter("diplomatic_relations",
			"player1_id={:p1} AND player2_id={:p2}",
			map[string]any{"p1": p1ID, "p2": p2ID})
		if relation == nil {
			t.Fatal("Diplomatic relation not created/found for peace_treaty")
		}
		if relation.GetString("status") != "peace" {
			t.Errorf("Expected relation status 'peace', got '%s'", relation.GetString("status"))
		}
		if relation.GetInt("duration_ticks") != 0 {
			t.Errorf("Expected duration_ticks 0 for indefinite peace, got %d", relation.GetInt("duration_ticks"))
		}
		if !relation.GetDateTime("end_date").IsZero() {
			t.Error("Expected end_date to be zero/null for indefinite peace")
		}
	})


	t.Run("accept non-existent proposal", func(t *testing.T) {
		err := AcceptProposal(app, "nonexistentid", user2.Id)
		if err == nil {
			t.Error("Expected error for non-existent proposal, got nil")
		}
	})

	t.Run("accept by wrong user", func(t *testing.T) {
		proposal, _ := ProposeTreaty(app, user1.Id, user2.Id, "alliance_offer", "", 0)
		err := AcceptProposal(app, proposal.Id, user3.Id) // user3 tries to accept
		if err == nil {
			t.Error("Expected error when wrong user accepts, got nil")
		} else if !strings.Contains(err.Error(), "does not match proposal receiverID") {
             t.Errorf("Expected specific error message for wrong acceptor, got: %v", err)
        }
	})

	t.Run("accept already accepted proposal", func(t *testing.T) {
		proposal, _ := ProposeTreaty(app, user1.Id, user2.Id, "alliance_offer", "", 0)
		AcceptProposal(app, proposal.Id, user2.Id) // First acceptance

		err := AcceptProposal(app, proposal.Id, user2.Id) // Second attempt
		if err == nil {
			t.Error("Expected error when accepting already accepted proposal, got nil")
		} else if !strings.Contains(err.Error(), "is not pending") {
             t.Errorf("Expected specific error message for non-pending proposal, got: %v", err)
        }
	})
}


func TestRejectProposal(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	setupDiplomacyCollections(t, app)

	user1 := createTestUser(t, app, "rej1")
	user2 := createTestUser(t, app, "rej2")
	user3 := createTestUser(t, app, "rej3_other")

	t.Run("valid rejection", func(t *testing.T) {
		proposal, _ := ProposeTreaty(app, user1.Id, user2.Id, "alliance_offer", "", 0)

		err := RejectProposal(app, proposal.Id, user2.Id)
		if err != nil {
			t.Fatalf("RejectProposal failed: %v", err)
		}

		updatedProposal, _ := app.Dao().FindRecordById("diplomatic_proposals", proposal.Id)
		if updatedProposal.GetString("status") != "rejected" {
			t.Errorf("Expected proposal status 'rejected', got '%s'", updatedProposal.GetString("status"))
		}

		// Ensure no relation was created
		p1ID, p2ID := orderedPlayerIDs(user1.Id, user2.Id)
		relation, _ := app.Dao().FindFirstRecordByFilter("diplomatic_relations",
			"player1_id={:p1} AND player2_id={:p2}",
			map[string]any{"p1": p1ID, "p2": p2ID})
		if relation != nil {
			t.Error("Diplomatic relation should not be created on rejection")
		}
	})

	t.Run("reject non-existent proposal", func(t *testing.T) {
		err := RejectProposal(app, "nonexistentid", user2.Id)
		if err == nil {
			t.Error("Expected error for non-existent proposal, got nil")
		}
	})

	t.Run("reject by wrong user", func(t *testing.T) {
		proposal, _ := ProposeTreaty(app, user1.Id, user2.Id, "alliance_offer", "", 0)
		err := RejectProposal(app, proposal.Id, user3.Id) // user3 tries to reject
		if err == nil {
			t.Error("Expected error when wrong user rejects, got nil")
		}
	})

	t.Run("reject already rejected proposal", func(t *testing.T) {
		proposal, _ := ProposeTreaty(app, user1.Id, user2.Id, "alliance_offer", "", 0)
		RejectProposal(app, proposal.Id, user2.Id) // First rejection

		err := RejectProposal(app, proposal.Id, user2.Id) // Second attempt
		if err == nil {
			t.Error("Expected error when rejecting already rejected proposal, got nil")
		}
	})
}

func TestDeclareWar(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	setupDiplomacyCollections(t, app)

	user1 := createTestUser(t, app, "war1")
	user2 := createTestUser(t, app, "war2")

	t.Run("declare war - new relation", func(t *testing.T) {
		err := DeclareWar(app, user1.Id, user2.Id)
		if err != nil {
			t.Fatalf("DeclareWar failed: %v", err)
		}

		p1ID, p2ID := orderedPlayerIDs(user1.Id, user2.Id)
		relation, _ := app.Dao().FindFirstRecordByFilter("diplomatic_relations",
			"player1_id={:p1} AND player2_id={:p2}",
			map[string]any{"p1": p1ID, "p2": p2ID})

		if relation == nil {
			t.Fatal("War relation not created")
		}
		if relation.GetString("status") != "war" {
			t.Errorf("Expected status 'war', got '%s'", relation.GetString("status"))
		}
		if relation.GetDateTime("start_date").IsZero() {
			t.Error("start_date for war is zero")
		}
		if relation.GetInt("duration_ticks") != 0 || !relation.GetDateTime("end_date").IsZero() {
			t.Error("War should be indefinite (duration_ticks 0, end_date zero)")
		}
	})

	t.Run("declare war - override existing peace", func(t *testing.T) {
		// Setup initial peace treaty
		peaceProposal, _ := ProposeTreaty(app, user1.Id, user2.Id, "peace_treaty", "", 0)
		AcceptProposal(app, peaceProposal.Id, user2.Id)

		// Now declare war
		err := DeclareWar(app, user1.Id, user2.Id)
		if err != nil {
			t.Fatalf("DeclareWar (override) failed: %v", err)
		}

		p1ID, p2ID := orderedPlayerIDs(user1.Id, user2.Id)
		relation, _ := app.Dao().FindFirstRecordByFilter("diplomatic_relations",
			"player1_id={:p1} AND player2_id={:p2}",
			map[string]any{"p1": p1ID, "p2": p2ID})

		if relation == nil {
			t.Fatal("War relation not found after overriding peace")
		}
		if relation.GetString("status") != "war" {
			t.Errorf("Expected status 'war' after overriding, got '%s'", relation.GetString("status"))
		}
		// Check that there's only one relation between them
		relations, _ := app.Dao().FindRecordsByFilter("diplomatic_relations",
			"(player1_id={:p1} AND player2_id={:p2}) OR (player1_id={:p2} AND player2_id={:p1})",
			"",0,0, map[string]any{"p1": p1ID, "p2": p2ID})
		if len(relations) != 1 {
			t.Errorf("Expected 1 relation, found %d", len(relations))
		}
	})

	t.Run("declare war on oneself", func(t *testing.T) {
		err := DeclareWar(app, user1.Id, user1.Id)
		if err == nil {
			t.Error("Expected error when declaring war on oneself, got nil")
		}
	})
}


func TestGetRelationsForUser(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	setupDiplomacyCollections(t, app)

	user1 := createTestUser(t, app, "gr_user1")
	user2 := createTestUser(t, app, "gr_user2")
	user3 := createTestUser(t, app, "gr_user3")
	user4 := createTestUser(t, app, "gr_user4_isolated")

	// Create some relations
	DeclareWar(app, user1.Id, user2.Id) // user1 - user2: war
	prop, _ := ProposeTreaty(app, user1.Id, user3.Id, "alliance_offer", "", 10)
	AcceptProposal(app, prop.Id, user3.Id) // user1 - user3: alliance

	prop2, _ := ProposeTreaty(app, user2.Id, user3.Id, "peace_treaty", "", 0)
	AcceptProposal(app, prop2.Id, user3.Id) // user2 - user3: peace

	// Test for user1
	relationsUser1, err := GetRelationsForUser(app, user1.Id)
	if err != nil {
		t.Fatalf("GetRelationsForUser for user1 failed: %v", err)
	}
	if len(relationsUser1) != 2 {
		t.Errorf("Expected 2 relations for user1, got %d", len(relationsUser1))
	}
	// Check specific relations (optional, if order is predictable or by checking content)
	foundWarWithUser2 := false
	foundAllianceWithUser3 := false
	for _, r := range relationsUser1 {
		isWithUser2 := (r.GetString("player1_id") == user2.Id || r.GetString("player2_id") == user2.Id)
		isWithUser3 := (r.GetString("player1_id") == user3.Id || r.GetString("player2_id") == user3.Id)
		if isWithUser2 && r.GetString("status") == "war" {
			foundWarWithUser2 = true
		}
		if isWithUser3 && r.GetString("status") == "alliance" {
			foundAllianceWithUser3 = true
		}
	}
	if !foundWarWithUser2 { t.Error("Expected war relation with user2 for user1 not found") }
	if !foundAllianceWithUser3 { t.Error("Expected alliance relation with user3 for user1 not found") }


	// Test for user4 (isolated)
	relationsUser4, err := GetRelationsForUser(app, user4.Id)
	if err != nil {
		t.Fatalf("GetRelationsForUser for user4 failed: %v", err)
	}
	if len(relationsUser4) != 0 {
		t.Errorf("Expected 0 relations for user4, got %d", len(relationsUser4))
	}
}


func TestGetPendingProposalsForUser(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	setupDiplomacyCollections(t, app)

	user1 := createTestUser(t, app, "gpp_user1") // Proposer
	user2 := createTestUser(t, app, "gpp_user2") // Receiver
	user3 := createTestUser(t, app, "gpp_user3") // Another receiver

	// Proposals for user2
	ProposeTreaty(app, user1.Id, user2.Id, "alliance_offer", `{"gold":100}`, 10)
	ProposeTreaty(app, user1.Id, user2.Id, "peace_treaty", ``, 0)

	// Proposal for user3
	ProposeTreaty(app, user1.Id, user3.Id, "trade_agreement", `{"resources":"iron"}`, 0)

	// An accepted proposal for user2 (should not be returned)
	acceptedProp, _ := ProposeTreaty(app, user1.Id, user2.Id, "alliance_offer", `{"detail":"accepted"}`, 0)
	AcceptProposal(app, acceptedProp.Id, user2.Id)


	// Test for user2
	pendingUser2, err := GetPendingProposalsForUser(app, user2.Id)
	if err != nil {
		t.Fatalf("GetPendingProposalsForUser for user2 failed: %v", err)
	}
	if len(pendingUser2) != 2 {
		for _, p := range pendingUser2 {
			t.Logf("Pending for user2: ID=%s, Type=%s, Status=%s, Terms=%s", p.Id, p.GetString("type"), p.GetString("status"), p.GetString("terms"))
		}
		t.Fatalf("Expected 2 pending proposals for user2, got %d", len(pendingUser2))
	}
	for _, p := range pendingUser2 {
		if p.GetString("status") != "pending" {
			t.Errorf("Returned proposal for user2 is not pending: %s", p.GetString("status"))
		}
		if p.GetString("receiver_id") != user2.Id {
			t.Errorf("Returned proposal has wrong receiver_id: %s", p.GetString("receiver_id"))
		}
	}


	// Test for user3
	pendingUser3, err := GetPendingProposalsForUser(app, user3.Id)
	if err != nil {
		t.Fatalf("GetPendingProposalsForUser for user3 failed: %v", err)
	}
	if len(pendingUser3) != 1 {
		t.Fatalf("Expected 1 pending proposal for user3, got %d", len(pendingUser3))
	}
	if pendingUser3[0].GetString("type") != "trade_agreement" {
		t.Errorf("Expected 'trade_agreement' proposal for user3, got '%s'", pendingUser3[0].GetString("type"))
	}

	// Test for user1 (should have no pending proposals *received*)
	pendingUser1, err := GetPendingProposalsForUser(app, user1.Id)
	if err != nil {
		t.Fatalf("GetPendingProposalsForUser for user1 failed: %v", err)
	}
	if len(pendingUser1) != 0 {
		t.Fatalf("Expected 0 pending proposals for user1 (receiver), got %d", len(pendingUser1))
	}
}

// Example of testing JSON terms in a proposal
func TestProposalWithJsonTerms(t *testing.T) {
	app, _ := tests.NewTestApp()
	defer app.Cleanup()
	setupDiplomacyCollections(t, app)
	user1 := createTestUser(t, app, "json_u1")
	user2 := createTestUser(t, app, "json_u2")

	complexTerms := map[string]any{
		"resource_exchange": map[string]any{"give_resource": "iron", "give_amount": 1000, "receive_resource": "food", "receive_amount": 500},
		"duration_years": 5,
		"clauses": []string{"Non-aggression", "Mutual trade benefits"},
	}
	termsBytes, _ := json.Marshal(complexTerms)
	termsStr := string(termsBytes)

	proposal, err := ProposeTreaty(app, user1.Id, user2.Id, "custom_treaty", termsStr, 0)
	if err != nil { t.Fatalf("ProposeTreaty failed: %v", err) }

	fetchedProposal, _ := app.Dao().FindRecordById("diplomatic_proposals", proposal.Id)

	var fetchedTerms map[string]any
	err = json.Unmarshal([]byte(fetchedProposal.GetString("terms")), &fetchedTerms)
	if err != nil { t.Fatalf("Failed to unmarshal fetched terms: %v", err) }

    // Example check - you might need a deep comparison helper for complex maps/structs
	if fmt.Sprintf("%v", fetchedTerms["duration_years"]) != fmt.Sprintf("%v", complexTerms["duration_years"]) {
		t.Errorf("JSON term 'duration_years' mismatch. Expected %v, got %v", complexTerms["duration_years"], fetchedTerms["duration_years"])
	}
}
```
