package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SupplyChainScoreSystem{
		BaseSystem: NewBaseSystem("SupplyChainScore", 145),
	})
}

// SupplyChainScoreSystem evaluates how complete each faction's
// production chain is and identifies missing links.
//
// Complete chain: Mine→Oil→Refinery→Fuel→Generator→Power→Factory→Electronics→Tech
//
// Each link present = +1. Score out of 8:
//   8/8: "Perfect chain" — +500cr bonus
//   6-7: "Strong chain" — working well
//   4-5: "Incomplete" — missing critical steps
//   0-3: "Broken chain" — economy can't function
//
// Reports what's missing: "Missing: Refinery (need Oil→Fuel), Factory (need Electronics)"
type SupplyChainScoreSystem struct {
	*BaseSystem
	nextReport int64
}

func (scss *SupplyChainScoreSystem) OnTick(tick int64) {
	ctx := scss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if scss.nextReport == 0 {
		scss.nextReport = tick + 8000 + int64(rand.Intn(5000))
	}
	if tick < scss.nextReport {
		return
	}
	scss.nextReport = tick + 12000 + int64(rand.Intn(8000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Check chain links across all planets
		hasMine := false
		hasOilDeposit := false
		hasRefinery := false
		hasGenerator := false
		hasFactory := false
		hasShipyard := false
		hasTP := false
		hasHabitat := false

		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner != player.Name {
					continue
				}

				for _, re := range planet.Resources {
					if r, ok := re.(*entities.Resource); ok && r.ResourceType == entities.ResOil {
						hasOilDeposit = true
					}
				}

				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.IsOperational {
						switch b.BuildingType {
						case entities.BuildingMine:
							hasMine = true
						case entities.BuildingRefinery:
							hasRefinery = true
						case entities.BuildingGenerator:
							hasGenerator = true
						case entities.BuildingFactory:
							hasFactory = true
						case entities.BuildingShipyard:
							hasShipyard = true
						case entities.BuildingTradingPost:
							hasTP = true
						case entities.BuildingHabitat:
							hasHabitat = true
						}
					}
				}
			}
		}

		score := 0
		var missing []string
		links := []struct {
			has  bool
			name string
			hint string
		}{
			{hasMine, "Mine", "extract raw resources"},
			{hasOilDeposit, "Oil deposit", "colonize planet with Oil"},
			{hasRefinery, "Refinery", "Oil→Fuel conversion"},
			{hasGenerator, "Generator", "Fuel→Power"},
			{hasFactory, "Factory", "produce Electronics"},
			{hasTP, "Trading Post", "market access"},
			{hasShipyard, "Shipyard", "build ships"},
			{hasHabitat, "Habitat", "house more workers"},
		}

		for _, link := range links {
			if link.has {
				score++
			} else {
				missing = append(missing, fmt.Sprintf("%s (%s)", link.name, link.hint))
			}
		}

		// Only report for factions with planets
		planetCount := 0
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
					planetCount++
				}
			}
		}
		if planetCount == 0 {
			continue
		}

		status := "Broken"
		if score >= 8 {
			status = "Perfect"
			player.Credits += 500
		} else if score >= 6 {
			status = "Strong"
		} else if score >= 4 {
			status = "Incomplete"
		}

		msg := fmt.Sprintf("🔗 %s supply chain: %d/8 (%s)", player.Name, score, status)
		if len(missing) > 0 && len(missing) <= 3 {
			msg += " | Missing: "
			for i, m := range missing {
				if i > 0 {
					msg += ", "
				}
				msg += m
			}
		}

		game.LogEvent("logistics", player.Name, msg)
	}
}
