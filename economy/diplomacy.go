package economy

import (
	"fmt"
	"sync"
)

// Relation levels
const (
	RelationHostile  = -2
	RelationCold     = -1
	RelationNeutral  = 0
	RelationFriendly = 1
	RelationAllied   = 2
)

// RelationName returns a human-readable name for a relation level.
func RelationName(level int) string {
	switch {
	case level <= RelationHostile:
		return "Hostile"
	case level == RelationCold:
		return "Cold"
	case level == RelationNeutral:
		return "Neutral"
	case level == RelationFriendly:
		return "Friendly"
	case level >= RelationAllied:
		return "Allied"
	default:
		return "Unknown"
	}
}

// DiplomacyManager tracks relations between factions.
// Relations affect trade fees, docking rights, and event interactions.
type DiplomacyManager struct {
	mu        sync.RWMutex
	relations map[string]map[string]int // faction → faction → relation level
}

// NewDiplomacyManager creates a new diplomacy manager.
func NewDiplomacyManager() *DiplomacyManager {
	return &DiplomacyManager{
		relations: make(map[string]map[string]int),
	}
}

// GetRelation returns the relation level between two factions.
func (dm *DiplomacyManager) GetRelation(a, b string) int {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	if m, ok := dm.relations[a]; ok {
		if level, ok := m[b]; ok {
			return level
		}
	}
	return RelationNeutral
}

// SetRelation sets the relation between two factions (symmetric).
func (dm *DiplomacyManager) SetRelation(a, b string, level int) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.relations[a] == nil {
		dm.relations[a] = make(map[string]int)
	}
	if dm.relations[b] == nil {
		dm.relations[b] = make(map[string]int)
	}
	dm.relations[a][b] = level
	dm.relations[b][a] = level

	fmt.Printf("[Diplomacy] %s ↔ %s: %s\n", a, b, RelationName(level))
}

// ImproveRelation moves the relation one step toward Allied.
func (dm *DiplomacyManager) ImproveRelation(a, b string) int {
	current := dm.GetRelation(a, b)
	if current < RelationAllied {
		dm.SetRelation(a, b, current+1)
		return current + 1
	}
	return current
}

// DegradeRelation moves the relation one step toward Hostile.
func (dm *DiplomacyManager) DegradeRelation(a, b string) int {
	current := dm.GetRelation(a, b)
	if current > RelationHostile {
		dm.SetRelation(a, b, current-1)
		return current - 1
	}
	return current
}

// GetAllRelations returns all relations for a faction.
func (dm *DiplomacyManager) GetAllRelations(faction string) map[string]int {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	result := make(map[string]int)
	if m, ok := dm.relations[faction]; ok {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// GetAllRelationsMap returns everything (for save/load).
func (dm *DiplomacyManager) GetAllRelationsMap() map[string]map[string]int {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	result := make(map[string]map[string]int)
	for a, m := range dm.relations {
		result[a] = make(map[string]int)
		for b, level := range m {
			result[a][b] = level
		}
	}
	return result
}

// RestoreRelations loads from save.
func (dm *DiplomacyManager) RestoreRelations(data map[string]map[string]int) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if data != nil {
		dm.relations = data
	}
}

// DockingFeeMultiplier returns the fee adjustment based on relation.
// Allies get reduced fees, hostile factions get increased fees.
func DockingFeeMultiplier(relation int) float64 {
	switch {
	case relation >= RelationAllied:
		return 0.25 // 75% discount for allies
	case relation == RelationFriendly:
		return 0.5 // 50% discount
	case relation == RelationNeutral:
		return 1.0 // standard
	case relation == RelationCold:
		return 1.5 // 50% surcharge
	default:
		return 2.0 // double fees for hostile
	}
}
