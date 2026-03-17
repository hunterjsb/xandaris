package economy

import (
	"fmt"
	"math/rand"
	"sync"
)

// SpyOperation represents an active espionage mission.
type SpyOperation struct {
	ID         int
	Operator   string // faction running the op
	Target     string // faction being spied on
	Type       string // "intel", "sabotage", "steal_tech"
	SystemID   int    // system where the op is running
	TicksLeft  int    // ticks until completion
	Cost       int    // credits already paid
	Active     bool
}

// SpyResult is returned when an operation completes.
type SpyResult struct {
	Success bool
	Message string
	Data    interface{} // varies by op type
}

// EspionageManager handles spy operations between factions.
type EspionageManager struct {
	mu     sync.RWMutex
	ops    []*SpyOperation
	nextID int
}

// NewEspionageManager creates a new espionage manager.
func NewEspionageManager() *EspionageManager {
	return &EspionageManager{
		ops:    make([]*SpyOperation, 0),
		nextID: 1,
	}
}

// LaunchOperation starts a new spy mission.
// Types:
//   "intel" (500cr, 200 ticks) — reveals target's planet details + storage
//   "sabotage" (2000cr, 500 ticks) — damages a random building on target's planet
//   "steal_tech" (5000cr, 1000 ticks) — steals 0.5 tech levels from target
func (em *EspionageManager) LaunchOperation(operator, target, opType string, systemID, cost, duration int) *SpyOperation {
	em.mu.Lock()
	defer em.mu.Unlock()

	op := &SpyOperation{
		ID:        em.nextID,
		Operator:  operator,
		Target:    target,
		Type:      opType,
		SystemID:  systemID,
		TicksLeft: duration,
		Cost:      cost,
		Active:    true,
	}
	em.nextID++
	em.ops = append(em.ops, op)

	fmt.Printf("[Espionage] #%d: %s launched %s op against %s in SYS-%d\n",
		op.ID, operator, opType, target, systemID+1)
	return op
}

// TickOperations decrements timers and returns completed operations.
func (em *EspionageManager) TickOperations() []*SpyOperation {
	em.mu.Lock()
	defer em.mu.Unlock()

	var completed []*SpyOperation
	for _, op := range em.ops {
		if !op.Active {
			continue
		}
		op.TicksLeft--
		if op.TicksLeft <= 0 {
			op.Active = false
			completed = append(completed, op)
		}
	}
	return completed
}

// ResolveOperation determines success/failure of a completed spy op.
// Base success rate: 60% for intel, 40% for sabotage, 30% for steal_tech.
// Planetary Shield reduces success by 20%.
func ResolveOperation(op *SpyOperation, targetHasShield bool) SpyResult {
	baseRate := 0.6
	switch op.Type {
	case "sabotage":
		baseRate = 0.4
	case "steal_tech":
		baseRate = 0.3
	}

	if targetHasShield {
		baseRate -= 0.2
	}

	success := rand.Float64() < baseRate

	if !success {
		return SpyResult{
			Success: false,
			Message: fmt.Sprintf("Spy operation '%s' against %s failed! Agent was detected.",
				op.Type, op.Target),
		}
	}

	switch op.Type {
	case "intel":
		return SpyResult{
			Success: true,
			Message: fmt.Sprintf("Intel gathered on %s — their planet details have been revealed!",
				op.Target),
		}
	case "sabotage":
		return SpyResult{
			Success: true,
			Message: fmt.Sprintf("Sabotage successful! A building on %s's planet has been damaged.",
				op.Target),
		}
	case "steal_tech":
		return SpyResult{
			Success: true,
			Message: fmt.Sprintf("Technology stolen from %s! +0.5 tech level gained.",
				op.Target),
		}
	default:
		return SpyResult{Success: false, Message: "Unknown operation type"}
	}
}

// GetActiveOps returns active operations for a player.
func (em *EspionageManager) GetActiveOps(player string) []*SpyOperation {
	em.mu.RLock()
	defer em.mu.RUnlock()
	var result []*SpyOperation
	for _, op := range em.ops {
		if op.Active && op.Operator == player {
			result = append(result, op)
		}
	}
	return result
}
