package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&InvestmentFundSystem{
		BaseSystem: NewBaseSystem("InvestmentFund", 108),
	})
}

// InvestmentFundSystem lets factions invest in each other's economies.
// When faction A invests in faction B, A pays credits upfront and
// receives a share of B's income over time.
//
// Investment mechanics:
//   - Auto-invest: wealthy factions (>500K) invest 1% in the faction
//     with the highest trade reputation (if relations are Friendly+)
//   - Returns: investor gets 0.5% of investee's credit income per interval
//   - Duration: investments last 20,000 ticks then mature
//   - Risk: if investee goes bankrupt, investment is lost
//
// This creates financial interdependence: your ally's success is
// your success. Attacking a faction that others are invested in
// makes you enemies with investors too.
type InvestmentFundSystem struct {
	*BaseSystem
	investments []*Investment
	nextCheck   int64
}

// Investment tracks a cross-faction financial stake.
type Investment struct {
	Investor   string
	Investee   string
	Amount     int
	TicksLeft  int
	Returns    int // total returns earned so far
	Active     bool
}

func (ifs *InvestmentFundSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := ifs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	dm := game.GetDiplomacyManager()
	if dm == nil {
		return
	}

	players := ctx.GetPlayers()

	// Process active investments
	for _, inv := range ifs.investments {
		if !inv.Active {
			continue
		}
		inv.TicksLeft -= 1000
		if inv.TicksLeft <= 0 {
			// Matured — return principal
			for _, p := range players {
				if p != nil && p.Name == inv.Investor {
					p.Credits += inv.Amount
					game.LogEvent("trade", inv.Investor,
						fmt.Sprintf("💼 Investment in %s matured! Principal %dcr returned + %dcr in dividends earned",
							inv.Investee, inv.Amount, inv.Returns))
					break
				}
			}
			inv.Active = false
			continue
		}

		// Pay dividends: 0.5% of investee's credits
		var investee *entities.Player
		for _, p := range players {
			if p != nil && p.Name == inv.Investee {
				investee = p
				break
			}
		}
		if investee == nil || investee.Credits <= 0 {
			continue
		}

		dividend := investee.Credits / 200 // 0.5%
		if dividend > 100 {
			dividend = 100 // cap per interval
		}
		if dividend <= 0 {
			continue
		}

		for _, p := range players {
			if p != nil && p.Name == inv.Investor {
				p.Credits += dividend
				inv.Returns += dividend
				break
			}
		}
	}

	// Create new investments
	if ifs.nextCheck == 0 {
		ifs.nextCheck = tick + 10000
	}
	if tick < ifs.nextCheck {
		return
	}
	ifs.nextCheck = tick + 15000 + int64(rand.Intn(10000))

	// Find wealthy factions to invest
	for _, investor := range players {
		if investor == nil || investor.Credits < 500000 {
			continue
		}

		// Already has active investment?
		hasInvestment := false
		for _, inv := range ifs.investments {
			if inv.Active && inv.Investor == investor.Name {
				hasInvestment = true
				break
			}
		}
		if hasInvestment {
			continue
		}

		// Find best investee (friendly, different faction)
		for _, investee := range players {
			if investee == nil || investee.Name == investor.Name {
				continue
			}
			rel := dm.GetRelation(investor.Name, investee.Name)
			if rel < 1 {
				continue
			}

			// Invest 1% of credits
			amount := investor.Credits / 100
			if amount < 5000 {
				continue
			}
			if amount > 50000 {
				amount = 50000
			}

			investor.Credits -= amount
			ifs.investments = append(ifs.investments, &Investment{
				Investor:  investor.Name,
				Investee:  investee.Name,
				Amount:    amount,
				TicksLeft: 20000,
				Active:    true,
			})

			game.LogEvent("trade", investor.Name,
				fmt.Sprintf("💼 %s invested %dcr in %s's economy! Dividends incoming for ~33 minutes",
					investor.Name, amount, investee.Name))
			break
		}
	}
}
