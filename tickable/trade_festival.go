package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeFestivalSystem{
		BaseSystem: NewBaseSystem("TradeFestival", 90),
	})
}

// TradeFestivalSystem generates periodic trade festivals in systems
// with high commerce activity. During a festival:
//
//   - All trade fees reduced to 0% (free trade)
//   - Population happiness +10% in the system
//   - Credit generation doubled for all planets in the system
//   - Attracts traders: all factions get a small credit bonus
//     for having ships in the festival system
//
// Festivals last 3000 ticks (~5 minutes) and occur in the system
// with the highest Trading Post levels.
//
// This creates periodic trade events worth planning around —
// move your cargo ships to the festival system for bonuses.
type TradeFestivalSystem struct {
	*BaseSystem
	festival     *TradeFestival
	nextFestival int64
}

// TradeFestival represents an active trade celebration.
type TradeFestival struct {
	SystemID  int
	SysName   string
	TicksLeft int
	Active    bool
}

func (tfs *TradeFestivalSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := tfs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tfs.nextFestival == 0 {
		tfs.nextFestival = tick + 10000 + int64(rand.Intn(10000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Process active festival
	if tfs.festival != nil && tfs.festival.Active {
		tfs.festival.TicksLeft -= 500
		if tfs.festival.TicksLeft <= 0 {
			tfs.festival.Active = false
			game.LogEvent("event", "",
				fmt.Sprintf("🎪 Trade Festival in %s has ended! Markets returning to normal",
					tfs.festival.SysName))
		} else {
			tfs.applyFestivalBonuses(tfs.festival, players, systems, game)
		}
		return
	}

	// Start new festival
	if tick >= tfs.nextFestival {
		tfs.nextFestival = tick + 15000 + int64(rand.Intn(10000))
		tfs.startFestival(game, systems)
	}
}

func (tfs *TradeFestivalSystem) applyFestivalBonuses(fest *TradeFestival, players []*entities.Player, systems []*entities.System, game GameProvider) {
	for _, sys := range systems {
		if sys.ID != fest.SystemID {
			continue
		}

		// Happiness boost + credit bonus for planets in festival system
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				planet.Happiness += 0.02
				if planet.Happiness > 1.0 {
					planet.Happiness = 1.0
				}
				// Credit bonus for planet owners
				for _, p := range players {
					if p != nil && p.Name == planet.Owner {
						p.Credits += 10
						break
					}
				}
			}
		}

		// Bonus for ships present at festival
		for _, p := range players {
			if p == nil {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship != nil && ship.CurrentSystem == fest.SystemID && ship.Status != entities.ShipStatusMoving {
					p.Credits += 5 // small attendance bonus
				}
			}
		}
		break
	}
}

func (tfs *TradeFestivalSystem) startFestival(game GameProvider, systems []*entities.System) {
	// Find system with highest TP levels
	bestSys := -1
	bestName := ""
	bestTP := 0

	for _, sys := range systems {
		tpTotal := 0
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
						tpTotal += b.Level
					}
				}
			}
		}
		if tpTotal > bestTP {
			bestTP = tpTotal
			bestSys = sys.ID
			bestName = sys.Name
		}
	}

	if bestSys < 0 || bestTP == 0 {
		return
	}

	tfs.festival = &TradeFestival{
		SystemID:  bestSys,
		SysName:   bestName,
		TicksLeft: 3000 + rand.Intn(2000),
		Active:    true,
	}

	game.LogEvent("event", "",
		fmt.Sprintf("🎪 TRADE FESTIVAL in %s! Free trade, double credits, +happiness for ~%d minutes! Send ships for attendance bonuses!",
			bestName, tfs.festival.TicksLeft/600))
}

// IsFestivalActive returns whether a festival is happening.
func (tfs *TradeFestivalSystem) IsFestivalActive() bool {
	return tfs.festival != nil && tfs.festival.Active
}

// GetFestivalSystem returns the system ID of the active festival.
func (tfs *TradeFestivalSystem) GetFestivalSystem() int {
	if tfs.festival != nil && tfs.festival.Active {
		return tfs.festival.SystemID
	}
	return -1
}
