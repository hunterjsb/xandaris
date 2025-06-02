# XANDARIS Simple Worldgen System

## Overview

The XANDARIS worldgen system generates star systems and planets using a simple table-based probability distribution. This replaces the complex CSV-based Python system with a clean, efficient Go implementation.

## World Type Distribution Table

| System Type | Type Probability | Food (cap 8) | Ore (Cap 8) | Oil (Cap 5) | Ti (Cap 2) | Xanium | Notes |
|-------------|------------------|---------------|--------------|-------------|------------|--------|-------|
| Abundant    | 0.0125          | 0.25         | 0.25        | 0.2        | 0.1       | 0      |       |
| Fertile     | 0.05            | 0.8          | 0.2         | 0          | 0         | 0      |       |
| Mountain    | 0.05            | 0            | 0.5         | 0.2        | 0.1       | 0      |       |
| Desert      | 0.025           | 0            | 0           | 0.5        | 0.15      | 0      |       |
| Volcanic    | 0.025           | 0            | 0.5         | 0.4        | 0         | 0      |       |
| Highlands   | 0.0375          | 0.5          | 0.5         | 0          | 0         | 0      |       |
| Swamp       | 0.0375          | 0.5          | 0           | 0.2        | 0.1       | 0      |       |
| Barren      | 0.005           | 0            | 0.1         | 0.12       | 0.2       | 0.025  |       |
| Radiant     | 0.00625         | 0            | 0           | 0          | 0.4       | 0      |       |
| Barred      | 0.00125         | N/A          | N/A         | N/A        | N/A       | N/A    | Always exactly 2 Xanium deposits |
| Null        | 0.75            | N/A          | N/A         | N/A        | N/A       | N/A    | Won't display to user |

## Implementation

### Go Code Location
- `internal/worldgen/worldgen.go` - Main implementation
- Integrated with existing PocketBase database schema
- Used by seeding process and API endpoints

### API Endpoints
- `GET /api/worldgen` - Generate random system
- `GET /api/worldgen/{seed}` - Generate system from specific seed

### Resource Generation Logic
1. **World Type Selection**: Random roll against cumulative probability distribution
2. **Resource Generation**: For each resource type, if probability > random roll, generate 1-cap resources
3. **Special Cases**: 
   - Barred worlds always have exactly 2 Xanium deposits
   - Null worlds have no resources and don't display to users

### Name Generation
- **System Names**: Greek letters + numbers + constellation names (e.g., "Alpha 42 Centauri", "Beta-789")
- **Planet Names**: Theme-based with world type specific vocabularies + suffixes

## Key Features

- **Simple**: Table-driven design, easy to modify probabilities
- **Fast**: Pure Go implementation, no external dependencies
- **Deterministic**: Same seed always produces same results
- **Integrated**: Works seamlessly with existing game database
- **Realistic**: Based on resource scarcity and world type logic

## Usage Example

```go
// Generate random system
seed32 := worldgen.GenerateRandomSystemSeed()
system := worldgen.ProcessSystemSeed(seed32)

// Generate specific system
seed32 := big.NewInt(12345678901234567890)
system := worldgen.ProcessSystemSeed(seed32)
```

## Sample Output

```json
{
  "system_seed": "0003a2f14a7043ee3709",
  "system_name": "Zeta 32 Piscium",
  "system_planets": [
    {
      "planet_name": "Grove Gamma",
      "planet_type": "Fertile",
      "planet_resources": [8, 0, 0, 0, 0]
    }
  ]
}
```

Resource array format: `[Food, Ore, Oil, Titanium, Xanium]`