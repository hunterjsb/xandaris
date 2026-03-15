package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&DeliverySystem{
		BaseSystem: NewBaseSystem("Delivery", 28),
	})
}

// DeliverySystem processes pending trade deliveries — advances multi-hop
// routes and unloads cargo when ships arrive at their destination.
type DeliverySystem struct {
	*BaseSystem
}

func (ds *DeliverySystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	context := ds.GetContext()
	if context == nil {
		return
	}

	game := context.GetGame()
	if game == nil {
		return
	}

	dm := game.GetDeliveryManager()
	if dm == nil {
		return
	}

	players := game.GetPlayers()
	systemsMap := game.GetSystemsMap()

	for _, delivery := range dm.GetActiveDeliveries() {
		ship := findShipByID(players, delivery.ShipID)
		if ship == nil {
			// Ship lost — refund buyer
			buyerName, refund := dm.FailDelivery(delivery.ID)
			refundPlayer(players, buyerName, refund)
			continue
		}

		// Ship still moving — wait
		if ship.Status == entities.ShipStatusMoving {
			continue
		}

		// Ship arrived at current hop — advance route
		if len(ship.RoutePath) > 0 {
			// Check if we're at the next hop
			if ship.CurrentSystem == ship.RoutePath[0] {
				ship.RoutePath = ship.RoutePath[1:]
			}

			// More hops to go — dispatch to next
			if len(ship.RoutePath) > 0 {
				game.StartShipJourney(ship, ship.RoutePath[0])
				continue
			}
		}

		// Route complete — check if we're at the destination
		if ship.CurrentSystem == delivery.DestSystemID {
			// Find destination planet and unload
			destPlanet := findPlanetByID(systemsMap, delivery.DestPlanetID)
			if destPlanet != nil {
				// Unload cargo
				qty := ship.CargoHold[delivery.Resource]
				if qty > 0 {
					if qty > delivery.Quantity {
						qty = delivery.Quantity
					}
					ship.CargoHold[delivery.Resource] -= qty
					if ship.CargoHold[delivery.Resource] <= 0 {
						delete(ship.CargoHold, delivery.Resource)
					}
					destPlanet.AddStoredResource(delivery.Resource, qty)
				}
			}

			dm.CompleteDelivery(delivery.ID)
			ship.DeliveryID = 0
			ship.RoutePath = nil

			fmt.Printf("[Delivery] Ship %s delivered %d %s to %s\n",
				ship.Name, delivery.Quantity, delivery.Resource, delivery.BuyerName)
		}
	}
}

func findShipByID(players []*entities.Player, shipID int) *entities.Ship {
	for _, p := range players {
		if p == nil {
			continue
		}
		for _, ship := range p.OwnedShips {
			if ship != nil && ship.GetID() == shipID {
				return ship
			}
		}
	}
	return nil
}

func findPlanetByID(systemsMap map[int]*entities.System, planetID int) *entities.Planet {
	for _, sys := range systemsMap {
		for _, e := range sys.Entities {
			if p, ok := e.(*entities.Planet); ok && p.GetID() == planetID {
				return p
			}
		}
	}
	return nil
}

func refundPlayer(players []*entities.Player, name string, amount int) {
	for _, p := range players {
		if p != nil && p.Name == name {
			p.Credits += amount
			fmt.Printf("[Delivery] Refunded %d credits to %s (shipment lost)\n", amount, name)
			return
		}
	}
}
