package economy

import (
	"fmt"
	"sync"
)

// CouncilProposal is a galaxy-wide policy up for vote.
type CouncilProposal struct {
	ID          int
	Title       string
	Description string
	Effect      string // what happens if passed
	Proposer    string
	Votes       map[string]bool // faction → yes/no
	TicksLeft   int
	Passed      bool
	Resolved    bool
}

// GalacticCouncil manages policy proposals and voting.
type GalacticCouncil struct {
	mu        sync.RWMutex
	proposals []*CouncilProposal
	nextID    int
}

// NewGalacticCouncil creates a new council.
func NewGalacticCouncil() *GalacticCouncil {
	return &GalacticCouncil{
		proposals: make([]*CouncilProposal, 0),
		nextID:    1,
	}
}

// Propose creates a new proposal for voting.
func (gc *GalacticCouncil) Propose(proposer, title, description, effect string, duration int) *CouncilProposal {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	p := &CouncilProposal{
		ID:          gc.nextID,
		Title:       title,
		Description: description,
		Effect:      effect,
		Proposer:    proposer,
		Votes:       map[string]bool{proposer: true}, // proposer auto-votes yes
		TicksLeft:   duration,
	}
	gc.nextID++
	gc.proposals = append(gc.proposals, p)

	fmt.Printf("[Council] Proposal #%d: %s by %s\n", p.ID, title, proposer)
	return p
}

// Vote casts a vote on a proposal.
func (gc *GalacticCouncil) Vote(proposalID int, faction string, yes bool) bool {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	for _, p := range gc.proposals {
		if p.ID == proposalID && !p.Resolved && p.TicksLeft > 0 {
			p.Votes[faction] = yes
			return true
		}
	}
	return false
}

// TickProposals decrements timers and returns resolved proposals.
func (gc *GalacticCouncil) TickProposals(totalFactions int) []*CouncilProposal {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	var resolved []*CouncilProposal
	for _, p := range gc.proposals {
		if p.Resolved {
			continue
		}
		p.TicksLeft--
		if p.TicksLeft <= 0 {
			// Count votes
			yes, no := 0, 0
			for _, v := range p.Votes {
				if v {
					yes++
				} else {
					no++
				}
			}
			p.Passed = yes > no && yes > totalFactions/3 // need majority of voters AND >1/3 of factions
			p.Resolved = true
			resolved = append(resolved, p)
		}
	}
	return resolved
}

// GetActiveProposals returns proposals still open for voting.
func (gc *GalacticCouncil) GetActiveProposals() []*CouncilProposal {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	var result []*CouncilProposal
	for _, p := range gc.proposals {
		if !p.Resolved && p.TicksLeft > 0 {
			result = append(result, p)
		}
	}
	return result
}
