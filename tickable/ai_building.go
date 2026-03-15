package tickable

import (
	"fmt"

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

// BuildingAdder is implemented by the game server to let AI build things.
type BuildingAdder interface {
	AIBuildOnPlanet(planet *entities.Planet, buildingType string, owner string, systemID int)
}

func (abs *AIBuildingSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := abs.GetContext()
	if ctx == nil {
		return
	}

	gameObj := ctx.GetGame()
	if gameObj == nil {
		return
	}

	builder, ok := gameObj.(BuildingAdder)
	if !ok {
		return
	}

	logger, _ := gameObj.(EventLogger)

	playersIface := ctx.GetPlayers()
	if playersIface == nil {
		return
	}
	players, ok := playersIface.([]*entities.Player)
	if !ok {
		return
	}

	mp, ok := gameObj.(MarketProvider)
	if !ok {
		return
	}
	market := mp.GetMarketEngine()
	if market == nil {
		return
	}

	sp, ok := gameObj.(SystemsProvider)
	if !ok {
		return
	}
	systems := sp.GetSystems()

	for _, player := range players {
		if player == nil || player.IsHuman() {
			continue
		}
		abs.evaluateInvestment(player, market, builder, systems, logger)
	}
}

func (abs *AIBuildingSystem) evaluateInvestment(player *entities.Player, market interface{ GetBuyPrice(string) float64 }, builder BuildingAdder, systems []*entities.System, logger EventLogger) {
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
					if b.BuildingType == "Mine" && b.AttachedTo == resIDStr {
						mineCount++
					}
				}
			}

			if mineCount == 0 && player.Credits >= 500 {
				player.Credits -= 500
				builder.AIBuildOnPlanet(planet, "Mine", player.Name, systemID)
				for i := len(planet.Buildings) - 1; i >= 0; i-- {
					if b, ok := planet.Buildings[i].(*entities.Building); ok {
						if b.BuildingType == "Mine" && b.AttachedTo == fmt.Sprintf("%d", planet.GetID()) {
							b.AttachedTo = resIDStr
							b.AttachmentType = "Resource"
							break
						}
					}
				}
				msg := fmt.Sprintf("%s built mine on %s at %s", player.Name, res.ResourceType, planet.Name)
				fmt.Printf("[AIBuild] %s\n", msg)
				logBuildEvent(logger, player.Name, msg)
				return
			}
		}

		// PRIORITY 2: Build Generator for power (critical for productivity)
		if !hasBuilding(planet, "Generator") && planet.Population > 1000 && player.Credits >= 1000 {
			player.Credits -= 1000
			builder.AIBuildOnPlanet(planet, "Generator", player.Name, systemID)
			logBuildEvent(logger, player.Name, fmt.Sprintf("%s built Generator at %s", player.Name, planet.Name))
			return
		}

		// PRIORITY 2b: Build Fusion Reactor if we have He-3 and high population
		hasHe3Mine := false
		for _, resEntity := range planet.Resources {
			if res, ok := resEntity.(*entities.Resource); ok && res.ResourceType == "Helium-3" {
				resIDStr := fmt.Sprintf("%d", res.GetID())
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == "Mine" && b.AttachedTo == resIDStr {
						hasHe3Mine = true
						break
					}
				}
			}
		}
		if hasHe3Mine && !hasBuilding(planet, "Fusion Reactor") && planet.Population > 5000 && player.Credits >= 3000 {
			player.Credits -= 3000
			builder.AIBuildOnPlanet(planet, "Fusion Reactor", player.Name, systemID)
			logBuildEvent(logger, player.Name, fmt.Sprintf("%s built Fusion Reactor at %s", player.Name, planet.Name))
			return
		}

		// PRIORITY 3: Build Refinery if we have Oil mines but no refinery
		hasOilMine := false
		for _, resEntity := range planet.Resources {
			if res, ok := resEntity.(*entities.Resource); ok && res.ResourceType == "Oil" {
				resIDStr := fmt.Sprintf("%d", res.GetID())
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == "Mine" && b.AttachedTo == resIDStr {
						hasOilMine = true
						break
					}
				}
			}
		}
		refineryCount := countBuildings(planet, "Refinery")
		if hasOilMine && refineryCount == 0 && player.Credits >= 1500 {
			player.Credits -= 1500
			builder.AIBuildOnPlanet(planet, "Refinery", player.Name, systemID)
			fmt.Printf("[AIBuild] %s built refinery at %s\n", player.Name, planet.Name)
			logBuildEvent(logger, player.Name, fmt.Sprintf("%s built Refinery at %s", player.Name, planet.Name))
			return
		}

		// PRIORITY 3: Build Factory if we have Rare Metals and Iron mines
		hasRMmine := false
		hasIronMine := false
		for _, resEntity := range planet.Resources {
			if res, ok := resEntity.(*entities.Resource); ok {
				resIDStr := fmt.Sprintf("%d", res.GetID())
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == "Mine" && b.AttachedTo == resIDStr {
						if res.ResourceType == "Rare Metals" {
							hasRMmine = true
						}
						if res.ResourceType == "Iron" {
							hasIronMine = true
						}
					}
				}
			}
		}
		factoryCount := countBuildings(planet, "Factory")
		if hasRMmine && hasIronMine && factoryCount == 0 && player.Credits >= 2000 {
			player.Credits -= 2000
			builder.AIBuildOnPlanet(planet, "Factory", player.Name, systemID)
			fmt.Printf("[AIBuild] %s built factory at %s\n", player.Name, planet.Name)
			logBuildEvent(logger, player.Name, fmt.Sprintf("%s built Factory at %s", player.Name, planet.Name))
			return
		}

		// PRIORITY 4: Build Habitat when population at 70%+ capacity
		capacity := planet.GetTotalPopulationCapacity()
		if capacity > 0 && planet.Population > int64(float64(capacity)*0.7) && player.Credits >= 800 {
			player.Credits -= 800
			builder.AIBuildOnPlanet(planet, "Habitat", player.Name, systemID)
			fmt.Printf("[AIBuild] %s built habitat at %s (pop %d/%d)\n",
				player.Name, planet.Name, planet.Population, capacity)
			return
		}

		// PRIORITY 5: Build Trading Post if missing
		if !hasBuilding(planet, "Trading Post") && player.Credits >= 1200 {
			player.Credits -= 1200
			builder.AIBuildOnPlanet(planet, "Trading Post", player.Name, systemID)
			logBuildEvent(logger, player.Name, fmt.Sprintf("%s built Trading Post at %s", player.Name, planet.Name))
			return
		}

		// PRIORITY 6: Upgrade mines on expensive resources
		for _, resEntity := range planet.Resources {
			res, ok := resEntity.(*entities.Resource)
			if !ok || res.Owner != player.Name {
				continue
			}
			buyPrice := market.GetBuyPrice(res.ResourceType)
			basePrice := getBasePrice(res.ResourceType)
			resIDStr := fmt.Sprintf("%d", res.GetID())

			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == "Mine" && b.AttachedTo == resIDStr && b.CanUpgrade() {
						if buyPrice > basePrice*1.2 && player.Credits >= b.GetUpgradeCost() {
							player.Credits -= b.GetUpgradeCost()
							b.Upgrade()
							msg := fmt.Sprintf("%s upgraded %s mine to L%d at %s", player.Name, res.ResourceType, b.Level, planet.Name)
							fmt.Printf("[AIBuild] %s\n", msg)
							logBuildEvent(logger, player.Name, msg)
							return
						}
					}
				}
			}
		}

		// PRIORITY 7: Build Shipyard when affordable
		if !hasBuilding(planet, "Shipyard") && player.Credits >= 2500 {
			player.Credits -= 2000
			builder.AIBuildOnPlanet(planet, "Shipyard", player.Name, systemID)
			logBuildEvent(logger, player.Name, fmt.Sprintf("%s built Shipyard at %s", player.Name, planet.Name))
			return
		}

		// PRIORITY 8: Build Colony ship for expansion
		if len(player.OwnedPlanets) < 3 && hasBuilding(planet, "Shipyard") && player.Credits >= 3000 {
			hasColony := false
			for _, ship := range player.OwnedShips {
				if ship != nil && ship.ShipType == entities.ShipTypeColony {
					hasColony = true
					break
				}
			}
			if !hasColony && planet.GetStoredAmount("Iron") >= 100 &&
				planet.GetStoredAmount("Fuel") >= 50 &&
				planet.GetStoredAmount("Rare Metals") >= 20 {
				player.Credits -= 2000
				planet.RemoveStoredResource("Iron", 100)
				planet.RemoveStoredResource("Fuel", 50)
				planet.RemoveStoredResource("Rare Metals", 20)
				location := fmt.Sprintf("planet_%d", planet.GetID())
				if constructionSystem := GetSystemByName("Construction"); constructionSystem != nil {
					if cs, ok := constructionSystem.(*ConstructionSystem); ok {
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
						logBuildEvent(logger, player.Name, fmt.Sprintf("%s building Colony ship at %s", player.Name, planet.Name))
					}
				}
				return
			}
		}

		// PRIORITY 9: Build second refinery if we have Oil surplus
		if hasOilMine && refineryCount == 1 && planet.GetStoredAmount("Oil") > 200 && player.Credits >= 1500 {
			player.Credits -= 1500
			builder.AIBuildOnPlanet(planet, "Refinery", player.Name, systemID)
			logBuildEvent(logger, player.Name, fmt.Sprintf("%s built Refinery #2 at %s", player.Name, planet.Name))
			return
		}
	}
}

func logBuildEvent(logger EventLogger, player, msg string) {
	if logger != nil {
		logger.LogEvent("build", player, msg)
	}
}

func countBuildings(planet *entities.Planet, buildingType string) int {
	count := 0
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok {
			if b.BuildingType == buildingType {
				count++
			}
		}
	}
	return count
}

func hasBuilding(planet *entities.Planet, buildingType string) bool {
	return countBuildings(planet, buildingType) > 0
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

// getBasePrice returns the equilibrium price for a resource.
// Duplicated from economy.BasePrices — can't import economy from tickable (cycle).
func getBasePrice(resourceType string) float64 {
	switch resourceType {
	case "Iron":
		return 75
	case "Water":
		return 100
	case "Oil":
		return 150
	case "Fuel":
		return 200
	case "Rare Metals":
		return 500
	case "Helium-3":
		return 600
	case "Electronics":
		return 800
	}
	return 100
}
