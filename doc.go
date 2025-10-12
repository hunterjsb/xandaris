// Package main implements Xandaris II - A space trading and exploration game
//
// # Architecture Overview
//
// The codebase is organized into logical groups using file naming conventions:
//
// ## Core Game Files
//   - main.go          - Game entry point and main loop
//   - player.go        - Player management and state
//   - system.go        - Star system data structures
//   - entity.go        - Entity helper methods
//
// ## View Layer (MVC Pattern)
//   - view.go          - View interface and manager
//   - galaxy_view.go   - Galaxy map view (top-level)
//   - system_view.go   - System detail view (mid-level)
//   - planet_view.go   - Planet detail view (close-up)
//
// ## UI Components
//   - ui.go                     - Core UI primitives (panels, text, click handling)
//   - build_menu.go             - Building construction menu
//   - construction_queue_ui.go  - Active construction queue display
//   - colors.go                 - Color palette constants
//   - scaling.go                - Dynamic scaling system for rendering
//
// ## Game Systems
//   - tick.go          - Tick/time management system
//   - interfaces.go    - Interface definitions for dependency inversion
//
// ## External Packages
//   - entities/        - Entity system (planets, stations, resources, buildings, stars)
//   - tickable/        - Concurrent tickable systems (construction, resources, etc.)
//
// # Design Patterns
//
// ## Entity System
// Uses a registry pattern where entities self-register via init() functions.
// See entities/ package for details.
//
// ## Tickable Systems
// Concurrent game systems that process each tick in parallel using goroutines.
// See tickable/ package for details.
//
// ## Dependency Inversion
// UI components depend on interfaces (not concrete types) to avoid circular dependencies.
// Core game implements these interfaces and passes them to UI components.
//
// ## View Management
// Views implement a common interface and are managed by ViewManager for state transitions.
//
// # Adding New Features
//
// ## New Entity Type
// 1. Add EntityType constant to entities/types.go
// 2. Create entity struct in entities/
// 3. Create generators in entities/{type}/ directory
// 4. Import in main.go with _ import
//
// ## New Tickable System
// 1. Implement TickableSystem interface in tickable/
// 2. Register in init() function
// 3. System automatically runs concurrently each tick
//
// ## New UI Component
// 1. Create {component}_ui.go file
// 2. Accept GameContext interface instead of *Game
// 3. Implement Update() and Draw() methods
//
// ## New Building Type
// 1. Create generator in entities/building/
// 2. Set GetWeight() to 0.0 (buildings aren't auto-generated)
// 3. Add to build menu in build_menu.go
//
// # Key Concepts
//
// ## Concurrency
// The game uses Go's concurrency model extensively:
//   - Tickable systems run in parallel
//   - Entity processing uses worker pools
//   - Thread-safe data structures (SafeMap, SafeCounter)
//
// ## Scaling
// Dynamic scaling ensures entities fit on screen at all zoom levels:
//   - Galaxy view: systems are small dots
//   - System view: planets and orbits scale to screen
//   - Planet view: resources and buildings on surface
//
// ## Ownership
// Entities track ownership via Owner field:
//   - Visual rings indicate player ownership
//   - Resources/buildings inherit planet ownership
//   - Ready for multiplayer/AI expansion
//
// # Performance Considerations
//
//   - Entity generation is procedural (minimal memory)
//   - Concurrent tick processing scales with CPU cores
//   - UI only draws visible elements
//   - Construction uses channels for efficient completion handling
package main
