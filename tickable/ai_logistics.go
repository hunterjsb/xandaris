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

func (als *AILogisticsSystem) processAILogistics(player *entities.Player, cargoOp CargoOperator, systems []*entities.System) {
	for _, ship := range player.OwnedShips {
		if ship == nil || ship.ShipType != entities.ShipTypeCargo {
			continue
		}

		// Skip ships that are currently moving
		if ship.Status == entities.ShipStatusMoving {
			continue
		}

		// Find the planet this ship is orbiting (if any)
		planet := findPlanetAtShipOrbit(ship, systems)

		if planet != nil && planet.Owner == ship.Owner {
			// Ship is at an owned planet
			if ship.GetTotalCargo() > 0 {
				// Has cargo — unload everything
				als.unloadAllCargo(ship, planet, cargoOp)
			} else {
				// Empty — load surplus resources
				als.loadSurplus(ship, planet, cargoOp)
			}
		}
	}
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
	for resType, storage := range planet.StoredResources {
		if storage == nil || storage.Capacity <= 0 {
			continue
		}
		ratio := float64(storage.Amount) / float64(storage.Capacity)
		if ratio > 0.60 {
			excess := storage.Amount - int(float64(storage.Capacity)*0.50)
			if excess <= 0 {
				continue
			}
			loaded, err := cargoOp.LoadCargo(ship, planet, resType, excess)
			if err == nil && loaded > 0 {
				fmt.Printf("[AILogistics] %s loaded %d %s from %s\n", ship.Name, loaded, resType, planet.Name)
			}
		}
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
