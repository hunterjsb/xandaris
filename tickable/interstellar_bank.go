package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&InterstellarBankSystem{
		BaseSystem: NewBaseSystem("InterstellarBank", 94),
	})
}

// InterstellarBankSystem provides banking services: loans, interest
// on deposits, and credit transfers between factions.
//
// Services:
//   Savings: factions with >100K credits earn 0.1% interest per interval
//   Loans: factions with <5000 credits can take a loan (10K cr, 15% interest)
//   Debt: unpaid loans accrue interest and eventually trigger asset seizure
//
// The bank creates financial depth:
//   - Wealthy factions earn passive income (compound interest)
//   - Struggling factions can borrow to bootstrap
//   - Debt creates pressure to generate revenue
//   - Bank announces "interest rates" galaxy-wide
type InterstellarBankSystem struct {
	*BaseSystem
	loans      map[string]*Loan // factionName → active loan
	interestRate float64
	nextUpdate int64
}

// Loan tracks an outstanding loan.
type Loan struct {
	Principal  int
	Outstanding int // principal + accrued interest
	TakenAt    int64
}

func (ibs *InterstellarBankSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := ibs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ibs.loans == nil {
		ibs.loans = make(map[string]*Loan)
		ibs.interestRate = 0.001 // 0.1% per interval
	}

	players := ctx.GetPlayers()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Savings interest for wealthy factions
		if player.Credits > 100000 {
			interest := int(float64(player.Credits) * ibs.interestRate)
			if interest > 500 {
				interest = 500 // cap per interval
			}
			player.Credits += interest
		}

		// Auto-loan for struggling factions
		loan := ibs.loans[player.Name]
		if loan != nil {
			// Accrue interest on outstanding debt
			debtInterest := int(float64(loan.Outstanding) * 0.005) // 0.5% per interval
			loan.Outstanding += debtInterest

			// Auto-repay if faction has enough credits
			if player.Credits > loan.Outstanding*2 {
				player.Credits -= loan.Outstanding
				game.LogEvent("trade", player.Name,
					fmt.Sprintf("🏦 %s repaid bank loan of %dcr. Debt cleared!",
						player.Name, loan.Outstanding))
				delete(ibs.loans, player.Name)
			} else if tick-loan.TakenAt > 30000 {
				// Overdue — force partial repayment
				payment := player.Credits / 10
				if payment > 0 {
					player.Credits -= payment
					loan.Outstanding -= payment
				}
			}
		} else if player.Credits < 5000 && player.Credits > 0 {
			// Offer loan
			if rand.Intn(5) == 0 { // don't spam
				loanAmount := 10000
				ibs.loans[player.Name] = &Loan{
					Principal:   loanAmount,
					Outstanding: loanAmount,
					TakenAt:     tick,
				}
				player.Credits += loanAmount
				game.LogEvent("trade", player.Name,
					fmt.Sprintf("🏦 Interstellar Bank issued %s a %dcr loan. Repay when able (0.5%% interest per interval)",
						player.Name, loanAmount))
			}
		}
	}

	// Announce rate changes
	if ibs.nextUpdate == 0 {
		ibs.nextUpdate = tick + 20000
	}
	if tick >= ibs.nextUpdate {
		ibs.nextUpdate = tick + 20000 + int64(rand.Intn(10000))
		// Fluctuate rates slightly
		ibs.interestRate = 0.0005 + rand.Float64()*0.001
		game.LogEvent("intel", "",
			fmt.Sprintf("🏦 Interstellar Bank rate update: savings %.2f%% per interval. Deposits over 100K earn passive income!",
				ibs.interestRate*100))
	}
}

// GetDebt returns a faction's outstanding loan amount.
func (ibs *InterstellarBankSystem) GetDebt(faction string) int {
	if ibs.loans == nil {
		return 0
	}
	if loan, ok := ibs.loans[faction]; ok {
		return loan.Outstanding
	}
	return 0
}
