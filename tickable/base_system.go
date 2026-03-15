package tickable

import (
	"sync"
)

// BaseSystem provides common functionality for tickable systems
type BaseSystem struct {
	name     string
	priority int
	enabled  bool
	context  SystemContext
	mutex    sync.RWMutex
}

// NewBaseSystem creates a new base system
func NewBaseSystem(name string, priority int) *BaseSystem {
	return &BaseSystem{
		name:     name,
		priority: priority,
		enabled:  true,
	}
}

// GetName returns the system name
func (bs *BaseSystem) GetName() string {
	return bs.name
}

// GetPriority returns the execution priority
func (bs *BaseSystem) GetPriority() int {
	return bs.priority
}

// IsEnabled returns whether the system is enabled
func (bs *BaseSystem) IsEnabled() bool {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()
	return bs.enabled
}

// SetEnabled sets the enabled state
func (bs *BaseSystem) SetEnabled(enabled bool) {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()
	bs.enabled = enabled
}

// Initialize stores the system context
func (bs *BaseSystem) Initialize(context SystemContext) {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()
	bs.context = context
}

// GetContext returns the system context (thread-safe)
func (bs *BaseSystem) GetContext() SystemContext {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()
	return bs.context
}

// ProcessConcurrent processes a slice of items concurrently using worker pool
// This is the main helper for parallel processing
func ProcessConcurrent[T any](items []T, workerCount int, processFn func(T)) {
	if len(items) == 0 {
		return
	}

	// Limit worker count to number of items
	if workerCount > len(items) {
		workerCount = len(items)
	}

	// Create channel for work distribution
	jobs := make(chan T, len(items))
	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				processFn(item)
			}
		}()
	}

	// Send jobs to workers
	for _, item := range items {
		jobs <- item
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
}

// SafeMap provides thread-safe map operations
type SafeMap[K comparable, V any] struct {
	data  map[K]V
	mutex sync.RWMutex
}

// NewSafeMap creates a new thread-safe map
func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		data: make(map[K]V),
	}
}

// Set adds or updates a key-value pair
func (sm *SafeMap[K, V]) Set(key K, value V) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.data[key] = value
}

// Get retrieves a value by key
func (sm *SafeMap[K, V]) Get(key K) (V, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	val, ok := sm.data[key]
	return val, ok
}

// Delete removes a key-value pair
func (sm *SafeMap[K, V]) Delete(key K) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(sm.data, key)
}

// Len returns the number of items
func (sm *SafeMap[K, V]) Len() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return len(sm.data)
}

// ForEach iterates over all items (snapshot)
func (sm *SafeMap[K, V]) ForEach(fn func(K, V)) {
	sm.mutex.RLock()
	snapshot := make(map[K]V, len(sm.data))
	for k, v := range sm.data {
		snapshot[k] = v
	}
	sm.mutex.RUnlock()

	for k, v := range snapshot {
		fn(k, v)
	}
}
