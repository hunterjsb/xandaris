package game

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

// TransferPlanetOwnership moves a planet from its current owner to a new owner.
// Handles cleanup of the old owner's OwnedPlanets list. Safe to call with nil oldOwner.
func TransferPlanetOwnership(planet *entities.Planet, oldOwner, newOwner *entities.Player) {
	if oldOwner != nil {
		oldOwner.RemoveOwnedPlanet(planet)
	}
	planet.Owner = newOwner.Name
	planet.SetBaseOwner(newOwner.Name)
	newOwner.AddOwnedPlanet(planet)

	// Transfer resource ownership
	for _, resEntity := range planet.Resources {
		if res, ok := resEntity.(*entities.Resource); ok {
			res.Owner = newOwner.Name
		}
	}
}

// ColonizePlanet transfers ownership of an unclaimed planet to a player,
// sets up colony infrastructure, and consumes the colony ship's colonists.
// This is the single source of truth for colonization — used by both player
// commands and AI logistics.
func ColonizePlanet(planet *entities.Planet, ship *entities.Ship, player *entities.Player, systemID int) {
	TransferPlanetOwnership(planet, nil, player)
	planet.Population = int64(ship.Colonists)

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
