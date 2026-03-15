package game

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

// ColonizePlanet transfers ownership of an unclaimed planet to a player,
// sets up colony infrastructure, and consumes the colony ship's colonists.
// This is the single source of truth for colonization — used by both player
// commands and AI logistics.
func ColonizePlanet(planet *entities.Planet, ship *entities.Ship, player *entities.Player, systemID int) {
	planet.Owner = player.Name
	planet.Population = int64(ship.Colonists)
	planet.SetBaseOwner(player.Name)
	player.AddOwnedPlanet(planet)

	// Mark all resources on the planet as owned
	for _, resEntity := range planet.Resources {
		if res, ok := resEntity.(*entities.Resource); ok {
			res.Owner = player.Name
		}
	}

	// Ensure key deposits exist for production chains
	EnsureResourceDeposit(planet, entities.ResRareMetals, player.Name)
	EnsureResourceDeposit(planet, entities.ResHelium3, player.Name)

	// Build infrastructure
	AddBuildingToPlanet(planet, entities.BuildingTradingPost, player.Name, systemID)
	AddBuildingToPlanet(planet, entities.BuildingRefinery, player.Name, systemID)
	AddBuildingToPlanet(planet, entities.BuildingGenerator, player.Name, systemID)
	BuildMinesOnResources(planet, player.Name, systemID)
	SeedInitialCommodities(planet, player.Name)

	// Starting resources
	planet.AddStoredResource(entities.ResFuel, 100)
	planet.AddStoredResource(entities.ResWater, 100)

	// Consume the colony ship
	ship.Colonists = 0
	ship.Status = entities.ShipStatusOrbiting

	planet.RebalanceWorkforce()

	fmt.Printf("[Colonize] %s colonized %s (pop %d, sys %d)\n",
		player.Name, planet.Name, planet.Population, systemID)
}
