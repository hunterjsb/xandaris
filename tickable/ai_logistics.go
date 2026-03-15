package tickable

import (
	"fmt"
	"math"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AILogisticsSystem{
		BaseSystem: NewBaseSystem("AILogistics", 30),
	})
}

// AILogisticsSystem manages AI cargo ships: loading, delivering, and unloading goods.
type AILogisticsSystem struct {
	*BaseSystem
}

func (als *AILogisticsSystem) OnTick(tick int64) {
	// Run every 50 ticks
	if tick%50 != 0 {
		return
	}

	ctx := als.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil || player.IsHuman() {
			continue
		}
		als.processAILogistics(player, game, systems)
	}
}

func (als *AILogisticsSystem) processAILogistics(player *entities.Player, game GameProvider, systems []*entities.System) {
	als.processColonyShips(player, game, systems)
	als.processCargoShips(player, game, systems)
}

// processColonyShips handles AI colony ship navigation and colonization.
func (als *AILogisticsSystem) processColonyShips(player *entities.Player, game GameProvider, systems []*entities.System) {
	for _, ship := range player.OwnedShips {
		if ship == nil || ship.ShipType != entities.ShipTypeColony || ship.Colonists <= 0 {
			continue
		}
		if ship.Status == entities.ShipStatusMoving {
			continue
		}
		// Check if there's an unclaimed habitable planet in this system
		planet := findUnclaimedHabitable(ship.CurrentSystem, systems)
		if planet != nil {
			planet.Owner = ship.Owner
			planet.Population = int64(ship.Colonists)
			planet.SetBaseOwner(ship.Owner)
			player.AddOwnedPlanet(planet)
			for _, resEntity := range planet.Resources {
				if res, ok := resEntity.(*entities.Resource); ok {
					res.Owner = ship.Owner
				}
			}

			// Set up colony infrastructure via the game server
			systemID := ship.CurrentSystem
			game.AIBuildOnPlanet(planet, entities.BuildingTradingPost, ship.Owner, systemID)
			game.AIBuildOnPlanet(planet, entities.BuildingRefinery, ship.Owner, systemID)
			game.AIBuildOnPlanet(planet, entities.BuildingGenerator, ship.Owner, systemID)

			// Seed starting resources
			planet.AddStoredResource(entities.ResFuel, 100)
			planet.AddStoredResource(entities.ResWater, 100)

			ship.Colonists = 0
			ship.Status = entities.ShipStatusOrbiting
			planet.RebalanceWorkforce()
			msg := fmt.Sprintf("%s colonized %s with infrastructure!", player.Name, planet.Name)
			fmt.Printf("[AIColonize] %s (%d colonists)\n", msg, planet.Population)
			game.LogEvent("colonize", player.Name, msg)
			continue
		}
		// Find nearest unclaimed habitable planet and fly there (if enough fuel)
		fuelNeeded := ship.FuelPerJump + int(ship.FuelPerTick*120)
		if ship.CurrentFuel >= fuelNeeded {
			targetSys := als.findColonyTarget(ship, player, systems, game)
			if targetSys >= 0 && targetSys != ship.CurrentSystem {
				if game.StartShipJourney(ship, targetSys) {
					fmt.Printf("[AIColonize] %s sending colony ship to SYS-%d\n", player.Name, targetSys)
				}
			}
		}
	}
}

// processCargoShips handles AI cargo ship trade routes: load surplus, deliver, return.
func (als *AILogisticsSystem) processCargoShips(player *entities.Player, game GameProvider, systems []*entities.System) {
	for _, ship := range player.OwnedShips {
		if ship == nil || ship.ShipType != entities.ShipTypeCargo {
			continue
		}

		// Skip ships that are currently moving or on a delivery mission
		if ship.Status == entities.ShipStatusMoving {
			continue
		}
		if ship.DeliveryID != 0 {
			continue
		}

		// Skip ships without enough fuel for a round trip
		// Each jump costs FuelPerJump + ~100 ticks of FuelPerTick
		fuelPerTrip := ship.FuelPerJump + int(ship.FuelPerTick*120)
		if ship.CurrentFuel < fuelPerTrip*2 {
			continue
		}

		// Find the planet this ship is orbiting
		planet := findPlanetAtShipOrbit(ship, systems)
		isHome := planet != nil && planet.Owner == ship.Owner

		if isHome {
			if ship.GetTotalCargo() > 0 {
				// Returned home with cargo — unload
				als.unloadAllCargo(ship, planet, game)
			} else {
				// Empty at home — load surplus and send to another system
				// Only dispatch if enough fuel for round trip
				fuelNeeded := (ship.FuelPerJump + int(ship.FuelPerTick*120)) * 2
				if ship.CurrentFuel < fuelNeeded {
					continue // wait for refueling
				}
				als.loadSurplus(ship, planet, game)
				if ship.GetTotalCargo() > 0 {
					// Pick the best connected system based on demand
					connected := game.GetConnectedSystems(ship.CurrentSystem)
					target := als.pickBestTradeTarget(connected, ship, systems)
					if target >= 0 {
						if game.StartShipJourney(ship, target) {
							fmt.Printf("[AILogistics] %s dispatched %s to SYS-%d with cargo\n",
								player.Name, ship.Name, target)
						}
					}
				}
			}
		} else {
			// At a foreign system — try to sell cargo at a Trading Post, then head home
			if ship.GetTotalCargo() > 0 {
				// Find a planet with a Trading Post in this system to unload at
				for _, sys := range systems {
					if sys.ID != ship.CurrentSystem {
						continue
					}
					for _, e := range sys.Entities {
						if p, ok := e.(*entities.Planet); ok && p.Owner != "" {
							// Try unloading at any planet (Trading Post check is in UnloadCargo)
							als.unloadAllCargo(ship, p, game)
							if ship.GetTotalCargo() == 0 {
								break
							}
						}
					}
					break
				}
			}
			// Head home if enough fuel
			fuelForReturn := ship.FuelPerJump + int(ship.FuelPerTick*120)
			if ship.CurrentFuel >= fuelForReturn {
				homeSys := als.findHomeSystem(player, systems)
				if homeSys >= 0 && homeSys != ship.CurrentSystem {
					if game.StartShipJourney(ship, homeSys) {
						fmt.Printf("[AILogistics] %s returning %s home to SYS-%d\n",
							player.Name, ship.Name, homeSys)
					}
				}
			}
		}
	}
}

func (als *AILogisticsSystem) findHomeSystem(player *entities.Player, systems []*entities.System) int {
	if player.HomeSystem != nil {
		return player.HomeSystem.ID
	}
	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if p, ok := e.(*entities.Planet); ok && p.GetID() == planet.GetID() {
					return sys.ID
				}
			}
		}
	}
	return -1
}

func (als *AILogisticsSystem) unloadAllCargo(ship *entities.Ship, planet *entities.Planet, game GameProvider) {
	for resType, amount := range ship.CargoHold {
		if amount <= 0 {
			continue
		}
		unloaded, err := game.UnloadCargo(ship, planet, resType, amount)
		if err == nil && unloaded > 0 {
			fmt.Printf("[AILogistics] %s unloaded %d %s at %s\n", ship.Name, unloaded, resType, planet.Name)
		}
	}
}

func (als *AILogisticsSystem) loadSurplus(ship *entities.Ship, planet *entities.Planet, game GameProvider) {
	// Find the resource with highest stock ratio and load some of it
	var bestRes string
	bestRatio := 0.0
	for resType, storage := range planet.StoredResources {
		if storage == nil || storage.Capacity <= 0 || storage.Amount < 30 {
			continue
		}
		ratio := float64(storage.Amount) / float64(storage.Capacity)
		if ratio > bestRatio {
			bestRatio = ratio
			bestRes = resType
		}
	}
	if bestRes == "" || bestRatio < 0.20 {
		return // nothing worth transporting
	}
	storage := planet.StoredResources[bestRes]
	// Load up to 100 units, keeping at least 20% on planet
	qty := storage.Amount - int(float64(storage.Capacity)*0.20)
	if qty > 100 {
		qty = 100
	}
	if qty <= 0 {
		return
	}
	loaded, err := game.LoadCargo(ship, planet, bestRes, qty)
	if err == nil && loaded > 0 {
		fmt.Printf("[AILogistics] %s loaded %d %s from %s\n", ship.Name, loaded, bestRes, planet.Name)
	}
}

// pickBestTradeTarget picks the connected system with the highest demand (lowest stock).
// Prefers systems with owned planets (can unload) or planets with Trading Posts.
func (als *AILogisticsSystem) pickBestTradeTarget(connected []int, ship *entities.Ship, systems []*entities.System) int {
	if len(connected) == 0 {
		return -1
	}
	bestSys := connected[0]
	bestScore := -1

	for _, sysID := range connected {
		for _, sys := range systems {
			if sys.ID != sysID {
				continue
			}
			score := 0
			for _, e := range sys.Entities {
				p, ok := e.(*entities.Planet)
				if !ok {
					continue
				}
				// Prefer systems with owned planets (easy unload + refuel)
				if p.Owner == ship.Owner {
					score += 100
				}
				// Prefer systems with Trading Posts (can trade there)
				for _, be := range p.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
						score += 50
						break
					}
				}
				// Prefer systems with low stock (high demand)
				totalStock := 0
				for _, s := range p.StoredResources {
					if s != nil {
						totalStock += s.Amount
					}
				}
				if totalStock < 500 {
					score += 30 // low stock = needs goods
				}
				// Prefer populated planets (consumers)
				if p.Population > 1000 {
					score += 20
				}
			}
			if score > bestScore {
				bestScore = score
				bestSys = sysID
			}
		}
	}
	return bestSys
}

// findUnclaimedHabitable finds any unclaimed habitable planet in a system.
func findUnclaimedHabitable(systemID int, systems []*entities.System) *entities.Planet {
	for _, sys := range systems {
		if sys.ID != systemID {
			continue
		}
		for _, e := range sys.Entities {
			if p, ok := e.(*entities.Planet); ok && p.Owner == "" && p.IsHabitable() {
				return p
			}
		}
		break
	}
	return nil
}

// findColonyTarget finds the nearest system with an unclaimed habitable planet.
func (als *AILogisticsSystem) findColonyTarget(ship *entities.Ship, player *entities.Player, systems []*entities.System, game GameProvider) int {
	// BFS from current system to find nearest unclaimed habitable planet
	visited := map[int]bool{ship.CurrentSystem: true}
	queue := game.GetConnectedSystems(ship.CurrentSystem)
	for _, id := range queue {
		visited[id] = true
	}
	for len(queue) > 0 {
		sysID := queue[0]
		queue = queue[1:]
		// Check if this system has unclaimed habitable planets
		for _, sys := range systems {
			if sys.ID != sysID {
				continue
			}
			for _, e := range sys.Entities {
				if p, ok := e.(*entities.Planet); ok && p.Owner == "" && p.IsHabitable() {
					return sysID
				}
			}
		}
		// Expand search
		for _, next := range game.GetConnectedSystems(sysID) {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
		if len(visited) > 15 {
			break // Don't search too far
		}
	}
	return -1
}

// findPlanetAtShipOrbit finds the planet a ship is orbiting in its current system.
func findPlanetAtShipOrbit(ship *entities.Ship, systems []*entities.System) *entities.Planet {
	for _, system := range systems {
		if system.ID != ship.CurrentSystem {
			continue
		}
		for _, entity := range system.Entities {
			if planet, ok := entity.(*entities.Planet); ok {
				if math.Abs(ship.GetOrbitDistance()-planet.GetOrbitDistance()) < 5.0 {
					return planet
				}
			}
		}
		break
	}
	return nil
}
