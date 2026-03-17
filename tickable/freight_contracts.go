package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FreightContractSystem{
		BaseSystem: NewBaseSystem("FreightContracts", 66),
	})
}

// FreightContractSystem generates NPC delivery contracts that pay
// factions to haul specific resources between systems. Unlike player-
// created shipping routes, these are time-limited jobs from "galactic
// commerce guilds" that any faction can claim.
//
// Contract structure:
//   - Pickup: specific resource at a specific system
//   - Delivery: different system
//   - Quantity: 50-300 units
//   - Deadline: 5000-10000 ticks
//   - Reward: 2-5x market value (premium for reliability)
//   - Penalty: lose deposit if deadline missed
//
// Contracts incentivize building efficient logistics networks.
// Completing contracts also boosts trade reputation.
//
// Multiple factions can see the same contract, but only one can claim it.
// First to deliver wins the reward.
type FreightContractSystem struct {
	*BaseSystem
	contracts    []*FreightContract
	nextContract int64
}

// FreightContract represents an NPC delivery job.
type FreightContract struct {
	ID          int
	PickupSys   int
	PickupName  string
	DeliverSys  int
	DeliverName string
	Resource    string
	Quantity    int
	Reward      int
	Deposit     int // paid on claim, returned + reward on delivery
	Deadline    int64
	ClaimedBy   string
	Completed   bool
	Expired     bool
}

func (fcs *FreightContractSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := fcs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if fcs.nextContract == 0 {
		fcs.nextContract = tick + 2000 + int64(rand.Intn(3000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Check deadlines
	for _, contract := range fcs.contracts {
		if contract.Completed || contract.Expired {
			continue
		}
		if tick > contract.Deadline {
			contract.Expired = true
			if contract.ClaimedBy != "" {
				// Forfeit deposit
				game.LogEvent("logistics", contract.ClaimedBy,
					fmt.Sprintf("❌ Freight contract expired! Failed to deliver %d %s to %s. Deposit lost!",
						contract.Quantity, contract.Resource, contract.DeliverName))
			}
		}
	}

	// Check for deliveries (cargo ships at delivery system with right cargo)
	for _, contract := range fcs.contracts {
		if contract.Completed || contract.Expired || contract.ClaimedBy == "" {
			continue
		}
		fcs.checkDelivery(contract, players, systems, game)
	}

	// Auto-claim: any faction with a cargo ship at the pickup system can claim
	for _, contract := range fcs.contracts {
		if contract.Completed || contract.Expired || contract.ClaimedBy != "" {
			continue
		}
		fcs.checkClaim(tick, contract, players, systems, game)
	}

	// Generate new contracts
	if tick >= fcs.nextContract {
		fcs.nextContract = tick + 5000 + int64(rand.Intn(8000))

		activeCount := 0
		for _, c := range fcs.contracts {
			if !c.Completed && !c.Expired {
				activeCount++
			}
		}
		if activeCount < 5 {
			fcs.generateContract(tick, game, systems)
		}
	}
}

func (fcs *FreightContractSystem) checkClaim(tick int64, contract *FreightContract, players []*entities.Player, systems []*entities.System, game GameProvider) {
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo || ship.Status == entities.ShipStatusMoving {
				continue
			}
			if ship.CurrentSystem != contract.PickupSys {
				continue
			}
			// Auto-claim if player has enough credits for deposit
			if player.Credits < contract.Deposit {
				continue
			}

			contract.ClaimedBy = player.Name
			player.Credits -= contract.Deposit
			contract.Deadline = tick + 8000 + int64(rand.Intn(5000))

			game.LogEvent("logistics", player.Name,
				fmt.Sprintf("📋 %s claimed freight contract: deliver %d %s from %s to %s. Reward: %dcr (deposit: %dcr)",
					player.Name, contract.Quantity, contract.Resource,
					contract.PickupName, contract.DeliverName, contract.Reward, contract.Deposit))
			return
		}
	}
}

func (fcs *FreightContractSystem) checkDelivery(contract *FreightContract, players []*entities.Player, systems []*entities.System, game GameProvider) {
	for _, player := range players {
		if player == nil || player.Name != contract.ClaimedBy {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo {
				continue
			}
			if ship.CurrentSystem != contract.DeliverSys {
				continue
			}
			// Check if ship has enough of the resource
			if ship.CargoHold[contract.Resource] >= contract.Quantity {
				// Deliver!
				ship.RemoveCargo(contract.Resource, contract.Quantity)
				contract.Completed = true
				// Return deposit + reward
				player.Credits += contract.Deposit + contract.Reward
				game.LogEvent("logistics", player.Name,
					fmt.Sprintf("✅ Freight contract completed! %s delivered %d %s to %s. Earned %dcr (+%dcr deposit returned)",
						player.Name, contract.Quantity, contract.Resource,
						contract.DeliverName, contract.Reward, contract.Deposit))
				return
			}
		}
	}
}

func (fcs *FreightContractSystem) generateContract(tick int64, game GameProvider, systems []*entities.System) {
	if len(systems) < 5 {
		return
	}

	// Pick two different systems
	a := rand.Intn(len(systems))
	b := rand.Intn(len(systems))
	for b == a {
		b = rand.Intn(len(systems))
	}

	resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
		entities.ResFuel, entities.ResRareMetals}
	res := resources[rand.Intn(len(resources))]

	qty := 50 + rand.Intn(250)
	market := game.GetMarketEngine()
	baseValue := 0
	if market != nil {
		baseValue = int(market.GetSellPrice(res)) * qty
	}
	if baseValue < 500 {
		baseValue = 500
	}
	reward := baseValue*2 + rand.Intn(baseValue)
	deposit := reward / 4

	contract := &FreightContract{
		ID:          len(fcs.contracts) + 1,
		PickupSys:   systems[a].ID,
		PickupName:  systems[a].Name,
		DeliverSys:  systems[b].ID,
		DeliverName: systems[b].Name,
		Resource:    res,
		Quantity:    qty,
		Reward:      reward,
		Deposit:     deposit,
		Deadline:    tick + 15000, // initial deadline before claim
	}
	fcs.contracts = append(fcs.contracts, contract)

	game.LogEvent("logistics", "",
		fmt.Sprintf("📋 FREIGHT CONTRACT: Deliver %d %s from %s to %s. Reward: %dcr. Send a cargo ship to %s to claim!",
			qty, res, systems[a].Name, systems[b].Name, reward, systems[a].Name))
}

// GetActiveContracts returns active freight contracts.
func (fcs *FreightContractSystem) GetActiveContracts() []*FreightContract {
	var result []*FreightContract
	for _, c := range fcs.contracts {
		if !c.Completed && !c.Expired {
			result = append(result, c)
		}
	}
	return result
}
