package diplomacy

import (
	"fmt"
	"sort"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

// DiplomaticRelation represents the diplomatic state between two players.
type DiplomaticRelation struct {
	Player1ID string    `json:"player1_id" db:"player1_id"`
	Player2ID string    `json:"player2_id" db:"player2_id"`
	Status    string    `json:"status" db:"status"` // e.g., "War", "Peace", "Alliance"
	StartDate time.Time `json:"start_date" db:"start_date"`
	Duration  int       `json:"duration" db:"duration"` // In game ticks, 0 for indefinite
	EndDate   time.Time `json:"end_date,omitempty" db:"end_date,omitempty"` // Calculated on creation if duration is not 0
}

// DiplomaticProposal represents a diplomatic proposal from one player to another.
type DiplomaticProposal struct {
	ProposerID     string    `json:"proposer_id" db:"proposer_id"`
	ReceiverID     string    `json:"receiver_id" db:"receiver_id"`
	Type           string    `json:"type" db:"type"` // e.g., "PeaceTreaty", "AllianceOffer", "TradeAgreement"
	Terms          string    `json:"terms" db:"terms"` // Can be JSON or other structured text for complex terms
	Status         string    `json:"status" db:"status"` // e.g., "Pending", "Accepted", "Rejected", "Expired"
	ProposedDate   time.Time `json:"proposed_date" db:"proposed_date"`
	ExpirationDate time.Time `json:"expiration_date" db:"expiration_date"`
	DurationTicks  int       `json:"duration_ticks" db:"duration_ticks"` // Used for proposals that define a duration for the relation
}

// Helper function to ensure consistent ordering of player IDs
func orderedPlayerIDs(id1, id2 string) (string, string) {
	ids := []string{id1, id2}
	sort.Strings(ids)
	return ids[0], ids[1]
}

// ProposeTreaty creates a new diplomatic proposal.
func ProposeTreaty(app *pocketbase.PocketBase, proposerID string, receiverID string, proposalType string, terms string, durationTicks int) (*models.Record, error) {
	if proposerID == "" || receiverID == "" {
		return nil, fmt.Errorf("proposerID and receiverID cannot be empty")
	}
	if proposerID == receiverID {
		return nil, fmt.Errorf("proposerID and receiverID cannot be the same")
	}

	collection, err := app.Dao().FindCollectionByNameOrId("diplomatic_proposals")
	if err != nil {
		return nil, fmt.Errorf("failed to find diplomatic_proposals collection: %w", err)
	}

	record := models.NewRecord(collection)
	record.Set("proposer_id", proposerID)
	record.Set("receiver_id", receiverID)
	record.Set("type", proposalType)
	record.Set("terms", terms) // Assuming terms is a JSON string or simple text
	record.Set("status", "pending")
	record.Set("proposed_date", time.Now().UTC())
	record.Set("expiration_date", time.Now().UTC().Add(24*time.Hour)) // Example: 24 hour expiration
	record.Set("duration_ticks", durationTicks)

	if err := app.Dao().SaveRecord(record); err != nil {
		return nil, fmt.Errorf("failed to save proposal: %w", err)
	}

	return record, nil
}

// AcceptProposal handles a player accepting a diplomatic proposal.
func AcceptProposal(app *pocketbase.PocketBase, proposalID string, acceptorID string) error {
	// Fetch the proposal
	proposalRecord, err := app.Dao().FindRecordById("diplomatic_proposals", proposalID)
	if err != nil {
		return fmt.Errorf("failed to find proposal %s: %w", proposalID, err)
	}

	// Verify acceptor
	if proposalRecord.GetString("receiver_id") != acceptorID {
		return fmt.Errorf("acceptorID %s does not match proposal receiverID %s", acceptorID, proposalRecord.GetString("receiver_id"))
	}

	// Verify proposal status
	if proposalRecord.GetString("status") != "pending" {
		return fmt.Errorf("proposal %s is not pending, current status: %s", proposalID, proposalRecord.GetString("status"))
	}

	// Update proposal status to "accepted"
	proposalRecord.Set("status", "accepted")
	if err := app.Dao().SaveRecord(proposalRecord); err != nil {
		return fmt.Errorf("failed to update proposal %s status: %w", proposalID, err)
	}

	// Create or update diplomatic relation
	relationsCollection, err := app.Dao().FindCollectionByNameOrId("diplomatic_relations")
	if err != nil {
		return fmt.Errorf("failed to find diplomatic_relations collection: %w", err)
	}

	proposerID := proposalRecord.GetString("proposer_id")
	player1ID, player2ID := orderedPlayerIDs(proposerID, acceptorID)
	proposalType := proposalRecord.GetString("type")
	durationTicks := proposalRecord.GetInt("duration_ticks")

	var relationStatus string
	switch proposalType {
	case "peace_treaty":
		relationStatus = "peace"
	case "alliance_offer":
		relationStatus = "alliance"
	case "non_aggression_pact":
		relationStatus = "truce" // Or a specific status for NAP
	// Add other proposal types that result in a diplomatic relation
	default:
		// For proposals like "trade_agreement", we might not create a diplomatic_relations record,
		// or we might have a different system. For now, we'll assume it creates a neutral relation or none.
		// If no specific relation is formed, we can return early or handle as per game logic.
		app.Logger().Warn(fmt.Sprintf("Proposal type %s does not directly translate to a standard diplomatic relation status.", proposalType))
		return nil // Or handle appropriately
	}

	// Try to find an existing relation
	existingRelation, _ := app.Dao().FindFirstRecordByFilter(
		"diplomatic_relations",
		"((player1_id = {:p1} AND player2_id = {:p2}) OR (player1_id = {:p2} AND player2_id = {:p1}))",
		daos.Params{"p1": player1ID, "p2": player2ID},
	)

	var relationRecord *models.Record
	if existingRelation != nil {
		relationRecord = existingRelation
	} else {
		relationRecord = models.NewRecord(relationsCollection)
		relationRecord.Set("player1_id", player1ID)
		relationRecord.Set("player2_id", player2ID)
	}

	relationRecord.Set("status", relationStatus)
	relationRecord.Set("start_date", time.Now().UTC())
	relationRecord.Set("duration_ticks", durationTicks)

	if durationTicks > 0 {
		// This is a placeholder for game tick based duration.
		// In a real scenario, EndDate might be calculated by a game loop or a cron job
		// For now, let's assume 1 game tick = 1 minute for calculation purposes if we need a time.Time
		// However, the schema asks for end_date to be calculated.
		// If we don't have a direct tick-to-time conversion readily available,
		// we might store ticks and calculate EndDate when materializing views or on specific events.
		// For simplicity, if we *had* to set an EndDate from ticks now, we'd need a conversion.
		// Let's assume we store duration_ticks and EndDate is managed by another process or set to null if not directly time-based.
		// The schema notes "calculated on creation if duration is not 0".
		// Let's assume a hypothetical conversion: 1 tick = 1 hour for this example.
		// This should ideally be tied to game's time progression system.
		endDate := time.Now().UTC().Add(time.Duration(durationTicks) * time.Hour)
		relationRecord.Set("end_date", endDate)
	} else {
		relationRecord.Set("end_date", nil) // Indefinite
	}

	if err := app.Dao().SaveRecord(relationRecord); err != nil {
		// Revert proposal status if saving relation fails? Complex transaction needed.
		// For now, log and return error.
		return fmt.Errorf("failed to save diplomatic relation: %w", err)
	}

	return nil
}

// RejectProposal handles a player rejecting a diplomatic proposal.
func RejectProposal(app *pocketbase.PocketBase, proposalID string, rejectorID string) error {
	proposalRecord, err := app.Dao().FindRecordById("diplomatic_proposals", proposalID)
	if err != nil {
		return fmt.Errorf("failed to find proposal %s: %w", proposalID, err)
	}

	if proposalRecord.GetString("receiver_id") != rejectorID {
		return fmt.Errorf("rejectorID %s does not match proposal receiverID %s", rejectorID, proposalRecord.GetString("receiver_id"))
	}

	if proposalRecord.GetString("status") != "pending" {
		return fmt.Errorf("proposal %s is not pending, current status: %s", proposalID, proposalRecord.GetString("status"))
	}

	proposalRecord.Set("status", "rejected")
	if err := app.Dao().SaveRecord(proposalRecord); err != nil {
		return fmt.Errorf("failed to update proposal %s status to rejected: %w", proposalID, err)
	}

	return nil
}

// DeclareWar declares war between two players.
func DeclareWar(app *pocketbase.PocketBase, declarerID string, targetID string) error {
	if declarerID == "" || targetID == "" {
		return fmt.Errorf("declarerID and targetID cannot be empty")
	}
	if declarerID == targetID {
		return fmt.Errorf("declarerID and targetID cannot be the same")
	}

	relationsCollection, err := app.Dao().FindCollectionByNameOrId("diplomatic_relations")
	if err != nil {
		return fmt.Errorf("failed to find diplomatic_relations collection: %w", err)
	}

	player1ID, player2ID := orderedPlayerIDs(declarerID, targetID)

	existingRelation, _ := app.Dao().FindFirstRecordByFilter(
		"diplomatic_relations",
		"((player1_id = {:p1} AND player2_id = {:p2}) OR (player1_id = {:p2} AND player2_id = {:p1}))",
		daos.Params{"p1": player1ID, "p2": player2ID},
	)

	var relationRecord *models.Record
	if existingRelation != nil {
		relationRecord = existingRelation
	} else {
		relationRecord = models.NewRecord(relationsCollection)
		relationRecord.Set("player1_id", player1ID)
		relationRecord.Set("player2_id", player2ID)
	}

	relationRecord.Set("status", "war")
	relationRecord.Set("start_date", time.Now().UTC())
	relationRecord.Set("duration_ticks", 0) // War is indefinite
	relationRecord.Set("end_date", nil)     // No end date for war unless a peace treaty is signed

	if err := app.Dao().SaveRecord(relationRecord); err != nil {
		return fmt.Errorf("failed to declare war by saving relation: %w", err)
	}

	// Optional: Cancel any pending peace proposals between these players
	// This would require another query and update operation.
	// For example:
	// proposalsToCancel, _ := app.Dao().FindRecordsByFilter(
	// 	"diplomatic_proposals",
	// 	"status = 'pending' AND type = 'peace_treaty' AND "+
	// 		"((proposer_id = {:p1} AND receiver_id = {:p2}) OR (proposer_id = {:p2} AND receiver_id = {:p1}))",
	// 	nil, 0, 0, daos.Params{"p1": declarerID, "p2": targetID},
	// )
	// for _, proposal := range proposalsToCancel {
	// 	proposal.Set("status", "cancelled") // or "expired"
	// 	app.Dao().SaveRecord(proposal)
	// }

	return nil
}

// GetRelationsForUser retrieves all diplomatic relations for a given user.
func GetRelationsForUser(app *pocketbase.PocketBase, userID string) ([]*models.Record, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	records, err := app.Dao().FindRecordsByFilter(
		"diplomatic_relations",
		"player1_id = {:userID} || player2_id = {:userID}",
		"+created", // Sort by oldest first
		0,          // No limit
		0,          // No offset
		daos.Params{"userID": userID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch relations for user %s: %w", userID, err)
	}
	return records, nil
}

// GetPendingProposalsForUser retrieves all pending diplomatic proposals for a given user (as receiver).
func GetPendingProposalsForUser(app *pocketbase.PocketBase, userID string) ([]*models.Record, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	records, err := app.Dao().FindRecordsByFilter(
		"diplomatic_proposals",
		"receiver_id = {:userID} && status = 'pending'",
		"+proposed_date", // Sort by oldest first
		0,                // No limit
		0,                // No offset
		daos.Params{"userID": userID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending proposals for user %s: %w", userID, err)
	}
	return records, nil
}

/*
PocketBase Collection Schemas:

Collection: diplomatic_relations
Fields:
- id (primary key, auto-generated)
- created (timestamp, auto-generated)
- updated (timestamp, auto-generated)
- player1_id (relation to users, non-empty, indexed)
- player2_id (relation to users, non-empty, indexed)
- status (text, non-empty, default: "neutral", allowed values: "war", "peace", "alliance", "neutral", "truce")
- start_date (date, non-empty)
- duration_ticks (number, default: 0) // 0 for indefinite
- end_date (date, optional, calculated on creation if duration_ticks is not 0)

Collection: diplomatic_proposals
Fields:
- id (primary key, auto-generated)
- created (timestamp, auto-generated)
- updated (timestamp, auto-generated)
- proposer_id (relation to users, non-empty, indexed)
- receiver_id (relation to users, non-empty, indexed)
- type (text, non-empty, allowed values: "peace_treaty", "alliance_offer", "trade_agreement", "declaration_of_war", "non_aggression_pact", "vassalage_offer")
- terms (json, optional) // For storing details of the proposal, e.g., tribute amount, resource exchange
- status (text, non-empty, default: "pending", allowed values: "pending", "accepted", "rejected", "expired", "cancelled")
- proposed_date (date, non-empty)
- expiration_date (date, non-empty)
- duration_ticks (number, default: 0) // Added to struct for convenience, matches proposal's intent for relation duration
*/
