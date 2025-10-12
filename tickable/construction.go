package tickable

import (
	"bytes"
	"encoding/gob"
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
	Location       string       // Planet/Station ID
	Owner          string       // Player name
	Progress       int          // Current progress (0-100)
	TotalTicks     int          // Ticks required to complete
	RemainingTicks int          // Ticks remaining
	Cost           int          // Credit cost
	Started        int64        // Tick when started
	Mutex          sync.RWMutex `gob:"-"` // Don't serialize mutex
}

// GobEncode implements gob.GobEncoder to exclude Mutex from serialization
func (ci *ConstructionItem) GobEncode() ([]byte, error) {
	// Create a temporary struct without the mutex
	temp := struct {
		ID             string
		Type           string
		Name           string
		Location       string
		Owner          string
		Progress       int
		TotalTicks     int
		RemainingTicks int
		Cost           int
		Started        int64
	}{
		ID:             ci.ID,
		Type:           ci.Type,
		Name:           ci.Name,
		Location:       ci.Location,
		Owner:          ci.Owner,
		Progress:       ci.Progress,
		TotalTicks:     ci.TotalTicks,
		RemainingTicks: ci.RemainingTicks,
		Cost:           ci.Cost,
		Started:        ci.Started,
	}

	// Use gob to encode the temp struct
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(&temp); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder to reconstruct without mutex serialization
func (ci *ConstructionItem) GobDecode(data []byte) error {
	// Create a temporary struct to decode into
	temp := struct {
		ID             string
		Type           string
		Name           string
		Location       string
		Owner          string
		Progress       int
		TotalTicks     int
		RemainingTicks int
		Cost           int
		Started        int64
	}{}

	// Decode from gob
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&temp); err != nil {
		return err
	}

	// Copy fields to the actual struct
	ci.ID = temp.ID
	ci.Type = temp.Type
	ci.Name = temp.Name
	ci.Location = temp.Location
	ci.Owner = temp.Owner
	ci.Progress = temp.Progress
	ci.TotalTicks = temp.TotalTicks
	ci.RemainingTicks = temp.RemainingTicks
	ci.Cost = temp.Cost
	ci.Started = temp.Started
	// Mutex is already initialized (zero value)

	return nil
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

// CompletionHandler is a function that handles construction completions
type CompletionHandler func(completion ConstructionCompletion)

// ConstructionSystem handles all construction queues
type ConstructionSystem struct {
	*BaseSystem
	queues      *SafeMap[string, *ConstructionQueue]
	completions chan ConstructionCompletion
	handlers    []CompletionHandler
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
	item.Mutex.Lock()
	defer item.Mutex.Unlock()

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
	// Notify all registered handlers
	cs.mutex.RLock()
	handlers := cs.handlers
	cs.mutex.RUnlock()

	for _, handler := range handlers {
		handler(completion)
	}
}

// GetAllQueues returns all construction queues for saving
func (cs *ConstructionSystem) GetAllQueues() map[string][]*ConstructionItem {
	result := make(map[string][]*ConstructionItem)

	cs.queues.ForEach(func(location string, queue *ConstructionQueue) {
		queue.mutex.RLock()
		defer queue.mutex.RUnlock()

		// Copy items
		items := make([]*ConstructionItem, len(queue.Items))
		copy(items, queue.Items)
		result[location] = items
	})

	return result
}

// RestoreQueues restores construction queues from saved data
func (cs *ConstructionSystem) RestoreQueues(savedQueues map[string][]*ConstructionItem) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// Clear existing queues
	cs.queues = NewSafeMap[string, *ConstructionQueue]()

	// Restore each queue
	for location, items := range savedQueues {
		if len(items) > 0 {
			queue := &ConstructionQueue{
				Location: location,
				Items:    items,
			}
			cs.queues.Set(location, queue)
		}
	}
}

// RegisterCompletionHandler registers a handler for construction completions
func (cs *ConstructionSystem) RegisterCompletionHandler(handler CompletionHandler) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.handlers = append(cs.handlers, handler)
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
// HasMineInQueue checks if there's a mine being constructed for a specific resource location
func (cs *ConstructionSystem) HasMineInQueue(resourceLocation string) bool {
	queue, exists := cs.queues.Get(resourceLocation)
	if !exists {
		return false
	}

	queue.mutex.RLock()
	defer queue.mutex.RUnlock()

	// Check if any item in the queue is a mine
	for _, item := range queue.Items {
		if item.Name == "Mining Complex" {
			return true
		}
	}
	return false
}

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
