# Entity System

A flexible, extensible entity generation system for Xandaris II.

## Overview

This system allows you to add new entity types and subtypes with **zero boilerplate** and **no central coordination**. Just drop in a new file and it works!

## Architecture

### Core Components

- **`registry.go`** - Central registry that tracks all entity generators
- **`EntityGenerator` interface** - What all entity generators must implement
- **Auto-registration** - Each entity file registers itself via `init()`

### Directory Structure

```
entities/
├── registry.go          # Core registry system
├── example.go           # Usage examples
├── planet/              # Planet entity types
│   ├── terrestrial.go   # Terrestrial planet generator
│   ├── lava.go          # Lava planet generator
│   ├── ocean.go         # (add more...)
│   └── ...
├── station/             # Station entity types
│   ├── military.go      # Military station generator
│   ├── trading.go       # Trading station generator
│   └── ...
└── ship/                # (future) Ship entity types
    └── ...
```

## How It Works

### 1. EntityGenerator Interface

Every entity generator implements this simple interface:

```go
type EntityGenerator interface {
    Generate(params GenerationParams) interface{}
    GetWeight() float64      // Spawn probability (higher = more common)
    GetEntityType() string   // "Planet", "Station", "Ship", etc.
    GetSubType() string      // "Military", "Lava", "Trading", etc.
}
```

### 2. Auto-Registration

Each entity file registers itself in `init()`:

```go
func init() {
    entities.RegisterGenerator(&MyEntityGenerator{})
}
```

### 3. Weighted Random Selection

The registry uses weights to randomly select entity types:
- Terrestrial planets: weight 15.0 (common)
- Lava planets: weight 5.0 (rare)
- Trading stations: weight 12.0 (very common)
- Military stations: weight 8.0 (fairly common)

## Adding a New Entity

### Example: Adding an Ocean Planet

**Step 1:** Create `entities/planet/ocean.go`

```go
package planet

import (
    "fmt"
    "math/rand"
    "github.com/hunterjsb/xandaris/entities"
)

func init() {
    entities.RegisterGenerator(&OceanGenerator{})
}

type OceanGenerator struct{}

func (g *OceanGenerator) GetWeight() float64 {
    return 10.0  // Moderately common
}

func (g *OceanGenerator) GetEntityType() string {
    return "Planet"
}

func (g *OceanGenerator) GetSubType() string {
    return "Ocean"
}

func (g *OceanGenerator) Generate(params entities.GenerationParams) interface{} {
    return struct {
        ID            int
        Name          string
        Type          string
        OrbitDistance float64
        OrbitAngle    float64
        Temperature   int
        Atmosphere    string
        Population    int64
    }{
        ID:            params.SystemID*1000 + rand.Intn(1000),
        Name:          fmt.Sprintf("Planet %d", rand.Intn(100)),
        Type:          "Ocean",
        OrbitDistance: params.OrbitDistance,
        OrbitAngle:    params.OrbitAngle,
        Temperature:   0 + rand.Intn(40),
        Atmosphere:    "Breathable",
        Population:    int64(rand.Intn(3000000000)),
    }
}
```

**Step 2:** That's it! The entity is now automatically:
- Registered in the system
- Available for generation
- Weighted for spawn probability

## Adding a New Entity Category

Want to add Ships? Asteroids? Anomalies?

**Step 1:** Create directory `entities/ship/`

**Step 2:** Create `entities/ship/fighter.go`

```go
package ship

import (
    "github.com/hunterjsb/xandaris/entities"
)

func init() {
    entities.RegisterGenerator(&FighterGenerator{})
}

type FighterGenerator struct{}

func (g *FighterGenerator) GetWeight() float64 {
    return 20.0  // Fighters are common
}

func (g *FighterGenerator) GetEntityType() string {
    return "Ship"  // New entity type!
}

func (g *FighterGenerator) GetSubType() string {
    return "Fighter"
}

func (g *FighterGenerator) Generate(params entities.GenerationParams) interface{} {
    // Create fighter ship
}
```

**Step 3:** Import the package to trigger registration:

```go
import (
    _ "github.com/hunterjsb/xandaris/entities/ship"
)
```

**Step 4:** (Optional) Add generation logic in `registry.go` if you want automatic spawning:

```go
// Generate ships (1-3 per system)
shipCount := 1 + rand.Intn(3)
shipGenerators := GetGeneratorsByType("Ship")
// ... generate ships
```

## Usage

```go
import (
    "github.com/hunterjsb/xandaris/entities"
    
    // Import to trigger auto-registration
    _ "github.com/hunterjsb/xandaris/entities/planet"
    _ "github.com/hunterjsb/xandaris/entities/station"
)

// Generate entities for a system
entities := entities.GenerateEntitiesForSystem(systemID, seed)

// Get all generators of a specific type
planetGens := entities.GetGeneratorsByType("Planet")

// Manual generation with custom parameters
params := entities.GenerationParams{
    SystemID:      42,
    OrbitDistance: 100.0,
    OrbitAngle:    3.14,
    SystemSeed:    12345,
}
planet := planetGen.Generate(params)
```

## Benefits

✅ **Zero boilerplate** - Just implement the interface and register  
✅ **No central coordination** - Each entity is self-contained  
✅ **Easy to extend** - Add new types without modifying existing code  
✅ **Weighted spawning** - Control rarity/commonality per entity  
✅ **Type-safe** - Uses Go interfaces  
✅ **Flexible** - Works for any entity category (planets, stations, ships, etc.)  

## Future Improvements

- Add entity validation
- Support for entity dependencies (e.g., "Guardian stations only spawn near military planets")
- Entity tags/categories for filtering
- Save/load entity definitions from JSON
- Mod support (load entities from external files)