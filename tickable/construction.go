package tickable

import (
	"sync"
)

func init() {
	RegisterSystem(&ConstructionSystem{
		BaseSystem:  NewBaseSystem("Construction", 20),
		queues:      NewSafeMap[string, *ConstructionQueue](),
		completions: make(chan ConstructionCompletion, 100),
	})
}

// ConstructionItem represents a single item being constructed
type ConstructionItem struct {
	ID             string
	Type           string // "Building", "Station", "Ship", etc.
	Name           string
	Location       string // Planet/Station ID
	Owner          string // Player name
	Progress       int    // Current progress (0-100)
	TotalTicks     int    // Ticks required to complete
	RemainingTicks int    // Ticks remaining
	Cost           int    // Credit cost
	Started        int64  // Tick when started
	mutex          sync.RWMutex
}

// ConstructionQueue manages construction for a single location
type ConstructionQueue struct {
	Location string
	Items    []*ConstructionItem
	mutex    sync.RWMutex
}

// ConstructionCompletion represents a completed construction item
type ConstructionCompletion struct {
	Item     *ConstructionItem
	Location string
	Owner    string
	Tick     int64
}

// ConstructionSystem handles all construction queues
type ConstructionSystem struct {
	*BaseSystem
	queues      *SafeMap[string, *ConstructionQueue]
	completions chan ConstructionCompletion
	mutex       sync.RWMutex
}

// OnTick processes all construction queues
func (cs *ConstructionSystem) OnTick(tick int64) {
	// Get all queues
	var allQueues []*ConstructionQueue
	cs.queues.ForEach(func(location string, queue *ConstructionQueue) {
		allQueues = append(allQueues, queue)
	})

	if len(allQueues) == 0 {
		return
	}

	// Process all queues concurrently
	ProcessConcurrent(allQueues, 4, func(queue *ConstructionQueue) {
		cs.processQueue(queue, tick)
	})

	// Handle completions
	cs.processCompletions()
}

// processQueue processes a single construction queue
func (cs *ConstructionSystem) processQueue(queue *ConstructionQueue, tick int64) {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	// Process only the first item (active construction)
	if len(queue.Items) == 0 {
		return
	}

	item := queue.Items[0]
	item.mutex.Lock()
	defer item.mutex.Unlock()

	// Decrement remaining ticks
	item.RemainingTicks--
	if item.RemainingTicks < 0 {
		item.RemainingTicks = 0
	}

	// Update progress percentage
	if item.TotalTicks > 0 {
		item.Progress = int(float64(item.TotalTicks-item.RemainingTicks) / float64(item.TotalTicks) * 100)
	}

	// Check if complete
	if item.RemainingTicks == 0 {
		// Send completion notification
		select {
		case cs.completions <- ConstructionCompletion{
			Item:     item,
			Location: queue.Location,
			Owner:    item.Owner,
			Tick:     tick,
		}:
		default:
			// Channel full, skip
		}

		// Remove from queue (already holding lock)
		queue.Items = queue.Items[1:]
	}
}

// processCompletions handles completed constructions
func (cs *ConstructionSystem) processCompletions() {
	// Drain completion channel
	for {
		select {
		case completion := <-cs.completions:
			cs.handleCompletion(completion)
		default:
			return
		}
	}
}

// handleCompletion handles a single completed construction
func (cs *ConstructionSystem) handleCompletion(completion ConstructionCompletion) {
	// This would trigger:
	// 1. Add building to planet
	// 2. Update planet stats
	// 3. Notify player
	// 4. Update UI

	// For now, just a placeholder
	// In full implementation, this would interact with game state
}

// AddToQueue adds a construction item to a location's queue
func (cs *ConstructionSystem) AddToQueue(location string, item *ConstructionItem) {
	queue, exists := cs.queues.Get(location)
	if !exists {
		queue = &ConstructionQueue{
			Location: location,
			Items:    make([]*ConstructionItem, 0),
		}
		cs.queues.Set(location, queue)
	}

	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	queue.Items = append(queue.Items, item)
}

// RemoveFromQueue removes an item from the queue by ID
func (cs *ConstructionSystem) RemoveFromQueue(location string, itemID string) bool {
	queue, exists := cs.queues.Get(location)
	if !exists {
		return false
	}

	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	for i, item := range queue.Items {
		if item.ID == itemID {
			queue.Items = append(queue.Items[:i], queue.Items[i+1:]...)
			return true
		}
	}
	return false
}

// GetQueue returns the construction queue for a location
func (cs *ConstructionSystem) GetQueue(location string) []*ConstructionItem {
	queue, exists := cs.queues.Get(location)
	if !exists {
		return []*ConstructionItem{}
	}

	queue.mutex.RLock()
	defer queue.mutex.RUnlock()

	// Return a copy
	items := make([]*ConstructionItem, len(queue.Items))
	copy(items, queue.Items)
	return items
}

// GetActiveConstruction returns the currently building item for a location
func (cs *ConstructionSystem) GetActiveConstruction(location string) *ConstructionItem {
	queue, exists := cs.queues.Get(location)
	if !exists {
		return nil
	}

	queue.mutex.RLock()
	defer queue.mutex.RUnlock()

	if len(queue.Items) > 0 {
		return queue.Items[0]
	}
	return nil
}

// GetTotalConstructions returns the total number of items being constructed
func (cs *ConstructionSystem) GetTotalConstructions() int {
	total := 0
	cs.queues.ForEach(func(location string, queue *ConstructionQueue) {
		queue.mutex.RLock()
		total += len(queue.Items)
		queue.mutex.RUnlock()
	})
	return total
}

// GetConstructionsByOwner returns all construction items for a specific owner
func (cs *ConstructionSystem) GetConstructionsByOwner(owner string) []*ConstructionItem {
	var items []*ConstructionItem
	var mu sync.Mutex

	cs.queues.ForEach(func(location string, queue *ConstructionQueue) {
		queue.mutex.RLock()
		defer queue.mutex.RUnlock()

		for _, item := range queue.Items {
			if item.Owner == owner {
				mu.Lock()
				items = append(items, item)
				mu.Unlock()
			}
		}
	})

	return items
}

// ClearQueue removes all items from a location's queue
func (cs *ConstructionSystem) ClearQueue(location string) {
	queue, exists := cs.queues.Get(location)
	if !exists {
		return
	}

	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	queue.Items = make([]*ConstructionItem, 0)
}

// PauseConstruction pauses construction at a location (for future use)
func (cs *ConstructionSystem) PauseConstruction(location string) {
	// Future implementation: add paused flag to queue
}

// ResumeConstruction resumes construction at a location (for future use)
func (cs *ConstructionSystem) ResumeConstruction(location string) {
	// Future implementation: remove paused flag from queue
}
