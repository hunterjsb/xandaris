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
	if player.Credits < 500 {
		return
	}

	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}

		systemID := findSystemIDForPlanet(planet, systems)

		// Strategy 1: Build additional mine on scarce resources, or upgrade existing mines
		for _, resEntity := range planet.Resources {
			res, ok := resEntity.(*entities.Resource)
			if !ok || res.Owner != player.Name {
				continue
			}

			buyPrice := market.GetBuyPrice(res.ResourceType)
			basePrice := getBasePrice(res.ResourceType)

			// Count existing mines on this resource
			resIDStr := fmt.Sprintf("%d", res.GetID())
			mineCount := 0
			var bestMine *entities.Building
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == "Mine" && b.AttachedTo == resIDStr {
						mineCount++
						if bestMine == nil || b.Level < bestMine.Level {
							bestMine = b
						}
					}
				}
			}

			// Build additional mine if resource is very expensive (price > 2x base)
			// Allow more mines during severe crises (> 3x base)
			maxMines := 3
			if buyPrice > basePrice*3.0 {
				maxMines = 4
			}
			if buyPrice > basePrice*2.0 && mineCount < maxMines && player.Credits >= 500 {
				player.Credits -= 500
				builder.AIBuildOnPlanet(planet, "Mine", player.Name, systemID)
				// Attach the new mine to this resource
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
				fmt.Printf("[AIBuild] %s (price %.0f, base %.0f)\n", msg, buyPrice, basePrice)
				logBuildEvent(logger, player.Name, msg)
				return
			}

			// Upgrade existing mine if resource is expensive (price > 1.5x base)
			if buyPrice > basePrice*1.5 && bestMine != nil && bestMine.CanUpgrade() && player.Credits >= 300 {
				player.Credits -= 300
				bestMine.Upgrade()
				msg := fmt.Sprintf("%s upgraded %s mine to L%d", player.Name, res.ResourceType, bestMine.Level)
				fmt.Printf("[AIBuild] %s at %s\n", msg, planet.Name)
				logBuildEvent(logger, player.Name, msg)
				return
			}
		}

		// Strategy 2: Build additional refinery if Oil is cheap and Fuel is expensive
		fuelPrice := market.GetBuyPrice("Fuel")
		oilPrice := market.GetBuyPrice("Oil")
		refineryCount := countBuildings(planet, "Refinery")
		if fuelPrice > 150 && oilPrice < 200 && refineryCount < 3 && player.Credits >= 1500 {
			player.Credits -= 1500
			builder.AIBuildOnPlanet(planet, "Refinery", player.Name, systemID)
			fmt.Printf("[AIBuild] %s built refinery #%d at %s (fuel@%.0f, oil@%.0f)\n",
				player.Name, refineryCount+1, planet.Name, fuelPrice, oilPrice)
			return
		}

		// Strategy 3: Build habitat when population is near capacity and water is available
		capacity := planet.GetTotalPopulationCapacity()
		if capacity > 0 && planet.Population > int64(float64(capacity)*0.7) && player.Credits >= 800 {
			if planet.GetStoredAmount("Water") > 50 {
				player.Credits -= 800
				builder.AIBuildOnPlanet(planet, "Habitat", player.Name, systemID)
				fmt.Printf("[AIBuild] %s built habitat at %s (pop %d/%d)\n",
					player.Name, planet.Name, planet.Population, capacity)
				return
			}
		}

		// Strategy 4: Build Trading Post if we don't have one (unlikely but handle it)
		if !hasBuilding(planet, "Trading Post") && player.Credits >= 1200 {
			player.Credits -= 1200
			builder.AIBuildOnPlanet(planet, "Trading Post", player.Name, systemID)
			fmt.Printf("[AIBuild] %s built Trading Post at %s\n", player.Name, planet.Name)
			return
		}

		// Strategy 5: Build Shipyard if we don't have one and can afford it
		if !hasBuilding(planet, "Shipyard") && player.Credits >= 3000 {
			player.Credits -= 2000
			builder.AIBuildOnPlanet(planet, "Shipyard", player.Name, systemID)
			msg := fmt.Sprintf("%s built Shipyard at %s", player.Name, planet.Name)
			fmt.Printf("[AIBuild] %s\n", msg)
			logBuildEvent(logger, player.Name, msg)
			return
		}

		// Strategy 6: Build Colony ship and expand if we have a Shipyard,
		// only 1 planet, enough resources, and enough credits
		if len(player.OwnedPlanets) < 3 && hasBuilding(planet, "Shipyard") && player.Credits >= 4000 {
			// Check if we already have a colony ship
			hasColony := false
			for _, ship := range player.OwnedShips {
				if ship != nil && ship.ShipType == entities.ShipTypeColony {
					hasColony = true
					break
				}
			}
			if !hasColony && planet.GetStoredAmount("Iron") >= 100 &&
				planet.GetStoredAmount("Fuel") >= 80 &&
				planet.GetStoredAmount("Rare Metals") >= 20 {
				// Deduct resources and credits (matches entities.GetShipResourceRequirements)
				player.Credits -= 2000
				planet.RemoveStoredResource("Iron", 100)
				planet.RemoveStoredResource("Fuel", 80)
				planet.RemoveStoredResource("Rare Metals", 20)
				// Queue construction
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
						msg := fmt.Sprintf("%s building Colony ship for expansion", player.Name)
						fmt.Printf("[AIBuild] %s at %s\n", msg, planet.Name)
						logBuildEvent(logger, player.Name, msg)
					}
				}
				return
			}
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
	}
	return 100
}
