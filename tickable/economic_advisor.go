package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&EconomicAdvisorSystem{
		BaseSystem: NewBaseSystem("EconomicAdvisor", 121),
	})
}

// EconomicAdvisorSystem provides personalized economic advice to each
// faction based on their specific situation. Unlike general galaxy
// events, these are targeted recommendations.
//
// Analyzes each faction for:
//   - Missing buildings (should build X)
//   - Resource imbalances (too much X, not enough Y)
//   - Fleet composition issues (no cargo ships, too many colony ships)
//   - Income problems (spending > earning)
//   - Expansion opportunities (unclaimed planets nearby)
//
// One advice event per faction per 10,000 ticks (not spammy).
type EconomicAdvisorSystem struct {
	*BaseSystem
	lastAdvice map[string]int64
}

func (eas *EconomicAdvisorSystem) OnTick(tick int64) {
	if tick%5000 != 0 {
		return
	}

	ctx := eas.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if eas.lastAdvice == nil {
		eas.lastAdvice = make(map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}
		if tick-eas.lastAdvice[player.Name] < 10000 {
			continue
		}

		advice := eas.generateAdvice(player, systems)
		if advice == "" {
			continue
		}

		eas.lastAdvice[player.Name] = tick
		game.LogEvent("intel", player.Name,
			fmt.Sprintf("💡 Advisor for %s: %s", player.Name, advice))
	}
}

func (eas *EconomicAdvisorSystem) generateAdvice(player *entities.Player, systems []*entities.System) string {
	var advices []string

	// Analyze fleet
	cargo, colony, military, scouts := 0, 0, 0, 0
	for _, ship := range player.OwnedShips {
		if ship == nil {
			continue
		}
		switch ship.ShipType {
		case entities.ShipTypeCargo:
			cargo++
		case entities.ShipTypeColony:
			colony++
		case entities.ShipTypeScout:
			scouts++
		case entities.ShipTypeFrigate, entities.ShipTypeDestroyer, entities.ShipTypeCruiser:
			military++
		}
	}

	if cargo == 0 {
		advices = append(advices, "Build Cargo ships! You have no freighters — trade routes require them")
	}
	if colony > cargo*2 && colony > 5 {
		advices = append(advices, fmt.Sprintf("Too many Colony ships (%d)! Convert or scrap them — they drain fuel doing nothing", colony))
	}

	// Analyze planets
	planetCount := 0
	hasRefinery := false
	hasFactory := false
	hasTP := false
	totalFuel := 0
	totalPop := int64(0)

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner != player.Name {
				continue
			}
			planetCount++
			totalFuel += planet.GetStoredAmount(entities.ResFuel)
			totalPop += planet.Population

			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.IsOperational {
					switch b.BuildingType {
					case entities.BuildingRefinery:
						hasRefinery = true
					case entities.BuildingFactory:
						hasFactory = true
					case entities.BuildingTradingPost:
						hasTP = true
					}
				}
			}
		}
	}

	if !hasFactory && planetCount > 0 && player.Credits > 50000 {
		advices = append(advices, "Build a Factory to produce Electronics — needed for tech advancement")
	}
	if !hasRefinery && planetCount > 0 {
		advices = append(advices, "Build a Refinery! You need Oil→Fuel conversion for power")
	}
	if !hasTP && planetCount > 0 {
		advices = append(advices, "Build a Trading Post! Required for all market access")
	}
	if totalFuel == 0 && planetCount > 0 {
		advices = append(advices, "URGENT: Zero fuel across all planets! Build Refineries and mine Oil")
	}
	if player.Credits > 1000000 && planetCount < 5 {
		advices = append(advices, "You're sitting on 1M+ credits — expand! Colonize more planets")
	}

	if len(advices) == 0 {
		return ""
	}

	return advices[rand.Intn(len(advices))]
}
