package economy

import (
	"fmt"
	"sync"
)

// Bounty is a player-posted job on the galaxy-wide bounty board.
// Types:
//   "deliver" — deliver X units of a resource to a planet
//   "clear_pirates" — defeat pirates in a system
//   "explore" — explore a specific system with a scout
//   "custom" — free-form task described in text
type Bounty struct {
	ID          int
	Poster      string // faction who posted it
	Type        string // "deliver", "clear_pirates", "explore", "custom"
	Description string // human-readable description
	Reward      int    // credits paid on completion
	// Delivery bounties
	Resource string // for "deliver" type
	Quantity int    // for "deliver" type
	PlanetID int    // destination planet for "deliver"
	SystemID int    // target system for "clear_pirates" / "explore"
	// Status
	Claimant  string // faction that claimed the bounty
	Completed bool
	Active    bool
}

// BountyBoard manages the galaxy-wide bounty board.
type BountyBoard struct {
	mu       sync.RWMutex
	bounties []*Bounty
	nextID   int
}

// NewBountyBoard creates an empty bounty board.
func NewBountyBoard() *BountyBoard {
	return &BountyBoard{
		bounties: make([]*Bounty, 0),
		nextID:   1,
	}
}

// PostBounty creates a new bounty. Poster's credits are held in escrow.
func (bb *BountyBoard) PostBounty(poster, bountyType, description string, reward int, resource string, quantity, planetID, systemID int) *Bounty {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	b := &Bounty{
		ID:          bb.nextID,
		Poster:      poster,
		Type:        bountyType,
		Description: description,
		Reward:      reward,
		Resource:    resource,
		Quantity:    quantity,
		PlanetID:    planetID,
		SystemID:    systemID,
		Active:      true,
	}
	bb.nextID++
	bb.bounties = append(bb.bounties, b)

	fmt.Printf("[Bounty] #%d posted by %s: %s (reward: %dcr)\n",
		b.ID, poster, description, reward)
	return b
}

// ClaimBounty marks a bounty as claimed by a faction.
func (bb *BountyBoard) ClaimBounty(id int, claimant string) bool {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	for _, b := range bb.bounties {
		if b.ID == id && b.Active && b.Claimant == "" && b.Poster != claimant {
			b.Claimant = claimant
			return true
		}
	}
	return false
}

// CompleteBounty marks a bounty as completed and returns the reward amount.
func (bb *BountyBoard) CompleteBounty(id int, claimant string) int {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	for _, b := range bb.bounties {
		if b.ID == id && b.Active && b.Claimant == claimant && !b.Completed {
			b.Completed = true
			b.Active = false
			return b.Reward
		}
	}
	return 0
}

// CancelBounty cancels an unclaimed bounty (poster gets refund).
func (bb *BountyBoard) CancelBounty(id int, poster string) bool {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	for _, b := range bb.bounties {
		if b.ID == id && b.Active && b.Poster == poster && b.Claimant == "" {
			b.Active = false
			return true
		}
	}
	return false
}

// GetActiveBounties returns all active bounties.
func (bb *BountyBoard) GetActiveBounties() []*Bounty {
	bb.mu.RLock()
	defer bb.mu.RUnlock()

	var result []*Bounty
	for _, b := range bb.bounties {
		if b.Active {
			result = append(result, b)
		}
	}
	return result
}
