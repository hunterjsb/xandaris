package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ContractExecutionSystem{
		BaseSystem: NewBaseSystem("ContractExecution", 28),
	})
}

// ContractExecutionSystem processes trade contracts — when a contract's
// timer fires, resources transfer from supplier to buyer and credits
// transfer from buyer to supplier. Both must be in the same system.
type ContractExecutionSystem struct {
	*BaseSystem
}

func (ces *ContractExecutionSystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	ctx := ces.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	cm := game.GetContractManager()
	if cm == nil {
		return
	}

	players := ctx.GetPlayers()
	systemsMap := game.GetSystemsMap()

	playerByName := make(map[string]*entities.Player)
	for _, p := range players {
		if p != nil {
			playerByName[p.Name] = p
		}
	}

	// Get contracts ready for execution
	ready := cm.TickContracts()
	for _, contract := range ready {
		supplier := playerByName[contract.Supplier]
		buyer := playerByName[contract.Buyer]
		if supplier == nil || buyer == nil {
			continue
		}

		// Find supplier's planet in the contract system with the resource
		var supplierPlanet *entities.Planet
		for _, sys := range game.GetSystems() {
			if sys.ID != contract.SystemID {
				continue
			}
			for _, e := range sys.Entities {
				if p, ok := e.(*entities.Planet); ok && p.Owner == contract.Supplier {
					if p.GetStoredAmount(contract.Resource) >= contract.Quantity {
						supplierPlanet = p
						break
					}
				}
			}
			break
		}

		buyerPlanet := findPlanetByID(systemsMap, contract.PlanetID)
		if supplierPlanet == nil || buyerPlanet == nil {
			continue
		}

		total := contract.PricePerUnit * contract.Quantity
		if buyer.Credits < total {
			continue // buyer can't afford
		}

		// Execute
		supplierPlanet.RemoveStoredResource(contract.Resource, contract.Quantity)
		buyerPlanet.AddStoredResource(contract.Resource, contract.Quantity)
		buyer.Credits -= total
		supplier.Credits += total
		cm.CompleteDelivery(contract.ID)

		game.LogEvent("trade", contract.Supplier,
			fmt.Sprintf("%s delivered %d %s to %s for %dcr (contract #%d)",
				contract.Supplier, contract.Quantity, contract.Resource,
				contract.Buyer, total, contract.ID))
	}
}
