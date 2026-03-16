package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AIBuildingSystem{
		BaseSystem: NewBaseSystem("AIBuilding", 30),
	})
}

// AIBuildingSystem decides when AI players should invest in infrastructure.
type AIBuildingSystem struct {
	*BaseSystem
}

func (abs *AIBuildingSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := abs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()

	if game.GetMarketEngine() == nil {
		return
	}

	systems := game.GetSystems()

	for _, player := range players {
		if player == nil || player.IsHuman() {
			continue
		}
		abs.evaluateInvestment(player, game, systems)
	}
}

func (abs *AIBuildingSystem) evaluateInvestment(player *entities.Player, game GameProvider, systems []*entities.System) {
	if player.Credits < 300 {
		return
	}

	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}

		systemID := findSystemIDForPlanet(planet, systems)

		// PRIORITY 1: Mine ALL unmined deposits (most important — resources drive everything)
		for _, resEntity := range planet.Resources {
			res, ok := resEntity.(*entities.Resource)
			if !ok || res.Owner != player.Name || res.Abundance <= 0 {
				continue
			}

			resIDStr := fmt.Sprintf("%d", res.GetID())
			mineCount := 0
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == entities.BuildingMine && b.AttachedTo == resIDStr {
						mineCount++
					}
				}
			}

			if mineCount == 0 && player.Credits >= 500 {
				player.Credits -= 500
				game.AIBuildOnPlanet(planet, entities.BuildingMine, player.Name, systemID)
				for i := len(planet.Buildings) - 1; i >= 0; i-- {
					if b, ok := planet.Buildings[i].(*entities.Building); ok {
						if b.BuildingType == entities.BuildingMine && b.AttachedTo == fmt.Sprintf("%d", planet.GetID()) {
							b.AttachedTo = resIDStr
							b.AttachmentType = "Resource"
							break
						}
					}
				}
				msg := fmt.Sprintf("%s built mine on %s at %s", player.Name, res.ResourceType, planet.Name)
				fmt.Printf("[AIBuild] %s\n", msg)
				logBuildEvent(game, player.Name, msg)
				return
			}
		}

		// PRIORITY 2: Build Trading Post ASAP (required for all trade)
		if !planet.HasBuilding(entities.BuildingTradingPost) && player.Credits >= 1200 {
			player.Credits -= 1200
			game.AIBuildOnPlanet(planet, entities.BuildingTradingPost, player.Name, systemID)
			logBuildEvent(game, player.Name, fmt.Sprintf("%s built Trading Post at %s", player.Name, planet.Name))
			return
		}

		// PRIORITY 3: Build Generators for power (critical for productivity)
		// Need enough generators to cover demand — build more if power ratio < 80%
		powerRatio := planet.GetPowerRatio()
		genCount := planet.CountBuildings(entities.BuildingGenerator)
		if (genCount == 0 || (powerRatio < 0.8 && genCount < 4)) && planet.Population > 500 && player.Credits >= 1000 {
			player.Credits -= 1000
			game.AIBuildOnPlanet(planet, entities.BuildingGenerator, player.Name, systemID)
			logBuildEvent(game, player.Name, fmt.Sprintf("%s built Generator #%d at %s (power %.0f%%)",
				player.Name, genCount+1, planet.Name, powerRatio*100))
			return
		}

		// PRIORITY 2b: Build Fusion Reactor if we have He-3 and high population
		if hasMinedResource(planet, entities.ResHelium3) && !planet.HasBuilding(entities.BuildingFusionReactor) && powerRatio < 0.9 && player.Credits >= 3000 {
			player.Credits -= 3000
			game.AIBuildOnPlanet(planet, entities.BuildingFusionReactor, player.Name, systemID)
			logBuildEvent(game, player.Name, fmt.Sprintf("%s built Fusion Reactor at %s", player.Name, planet.Name))
			return
		}

		// PRIORITY 3: Build Refinery if we have Oil mines but no refinery
		hasOilMine := hasMinedResource(planet, entities.ResOil)
		refineryCount := planet.CountBuildings(entities.BuildingRefinery)
		if hasOilMine && refineryCount == 0 && player.Credits >= 1500 {
			player.Credits -= 1500
			game.AIBuildOnPlanet(planet, entities.BuildingRefinery, player.Name, systemID)
			fmt.Printf("[AIBuild] %s built refinery at %s\n", player.Name, planet.Name)
			logBuildEvent(game, player.Name, fmt.Sprintf("%s built Refinery at %s", player.Name, planet.Name))
			return
		}

		// PRIORITY 3b: Build Factory if we have Rare Metals and Iron mines
		factoryCount := planet.CountBuildings(entities.BuildingFactory)
		if hasMinedResource(planet, entities.ResRareMetals) && hasMinedResource(planet, entities.ResIron) && factoryCount == 0 && player.Credits >= 2000 {
			player.Credits -= 2000
			game.AIBuildOnPlanet(planet, entities.BuildingFactory, player.Name, systemID)
			fmt.Printf("[AIBuild] %s built factory at %s\n", player.Name, planet.Name)
			logBuildEvent(game, player.Name, fmt.Sprintf("%s built Factory at %s", player.Name, planet.Name))
			return
		}

		// PRIORITY 4: Build Habitat when population at 85%+ capacity
		// Only build if power is stable (>60%) and max 5 habitats per planet
		capacity := planet.GetTotalPopulationCapacity()
		habitatCount := planet.CountBuildings(entities.BuildingHabitat)
		if capacity > 0 && planet.Population > int64(float64(capacity)*0.85) &&
			habitatCount < 5 && powerRatio > 0.6 && player.Credits >= 800 {
			player.Credits -= 800
			game.AIBuildOnPlanet(planet, entities.BuildingHabitat, player.Name, systemID)
			fmt.Printf("[AIBuild] %s built habitat at %s (pop %d/%d)\n",
				player.Name, planet.Name, planet.Population, capacity)
			return
		}

		// PRIORITY 6: Upgrade mines on expensive resources
		for _, resEntity := range planet.Resources {
			res, ok := resEntity.(*entities.Resource)
			if !ok || res.Owner != player.Name {
				continue
			}
			buyPrice := game.GetMarketEngine().GetBuyPrice(res.ResourceType)
			basePrice := economy.GetBasePrice(res.ResourceType)
			resIDStr := fmt.Sprintf("%d", res.GetID())

			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == entities.BuildingMine && b.AttachedTo == resIDStr && b.CanUpgrade() {
						if buyPrice > basePrice*1.2 && player.Credits >= b.GetUpgradeCost() {
							player.Credits -= b.GetUpgradeCost()
							b.Upgrade()
							msg := fmt.Sprintf("%s upgraded %s mine to L%d at %s", player.Name, res.ResourceType, b.Level, planet.Name)
							fmt.Printf("[AIBuild] %s\n", msg)
							logBuildEvent(game, player.Name, msg)
							return
						}
					}
				}
			}
		}

		// PRIORITY 7: Build Shipyard when affordable
		if !planet.HasBuilding(entities.BuildingShipyard) && player.Credits >= 2500 {
			player.Credits -= 2500
			game.AIBuildOnPlanet(planet, entities.BuildingShipyard, player.Name, systemID)
			logBuildEvent(game, player.Name, fmt.Sprintf("%s built Shipyard at %s", player.Name, planet.Name))
			return
		}

		// PRIORITY 8: Build Colony ship for expansion
		if len(player.OwnedPlanets) < 3 && planet.HasBuilding(entities.BuildingShipyard) && player.Credits >= 3000 {
			hasColony := false
			for _, ship := range player.OwnedShips {
				if ship != nil && ship.ShipType == entities.ShipTypeColony {
					hasColony = true
					break
				}
			}
			if !hasColony && planet.GetStoredAmount(entities.ResIron) >= 100 &&
				planet.GetStoredAmount(entities.ResFuel) >= 50 &&
				planet.GetStoredAmount(entities.ResRareMetals) >= 20 {
				player.Credits -= 2000
				planet.RemoveStoredResource(entities.ResIron, 100)
				planet.RemoveStoredResource(entities.ResFuel, 50)
				planet.RemoveStoredResource(entities.ResRareMetals, 20)
				location := fmt.Sprintf("planet_%d", planet.GetID())
				if cs := GetConstructionSystem(); cs != nil {
					item := &ConstructionItem{
						ID:             fmt.Sprintf("aiship_colony_%s_%d", player.Name, abs.GetContext().GetTick()),
						Type:           "Ship",
						Name:           string(entities.ShipTypeColony),
						Location:       location,
						Owner:          player.Name,
						Progress:       0,
						TotalTicks:     300,
						RemainingTicks: 300,
						Cost:           2000,
						Started:        abs.GetContext().GetTick(),
					}
					cs.AddToQueue(location, item)
					logBuildEvent(game, player.Name, fmt.Sprintf("%s building Colony ship at %s", player.Name, planet.Name))
				}
				return
			}
		}

		// PRIORITY 9: Build second refinery when generators need more Fuel
		fuelStored := planet.GetStoredAmount(entities.ResFuel)
		if hasOilMine && refineryCount < 3 && (fuelStored < 50 || refineryCount < genCount) && player.Credits >= 1500 {
			player.Credits -= 1500
			game.AIBuildOnPlanet(planet, entities.BuildingRefinery, player.Name, systemID)
			logBuildEvent(game, player.Name, fmt.Sprintf("%s built Refinery #2 at %s", player.Name, planet.Name))
			return
		}
	}
}

func logBuildEvent(game GameProvider, player, msg string) {
	game.LogEvent("build", player, msg)
}

// hasMinedResource returns true if the planet has a mine attached to a deposit of the given type.
func hasMinedResource(planet *entities.Planet, resourceType string) bool {
	for _, resEntity := range planet.Resources {
		if res, ok := resEntity.(*entities.Resource); ok && res.ResourceType == resourceType {
			resIDStr := fmt.Sprintf("%d", res.GetID())
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingMine && b.AttachedTo == resIDStr {
					return true
				}
			}
		}
	}
	return false
}

func findSystemIDForPlanet(planet *entities.Planet, systems []*entities.System) int {
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if p, ok := e.(*entities.Planet); ok && p.GetID() == planet.GetID() {
				return sys.ID
			}
		}
	}
	return 0
}
