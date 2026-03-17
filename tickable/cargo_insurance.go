package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&CargoInsuranceSystem{
		BaseSystem: NewBaseSystem("CargoInsurance", 51),
	})
}

// CargoInsuranceSystem provides cargo protection for trading factions.
// Insurance is automatic — factions with Trading Post level 3+ get coverage.
//
// How it works:
//   - Every 2000 ticks, evaluate each faction's cargo losses
//   - If a faction has a TP L3+, they pay a premium (1% of credits per interval)
//   - In exchange, 50% of pirate/blockade cargo losses are refunded as credits
//   - TP L4+ gets 75% refund, TP L5 gets 100% refund
//   - This makes higher-level Trading Posts genuinely valuable for logistics
//
// The premium creates a steady credit drain that scales with wealth,
// while the refund makes risky trade routes viable.
type CargoInsuranceSystem struct {
	*BaseSystem
	losses      map[string]int   // playerName → cargo value lost since last payout
	premiumPaid map[string]int   // playerName → total premiums paid
	coverage    map[string]float64 // playerName → coverage percentage
	nextPayout  int64
}

func (cis *CargoInsuranceSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := cis.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if cis.losses == nil {
		cis.losses = make(map[string]int)
		cis.premiumPaid = make(map[string]int)
		cis.coverage = make(map[string]float64)
	}

	if cis.nextPayout == 0 {
		cis.nextPayout = tick + 2000
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Calculate coverage level per faction from best Trading Post
	for _, player := range players {
		if player == nil {
			continue
		}

		bestTP := 0
		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner != player.Name {
					continue
				}
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
						if b.Level > bestTP {
							bestTP = b.Level
						}
					}
				}
			}
		}

		switch {
		case bestTP >= 5:
			cis.coverage[player.Name] = 1.0
		case bestTP >= 4:
			cis.coverage[player.Name] = 0.75
		case bestTP >= 3:
			cis.coverage[player.Name] = 0.50
		default:
			cis.coverage[player.Name] = 0
		}
	}

	// Collect premiums and process payouts
	if tick >= cis.nextPayout {
		cis.nextPayout = tick + 2000

		for _, player := range players {
			if player == nil {
				continue
			}

			cov := cis.coverage[player.Name]
			if cov <= 0 {
				continue
			}

			// Premium: 0.5% of credits
			premium := player.Credits / 200
			if premium > 0 {
				player.Credits -= premium
				cis.premiumPaid[player.Name] += premium
			}

			// Payout for losses
			loss := cis.losses[player.Name]
			if loss > 0 {
				refund := int(float64(loss) * cov)
				player.Credits += refund
				cis.losses[player.Name] = 0

				game.LogEvent("logistics", player.Name,
					fmt.Sprintf("🛡️ Cargo insurance payout: %d credits (%.0f%% coverage, premium: %d cr)",
						refund, cov*100, premium))
			}
		}
	}
}

// RecordCargoLoss records a cargo loss for insurance purposes.
// Called by pirate raids, blockade interceptions, etc.
func (cis *CargoInsuranceSystem) RecordCargoLoss(playerName string, value int) {
	if cis.losses == nil {
		cis.losses = make(map[string]int)
	}
	cis.losses[playerName] += value
}

// GetCoverage returns the insurance coverage percentage for a faction.
func (cis *CargoInsuranceSystem) GetCoverage(playerName string) float64 {
	if cis.coverage == nil {
		return 0
	}
	return cis.coverage[playerName]
}
