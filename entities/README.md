# Entity System

This directory contains the extensible entity system for Xandaris II. The entity system allows for easy creation and management of various objects within star systems (planets, stations, asteroids, fleets, etc.).

## Architecture

The entity system uses a **generator registry pattern** that allows new entity types to be added without modifying any central registry code. Each entity type self-registers on initialization.

### Core Components

- **`types.go`**: Core interfaces and base types
  - `Entity` interface: All entities must implement this
  - `EntityType` enum: Categories of entities (Planet, Station, Fleet, etc.)
  - `BaseEntity` struct: Common functionality for all entities

- **`registry.go`**: Generator registry system
  - `EntityGenerator` interface: All generators must implement this
  - `RegisterGenerator()`: Auto-called by generators in `init()`
  - `GenerateEntitiesForSystem()`: Main entry point for system generation

- **`planet_entity.go`**: Planet entity implementation
- **`station_entity.go`**: Station entity implementation

### Directory Structure

```
entities/
├── README.md                 # This file
├── types.go                  # Core types and interfaces
├── registry.go               # Generator registry
├── planet_entity.go          # Planet entity struct
├── station_entity.go         # Station entity struct
├── planet/                   # Planet generators
│   ├── terrestrial.go        # Terrestrial planet generator
│   ├── gas_giant.go          # Gas giant generator
│   ├── ice.go                # Ice world generator
│   ├── ocean.go              # Ocean world generator
│   └── lava.go               # Lava planet generator
└── station/                  # Station generators
    ├── trading.go            # Trading station generator
    ├── military.go           # Military station generator
    └── research.go           # Research station generator
```

## How It Works

### 1. Entity Interface

All entities implement the `Entity` interface:

```go
type Entity interface {
    GetID() int
    GetName() string
    GetType() EntityType
    GetSubType() string
    GetOrbitDistance() float64
    GetOrbitAngle() float64
    GetColor() color.RGBA
    GetDescription() string
    GetAbsolutePosition() (x, y float64)
    SetAbsolutePosition(x, y float64)
    GetClickRadius() float64
}
```

### 2. Generator Interface

Entity generators implement the `EntityGenerator` interface:

```go
type EntityGenerator interface {
    Generate(params GenerationParams) Entity
    GetWeight() float64           // Spawn probability weight
    GetEntityType() EntityType    // "Planet", "Station", etc.
    GetSubType() string           // "Terrestrial", "Trading", etc.
}
```

### 3. Auto-Registration

Each generator self-registers in its `init()` function:

```go
func init() {
    entities.RegisterGenerator(&TerrestrialGenerator{})
}
```

### 4. System Generation

When a system is created, the entity system:
1. Gets all registered generators for each entity type
2. Selects generators based on weights (probability)
3. Generates entities with appropriate parameters
4. Returns a slice of entities

## Adding New Entity Types

### Option 1: Add a New Generator for Existing Type

To add a new planet type (e.g., Desert World):

1. Create `entities/planet/desert.go`:

```go
package planet

import (
    "fmt"
    "image/color"
    "math/rand"
    "github.com/hunterjsb/xandaris/entities"
)

func init() {
    entities.RegisterGenerator(&DesertGenerator{})
}

type DesertGenerator struct{}

func (g *DesertGenerator) GetWeight() float64 {
    return 10.0 // Probability weight
}

func (g *DesertGenerator) GetEntityType() entities.EntityType {
    return entities.EntityTypePlanet
}

func (g *DesertGenerator) GetSubType() string {
    return "Desert"
}

func (g *DesertGenerator) Generate(params entities.GenerationParams) entities.Entity {
    id := params.SystemID*1000 + rand.Intn(1000)
    name := fmt.Sprintf("Dune %d", rand.Intn(100)+1)
    
    planetColor := color.RGBA{
        R: uint8(220 + rand.Intn(35)),
        G: uint8(200 + rand.Intn(35)),
        B: uint8(120 + rand.Intn(50)),
        A: 255,
    }
    
    planet := entities.NewPlanet(
        id, name, "Desert",
        params.OrbitDistance,
        params.OrbitAngle,
        planetColor,
    )
    
    planet.Size = 5 + rand.Intn(2)
    planet.Temperature = 30 + rand.Intn(70) // 30-100°C
    planet.Atmosphere = "Thin"
    planet.Population = int64(rand.Intn(100000000))
    planet.Resources = []string{"Silicon", "Sand", "Solar Energy"}
    planet.Habitability = 20 + rand.Intn(40) // 20-60%
    
    return planet
}
```

2. Import it in `main.go`:
```go
_ "github.com/hunterjsb/xandaris/entities/planet"
```

That's it! The new desert planet type will automatically appear in systems.

### Option 2: Add a Completely New Entity Category

To add a new entity category (e.g., Asteroids):

1. Add the entity type to `types.go`:
```go
const (
    EntityTypePlanet   EntityType = "Planet"
    EntityTypeStation  EntityType = "Station"
    EntityTypeFleet    EntityType = "Fleet"
    EntityTypeAsteroid EntityType = "Asteroid"  // New!
)
```

2. Create the entity struct in `entities/asteroid_entity.go`:
```go
package entities

type Asteroid struct {
    BaseEntity
    Composition string
    Size int
    Value int
}

func NewAsteroid(id int, name string, orbitDistance, orbitAngle float64, c color.RGBA) *Asteroid {
    return &Asteroid{
        BaseEntity: BaseEntity{
            ID: id,
            Name: name,
            Type: EntityTypeAsteroid,
            SubType: "Asteroid",
            Color: c,
            OrbitDistance: orbitDistance,
            OrbitAngle: orbitAngle,
        },
    }
}

// Implement required methods...
```

3. Create generators in `entities/asteroid/` directory

4. Update `registry.go` to generate asteroids in `GenerateEntitiesForSystem()`

5. Update rendering code in `system_view.go` to draw asteroids

## Entity Properties

### Planet Properties
- **Size**: Visual radius (pixels)
- **PlanetType**: Subtype (Terrestrial, Gas Giant, Ice, Ocean, Lava)
- **Population**: Number of inhabitants
- **Resources**: Available resources for mining/trading
- **Temperature**: In Celsius
- **Atmosphere**: Type (Breathable, Toxic, Thin, Dense, None, Corrosive)
- **HasRings**: Boolean for planetary rings
- **Habitability**: Score 0-100

### Station Properties
- **StationType**: Subtype (Trading, Military, Research)
- **Capacity**: Maximum population
- **CurrentPop**: Current population
- **Services**: Available services
- **Owner**: Owning faction
- **TradeGoods**: Available trade items
- **DefenseLevel**: Defense rating 0-10

## Weight System

Generators use a weight system for probability:

- **15.0**: Very common (Terrestrial planets)
- **12.0**: Common (Trading stations, Ocean worlds)
- **10.0**: Fairly common (Gas giants)
- **8.0**: Moderate (Ice worlds)
- **7.0**: Less common (Research stations)
- **6.0**: Uncommon (Military stations)
- **5.0**: Rare (Lava planets)

Higher weights = more likely to appear.

## Context Menus

Entities that implement `ContextMenuProvider` interface can display context menus:

```go
type ContextMenuProvider interface {
    GetContextMenuTitle() string
    GetContextMenuItems() []string
}
```

Both Planet and Station implement this for right-click information panels.

## Best Practices

1. **Keep generators small**: One generator per subtype
2. **Use meaningful weights**: Balance gameplay and realism
3. **Random variation**: Add randomness to make each entity unique
4. **Reasonable values**: Keep population, temperature, etc. realistic
5. **Test thoroughly**: Ensure new entities work with existing UI

## Future Enhancements

Potential additions to the entity system:

- **Fleets**: Mobile entities that can travel between systems
- **Asteroids**: Mineable resources
- **Anomalies**: Special locations (black holes, nebulae)
- **Wormholes**: Fast travel points
- **Derelicts**: Abandoned ships/stations
- **Resources**: Harvestable materials in space

## Examples

### Viewing Entity Statistics

```go
// In system generation
stats := entities.GetRegistryStats()
for entityType, count := range stats {
    fmt.Printf("%s generators: %d\n", entityType, count)
}
```

### Filtering Entities

```go
// Get all planets in a system
planets := system.GetEntitiesByType(entities.EntityTypePlanet)

// Get all trading stations
for _, entity := range system.Entities {
    if entity.GetType() == entities.EntityTypeStation && 
       entity.GetSubType() == "Trading" {
        // Do something with trading station
    }
}
```

### Custom Generation Parameters

```go
// Generate a specific planet at a specific location
params := entities.GenerationParams{
    SystemID:      42,
    OrbitDistance: 100.0,
    OrbitAngle:    1.57, // 90 degrees
    SystemSeed:    12345,
}

generator := &planet.TerrestrialGenerator{}
planet := generator.Generate(params)
```

## Troubleshooting

### New generator not appearing
- Check `init()` function exists and calls `RegisterGenerator()`
- Ensure package is imported in `main.go` with `_` prefix
- Verify weight is > 0

### Entities not rendering
- Check `GetAbsolutePosition()` is returning valid coordinates
- Ensure entity color has alpha = 255
- Verify `GetClickRadius()` returns a positive value

### Context menu not showing
- Implement `GetContextMenuTitle()` and `GetContextMenuItems()`
- Ensure entity implements both `Entity` and `ContextMenuProvider`
- Check click detection radius is reasonable

## License

Part of Xandaris II - Space Trading Game
