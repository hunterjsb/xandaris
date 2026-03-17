package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeLedgerSystem{
		BaseSystem: NewBaseSystem("TradeLedger", 50),
	})
}

// TradeLedgerSystem tracks per-faction trade economics and announces
// periodic financial reports. This gives factions (including LLM agents)
// visibility into whether their trade operations are profitable.
//
// Tracked metrics per faction:
//   - Total goods shipped (units)
//   - Total trade income (credits earned from sales)
//   - Total trade spending (credits spent on purchases)
//   - Net trade profit
//   - Shipping route efficiency (trips completed / routes active)
//   - Fleet utilization (cargo ships hauling / total cargo ships)
//
// Reports fire every ~5000 ticks with a summary.
type TradeLedgerSystem struct {
	*BaseSystem
	ledger     map[string]*FactionLedger // playerName → ledger
	nextReport int64
}

// FactionLedger tracks trade economics for a single faction.
type FactionLedger struct {
	GoodsShipped   int
	TradeIncome    int
	TradeSpending  int
	RoutesActive   int
	TripsCompleted int
	FleetCargo     int // cargo ships total
	FleetHauling   int // cargo ships currently carrying goods
}

func (tls *TradeLedgerSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := tls.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tls.ledger == nil {
		tls.ledger = make(map[string]*FactionLedger)
	}

	if tls.nextReport == 0 {
		tls.nextReport = tick + 3000 + int64(rand.Intn(2000))
	}

	players := ctx.GetPlayers()

	// Update fleet utilization snapshot
	for _, player := range players {
		if player == nil {
			continue
		}
		if tls.ledger[player.Name] == nil {
			tls.ledger[player.Name] = &FactionLedger{}
		}
		ledger := tls.ledger[player.Name]

		cargoTotal := 0
		cargoHauling := 0
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo {
				continue
			}
			cargoTotal++
			if ship.GetTotalCargo() > 0 || ship.Status == entities.ShipStatusMoving {
				cargoHauling++
			}
		}
		ledger.FleetCargo = cargoTotal
		ledger.FleetHauling = cargoHauling
	}

	// Update route stats
	routes := game.GetShippingRoutes()
	routeCounts := make(map[string]int)
	tripCounts := make(map[string]int)
	for _, route := range routes {
		if route.Active {
			routeCounts[route.Owner]++
			tripCounts[route.Owner] += route.TripsComplete
		}
	}
	for name, ledger := range tls.ledger {
		ledger.RoutesActive = routeCounts[name]
		ledger.TripsCompleted = tripCounts[name]
	}

	// Periodic financial report
	if tick >= tls.nextReport {
		tls.nextReport = tick + 5000 + int64(rand.Intn(3000))
		tls.generateReports(game, players)
	}
}

func (tls *TradeLedgerSystem) generateReports(game GameProvider, players []*entities.Player) {
	for _, player := range players {
		if player == nil {
			continue
		}
		ledger := tls.ledger[player.Name]
		if ledger == nil {
			continue
		}

		utilization := 0.0
		if ledger.FleetCargo > 0 {
			utilization = float64(ledger.FleetHauling) / float64(ledger.FleetCargo) * 100
		}

		// Only report for factions with some logistics activity
		if ledger.FleetCargo == 0 && ledger.RoutesActive == 0 {
			continue
		}

		game.LogEvent("logistics", player.Name,
			fmt.Sprintf("📊 %s Logistics Report: %d cargo ships (%.0f%% active), %d routes, %d trips completed",
				player.Name, ledger.FleetCargo, utilization,
				ledger.RoutesActive, ledger.TripsCompleted))
	}
}

// GetLedger returns the trade ledger for a faction.
func (tls *TradeLedgerSystem) GetLedger(playerName string) *FactionLedger {
	if tls.ledger == nil {
		return nil
	}
	return tls.ledger[playerName]
}

// RecordTrade logs a trade for ledger tracking (called by external systems).
func (tls *TradeLedgerSystem) RecordTrade(playerName string, income, spending, unitsShipped int) {
	if tls.ledger == nil {
		tls.ledger = make(map[string]*FactionLedger)
	}
	if tls.ledger[playerName] == nil {
		tls.ledger[playerName] = &FactionLedger{}
	}
	tls.ledger[playerName].TradeIncome += income
	tls.ledger[playerName].TradeSpending += spending
	tls.ledger[playerName].GoodsShipped += unitsShipped
}
