package tickable

import (
	"fmt"
	"sort"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&LogisticsScoreSystem{
		BaseSystem: NewBaseSystem("LogisticsScore", 53),
	})
}

// LogisticsScoreSystem computes a holistic logistics score per faction.
// The score measures how well a faction's supply chain is functioning,
// rewarding efficient logistics over raw credit accumulation.
//
// Scoring components:
//   - Fleet diversity: bonus for having Scouts, Cargo, and Military (max 20)
//   - Route efficiency: trips completed / routes active * 10 (max 30)
//   - Resource diversity: types stocked across all planets (max 20)
//   - Trading Post coverage: TP levels across planets (max 15)
//   - Supply chain health: % of planets with all critical resources (max 15)
//
// The top faction's score is announced every ~8000 ticks.
// The score is available via API for dashboard display.
type LogisticsScoreSystem struct {
	*BaseSystem
	scores     map[string]int // playerName → logistics score
	nextReport int64
}

func (lss *LogisticsScoreSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := lss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if lss.scores == nil {
		lss.scores = make(map[string]int)
	}

	if lss.nextReport == 0 {
		lss.nextReport = tick + 5000
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	routes := game.GetShippingRoutes()

	for _, player := range players {
		if player == nil {
			continue
		}
		lss.scores[player.Name] = lss.computeScore(player, systems, routes)
	}

	// Periodic leaderboard announcement
	if tick >= lss.nextReport {
		lss.nextReport = tick + 8000
		lss.announceLeaderboard(game, players)
	}
}

func (lss *LogisticsScoreSystem) computeScore(player *entities.Player, systems []*entities.System, routes []ShippingRouteInfo) int {
	score := 0

	// 1. Fleet diversity (max 20)
	hasScout, hasCargo, hasMilitary := false, false, false
	cargoCount := 0
	for _, ship := range player.OwnedShips {
		if ship == nil {
			continue
		}
		switch ship.ShipType {
		case entities.ShipTypeScout:
			hasScout = true
		case entities.ShipTypeCargo:
			hasCargo = true
			cargoCount++
		case entities.ShipTypeFrigate, entities.ShipTypeDestroyer, entities.ShipTypeCruiser:
			hasMilitary = true
		}
	}
	if hasScout {
		score += 5
	}
	if hasCargo {
		score += 5
	}
	if hasMilitary {
		score += 5
	}
	if cargoCount >= 3 {
		score += 5 // bonus for multiple freighters
	}

	// 2. Route efficiency (max 30)
	activeRoutes := 0
	totalTrips := 0
	for _, route := range routes {
		if route.Owner == player.Name && route.Active {
			activeRoutes++
			totalTrips += route.TripsComplete
		}
	}
	if activeRoutes > 0 {
		efficiency := float64(totalTrips) / float64(activeRoutes)
		routeScore := int(efficiency * 10)
		if routeScore > 30 {
			routeScore = 30
		}
		score += routeScore
	}

	// 3. Resource diversity across planets (max 20)
	resourceTypes := make(map[string]bool)
	planetCount := 0
	totalTPLevel := 0
	healthyPlanets := 0

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner != player.Name {
				continue
			}
			planetCount++

			// Count resource types stocked
			critical := []string{entities.ResIron, entities.ResWater, entities.ResFuel}
			hasCritical := true
			for _, res := range critical {
				if planet.GetStoredAmount(res) > 10 {
					resourceTypes[res] = true
				} else {
					hasCritical = false
				}
			}
			for _, res := range []string{entities.ResOil, entities.ResHelium3, entities.ResRareMetals, entities.ResElectronics} {
				if planet.GetStoredAmount(res) > 0 {
					resourceTypes[res] = true
				}
			}
			if hasCritical {
				healthyPlanets++
			}

			// TP levels
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
					totalTPLevel += b.Level
				}
			}
		}
	}
	diversityScore := len(resourceTypes) * 3
	if diversityScore > 20 {
		diversityScore = 20
	}
	score += diversityScore

	// 4. Trading Post coverage (max 15)
	tpScore := totalTPLevel * 3
	if tpScore > 15 {
		tpScore = 15
	}
	score += tpScore

	// 5. Supply chain health (max 15)
	if planetCount > 0 {
		healthPct := float64(healthyPlanets) / float64(planetCount)
		score += int(healthPct * 15)
	}

	return score
}

func (lss *LogisticsScoreSystem) announceLeaderboard(game GameProvider, players []*entities.Player) {
	type entry struct {
		name  string
		score int
	}
	var entries []entry
	for _, p := range players {
		if p == nil {
			continue
		}
		if s, ok := lss.scores[p.Name]; ok && s > 0 {
			entries = append(entries, entry{p.Name, s})
		}
	}
	if len(entries) == 0 {
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].score > entries[j].score
	})

	msg := "📦 Logistics Leaderboard: "
	for i, e := range entries {
		if i >= 3 {
			break
		}
		medal := "🥉"
		if i == 0 {
			medal = "🥇"
		} else if i == 1 {
			medal = "🥈"
		}
		msg += fmt.Sprintf("%s %s (%d) ", medal, e.name, e.score)
	}
	game.LogEvent("event", "", msg)
}

// GetScore returns the logistics score for a faction.
func (lss *LogisticsScoreSystem) GetScore(playerName string) int {
	if lss.scores == nil {
		return 0
	}
	return lss.scores[playerName]
}
