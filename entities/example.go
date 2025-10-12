package entities

import (
	"fmt"
)

// Example shows how to use the entity generation system
func Example() {
	// Generate entities for a system
	// The registry automatically knows about all registered entity types
	systemID := 1
	seed := int64(12345)

	entities := GenerateEntitiesForSystem(systemID, seed)

	fmt.Printf("Generated %d entities\n", len(entities))

	// To add a NEW entity type:
	// 1. Create a new file like entities/planet/ocean.go
	// 2. Implement the EntityGenerator interface
	// 3. Call RegisterGenerator in init()
	// 4. Done! No other changes needed!

	// NOTE: To use this example, import entity packages in your main package:
	//   import _ "github.com/hunterjsb/xandaris/entities/planet"
	//   import _ "github.com/hunterjsb/xandaris/entities/station"

	// To add a NEW entity category (like "Ship" or "Asteroid"):
	// 1. Create entities/ship/ directory
	// 2. Create entities/ship/fighter.go (or whatever)
	// 3. Implement EntityGenerator interface
	// 4. Update GenerateEntitiesForSystem() if you want automatic generation
	// 5. Done!
}

// ShowAllRegisteredGenerators displays all registered entity generators
func ShowAllRegisteredGenerators() {
	generators := GetAllGenerators()

	fmt.Println("Registered Entity Generators:")
	for _, gen := range generators {
		fmt.Printf("  - %s/%s (weight: %.1f)\n",
			gen.GetEntityType(),
			gen.GetSubType(),
			gen.GetWeight())
	}

	// Output example:
	// Registered Entity Generators:
	//   - Planet/Terrestrial (weight: 15.0)
	//   - Planet/Lava (weight: 5.0)
	//   - Station/Military (weight: 8.0)
	//   - Station/Trading (weight: 12.0)
}
