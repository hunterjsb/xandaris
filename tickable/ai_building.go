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
		abs.evaluateInvestment(player, market, builder, systems)
	}
}

func (abs *AIBuildingSystem) evaluateInvestment(player *entities.Player, market interface{ GetBuyPrice(string) float64 }, builder BuildingAdder, systems []*entities.System) {
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
			if buyPrice > basePrice*2.0 && mineCount < 3 && player.Credits >= 500 {
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
				fmt.Printf("[AIBuild] %s built extra mine on %s at %s (price %.0f, base %.0f)\n",
					player.Name, res.ResourceType, planet.Name, buyPrice, basePrice)
				return
			}

			// Upgrade existing mine if resource is expensive (price > 1.5x base)
			if buyPrice > basePrice*1.5 && bestMine != nil && bestMine.CanUpgrade() && player.Credits >= 300 {
				player.Credits -= 300
				bestMine.Upgrade()
				fmt.Printf("[AIBuild] %s upgraded %s mine to L%d at %s\n",
					player.Name, res.ResourceType, bestMine.Level, planet.Name)
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
