package tickable

import (
	"sync"
)

// TickableSystem is the interface that all tickable game systems must implement
type TickableSystem interface {
	// GetName returns the name of this system
	GetName() string

	// GetPriority returns the execution priority (lower = earlier)
	// This determines the order systems are updated each tick
	GetPriority() int

	// OnTick is called every game tick
	OnTick(tick int64)

	// IsEnabled returns whether this system is currently active
	IsEnabled() bool

	// SetEnabled enables or disables this system
	SetEnabled(enabled bool)

	// Initialize is called when the system is first registered
	Initialize(context SystemContext)
}

// SystemContext provides access to game state for tickable systems
type SystemContext interface {
	GetGame() interface{}
	GetPlayers() interface{}
	GetTick() int64
}

// Registry holds all registered tickable systems
var registry []TickableSystem

// RegisterSystem adds a system to the registry
// This should be called in init() functions of system files
func RegisterSystem(system TickableSystem) {
	registry = append(registry, system)
	sortRegistryByPriority()
}

// sortRegistryByPriority sorts systems by their priority
func sortRegistryByPriority() {
	// Simple bubble sort (fine for small number of systems)
	n := len(registry)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if registry[j].GetPriority() > registry[j+1].GetPriority() {
				registry[j], registry[j+1] = registry[j+1], registry[j]
			}
		}
	}
}

// GetAllSystems returns all registered systems
func GetAllSystems() []TickableSystem {
	return registry
}

// GetSystemByName finds a system by name
func GetSystemByName(name string) TickableSystem {
	for _, system := range registry {
		if system.GetName() == name {
			return system
		}
	}
	return nil
}

// UpdateAllSystems updates all enabled systems for the given tick concurrently
func UpdateAllSystems(tick int64) {
	var wg sync.WaitGroup

	// Process all enabled systems concurrently
	for _, system := range registry {
		if system.IsEnabled() {
			wg.Add(1)
			// Capture system in local scope for goroutine
			sys := system
			go func() {
				defer wg.Done()
				sys.OnTick(tick)
			}()
		}
	}

	// Wait for all systems to complete
	wg.Wait()
}

// UpdateAllSystemsSequential updates systems one by one (useful for debugging)
func UpdateAllSystemsSequential(tick int64) {
	for _, system := range registry {
		if system.IsEnabled() {
			system.OnTick(tick)
		}
	}
}

// InitializeAllSystems initializes all registered systems with context
func InitializeAllSystems(context SystemContext) {
	for _, system := range registry {
		system.Initialize(context)
	}
}

// EnableSystem enables a system by name
func EnableSystem(name string) bool {
	system := GetSystemByName(name)
	if system != nil {
		system.SetEnabled(true)
		return true
	}
	return false
}

// DisableSystem disables a system by name
func DisableSystem(name string) bool {
	system := GetSystemByName(name)
	if system != nil {
		system.SetEnabled(false)
		return true
	}
	return false
}

// GetSystemCount returns the number of registered systems
func GetSystemCount() int {
	return len(registry)
}

// GetEnabledSystemCount returns the number of enabled systems
func GetEnabledSystemCount() int {
	count := 0
	for _, system := range registry {
		if system.IsEnabled() {
			count++
		}
	}
	return count
}

// ClearRegistry clears all registered systems (useful for testing)
func ClearRegistry() {
	registry = make([]TickableSystem, 0)
}
