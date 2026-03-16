package economy

import (
	"fmt"
	"sync"
)

// CreditLedger tracks outstanding credit exposure between empires.
// When Player A buys from Player B, the trade value counts against A's
// credit limit with B. On delivery completion, the outstanding balance decreases.
type CreditLedger struct {
	mu          sync.RWMutex
	outstanding map[string]map[string]int // ownerEmpire → targetEmpire → outstandingCredits
	limits      map[string]map[string]int // ownerEmpire → targetEmpire → maxCredits (0 = use TP default)
}

// NewCreditLedger creates a new credit ledger.
func NewCreditLedger() *CreditLedger {
	return &CreditLedger{
		outstanding: make(map[string]map[string]int),
		limits:      make(map[string]map[string]int),
	}
}

// SetLimit sets the credit limit that `owner` extends to `target`.
// A limit of 0 means use the Trading Post's default limit.
// A limit of -1 means unlimited.
func (cl *CreditLedger) SetLimit(owner, target string, limit int) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if cl.limits[owner] == nil {
		cl.limits[owner] = make(map[string]int)
	}
	cl.limits[owner][target] = limit
}

// GetLimit returns the credit limit that `owner` extends to `target`.
// Returns 0 if no custom limit is set (caller should use TP default).
func (cl *CreditLedger) GetLimit(owner, target string) int {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	if m, ok := cl.limits[owner]; ok {
		if limit, ok := m[target]; ok {
			return limit
		}
	}
	return 0
}

// GetOutstanding returns the current outstanding credit balance from `buyer` toward `seller`.
func (cl *CreditLedger) GetOutstanding(buyer, seller string) int {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	if m, ok := cl.outstanding[buyer]; ok {
		return m[seller]
	}
	return 0
}

// AddOutstanding increases the outstanding balance when a trade is initiated.
func (cl *CreditLedger) AddOutstanding(buyer, seller string, amount int) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if cl.outstanding[buyer] == nil {
		cl.outstanding[buyer] = make(map[string]int)
	}
	cl.outstanding[buyer][seller] += amount
}

// ReduceOutstanding decreases the outstanding balance when a delivery completes.
func (cl *CreditLedger) ReduceOutstanding(buyer, seller string, amount int) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if m, ok := cl.outstanding[buyer]; ok {
		m[seller] -= amount
		if m[seller] < 0 {
			m[seller] = 0
		}
	}
}

// CheckLimit returns nil if the trade is within credit limits, or an error if over limit.
// tpDefaultLimit is the Trading Post's default credit limit (0 = unlimited).
func (cl *CreditLedger) CheckLimit(buyer, seller string, tradeValue int, tpDefaultLimit int) error {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	// Determine effective limit
	limit := tpDefaultLimit
	if m, ok := cl.limits[buyer]; ok {
		if customLimit, ok := m[seller]; ok {
			if customLimit == -1 {
				return nil // unlimited
			}
			limit = customLimit
		}
	}

	if limit <= 0 {
		return nil // no limit
	}

	current := 0
	if m, ok := cl.outstanding[buyer]; ok {
		current = m[seller]
	}

	if current+tradeValue > limit {
		return fmt.Errorf("credit limit exceeded with %s (outstanding %d + trade %d > limit %d)",
			seller, current, tradeValue, limit)
	}
	return nil
}

// GetAllOutstanding returns a copy of all outstanding balances (for save/load).
func (cl *CreditLedger) GetAllOutstanding() map[string]map[string]int {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	result := make(map[string]map[string]int, len(cl.outstanding))
	for buyer, sellers := range cl.outstanding {
		m := make(map[string]int, len(sellers))
		for seller, amount := range sellers {
			m[seller] = amount
		}
		result[buyer] = m
	}
	return result
}

// GetAllLimits returns a copy of all custom limits (for save/load).
func (cl *CreditLedger) GetAllLimits() map[string]map[string]int {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	result := make(map[string]map[string]int, len(cl.limits))
	for owner, targets := range cl.limits {
		m := make(map[string]int, len(targets))
		for target, limit := range targets {
			m[target] = limit
		}
		result[owner] = m
	}
	return result
}

// RestoreLedger loads outstanding and limits from a save.
func (cl *CreditLedger) RestoreLedger(outstanding, limits map[string]map[string]int) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if outstanding != nil {
		cl.outstanding = outstanding
	}
	if limits != nil {
		cl.limits = limits
	}
}
