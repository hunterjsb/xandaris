package tickable

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&MercenarySystem{
		BaseSystem: NewBaseSystem("Mercenaries", 43),
	})
}

// MercenarySystem lets players hire temporary combat ships.
// Mercenaries cost credits, last for a limited time, and fight
// pirates or hostile factions in the system they're hired to.
//
// Hire types:
//   Escort (2000cr, 3000 ticks): 1 Frigate guards your cargo ships
//   Strike Force (5000cr, 3000 ticks): 2 Frigates clear pirates
//   Armada (15000cr, 5000 ticks): 1 Destroyer + 2 Frigates
type MercenarySystem struct {
	*BaseSystem
	contracts []mercContract
}

type mercContract struct {
	owner     string
	systemID  int
	ships     []*entities.Ship
	ticksLeft int
}

func (ms *MercenarySystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	// Decay active contracts
	for i := len(ms.contracts) - 1; i >= 0; i-- {
		c := &ms.contracts[i]
		c.ticksLeft -= 100
		if c.ticksLeft <= 0 {
			// Remove mercenary ships
			for _, ship := range c.ships {
				ship.CurrentHealth = 0 // mark for removal
			}
			ms.contracts = append(ms.contracts[:i], ms.contracts[i+1:]...)
		}
	}
}

// HireMercenaries creates temporary combat ships for a player.
func (ms *MercenarySystem) HireMercenaries(player *entities.Player, mercType string, systemID int, sys *entities.System) (int, error) {
	if ms.contracts == nil {
		ms.contracts = make([]mercContract, 0)
	}

	var ships []*entities.Ship
	var cost int
	duration := 3000

	switch mercType {
	case "escort":
		cost = 2000
		ships = append(ships, createMercShip(entities.ShipTypeFrigate, player.Name, systemID))
	case "strike_force":
		cost = 5000
		ships = append(ships,
			createMercShip(entities.ShipTypeFrigate, player.Name, systemID),
			createMercShip(entities.ShipTypeFrigate, player.Name, systemID))
	case "armada":
		cost = 15000
		duration = 5000
		ships = append(ships,
			createMercShip(entities.ShipTypeDestroyer, player.Name, systemID),
			createMercShip(entities.ShipTypeFrigate, player.Name, systemID),
			createMercShip(entities.ShipTypeFrigate, player.Name, systemID))
	default:
		return 0, fmt.Errorf("unknown merc type (use escort, strike_force, or armada)")
	}

	if player.Credits < cost {
		return 0, fmt.Errorf("need %d credits (have %d)", cost, player.Credits)
	}

	player.Credits -= cost

	// Add ships to player's fleet and system
	for _, ship := range ships {
		player.OwnedShips = append(player.OwnedShips, ship)
		if sys != nil {
			sys.Entities = append(sys.Entities, ship)
		}
	}

	ms.contracts = append(ms.contracts, mercContract{
		owner:     player.Name,
		systemID:  systemID,
		ships:     ships,
		ticksLeft: duration,
	})

	return cost, nil
}

func createMercShip(shipType entities.ShipType, owner string, systemID int) *entities.Ship {
	id := rand.Intn(900000000) + 100000000
	name := fmt.Sprintf("Merc %s %d", shipType, id%1000)
	ship := entities.NewShip(id, name, shipType, systemID, owner,
		color.RGBA{255, 100, 100, 255}) // red for mercenaries
	ship.Status = entities.ShipStatusOrbiting
	return ship
}
