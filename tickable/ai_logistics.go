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

// CargoOperator defines cargo load/unload operations (avoids importing game package).
type CargoOperator interface {
	LoadCargo(ship *entities.Ship, planet *entities.Planet, resource string, qty int) (int, error)
	UnloadCargo(ship *entities.Ship, planet *entities.Planet, resource string, qty int) (int, error)
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

	gameObj := ctx.GetGame()
	if gameObj == nil {
		return
	}

	// Get cargo operator via the CargoOperator interface directly on the game object
	cargoOp, ok := gameObj.(CargoOperator)
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

	// Get systems for planet lookup
	sp, ok := gameObj.(SystemsProvider)
	if !ok {
		return
	}
	systems := sp.GetSystems()

	for _, player := range players {
		if player == nil || player.IsHuman() {
			continue
		}
		als.processAILogistics(player, cargoOp, systems)
	}
}

// FleetMover lets AI move ships without importing game package.
type FleetMover interface {
	GetConnectedSystems(fromSystemID int) []int
}

// ShipJourney starts a ship moving via ShipMovementHelper.
type ShipJourney interface {
	StartShipJourney(ship *entities.Ship, targetSystemID int) bool
}

func (als *AILogisticsSystem) processAILogistics(player *entities.Player, cargoOp CargoOperator, systems []*entities.System) {
	gameObj := als.GetContext().GetGame()
	mover, hasMover := gameObj.(FleetMover)
	journeyer, hasJourney := gameObj.(ShipJourney)

	for _, ship := range player.OwnedShips {
		if ship == nil || ship.ShipType != entities.ShipTypeCargo {
			continue
		}

		// Skip ships that are currently moving
		if ship.Status == entities.ShipStatusMoving {
			continue
		}

		// Find the planet this ship is orbiting
		planet := findPlanetAtShipOrbit(ship, systems)
		isHome := planet != nil && planet.Owner == ship.Owner

		if isHome {
			if ship.GetTotalCargo() > 0 {
				// Returned home with cargo — unload
				als.unloadAllCargo(ship, planet, cargoOp)
			} else {
				// Empty at home — load surplus and send to another system
				als.loadSurplus(ship, planet, cargoOp)
				if ship.GetTotalCargo() > 0 && hasMover && hasJourney {
					// Pick a connected system and go there
					connected := mover.GetConnectedSystems(ship.CurrentSystem)
					if len(connected) > 0 {
						// Pick the first connected system
						target := connected[0]
						if journeyer.StartShipJourney(ship, target) {
							fmt.Printf("[AILogistics] %s dispatched %s to SYS-%d with cargo\n",
								player.Name, ship.Name, target)
						}
					}
				}
			}
		} else {
			// At a foreign system or not at a planet — head home
			if hasJourney {
				homeSys := als.findHomeSystem(player, systems)
				if homeSys >= 0 && homeSys != ship.CurrentSystem {
					if journeyer.StartShipJourney(ship, homeSys) {
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

func (als *AILogisticsSystem) unloadAllCargo(ship *entities.Ship, planet *entities.Planet, cargoOp CargoOperator) {
	for resType, amount := range ship.CargoHold {
		if amount <= 0 {
			continue
		}
		unloaded, err := cargoOp.UnloadCargo(ship, planet, resType, amount)
		if err == nil && unloaded > 0 {
			fmt.Printf("[AILogistics] %s unloaded %d %s at %s\n", ship.Name, unloaded, resType, planet.Name)
		}
	}
}

func (als *AILogisticsSystem) loadSurplus(ship *entities.Ship, planet *entities.Planet, cargoOp CargoOperator) {
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
	loaded, err := cargoOp.LoadCargo(ship, planet, bestRes, qty)
	if err == nil && loaded > 0 {
		fmt.Printf("[AILogistics] %s loaded %d %s from %s\n", ship.Name, loaded, bestRes, planet.Name)
	}
}

// findPlanetAtShipOrbit finds the planet a ship is orbiting in its current system.
func findPlanetAtShipOrbit(ship *entities.Ship, systems []*entities.System) *entities.Planet {
	for _, system := range systems {
		if system.ID != ship.CurrentSystem {
			continue
		}
		for _, entity := range system.Entities {
			if planet, ok := entity.(*entities.Planet); ok {
				if math.Abs(ship.GetOrbitDistance()-planet.GetOrbitDistance()) < 1.0 {
					return planet
				}
			}
		}
		break
	}
	return nil
}
