package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FleetManagerSystem{
		BaseSystem: NewBaseSystem("FleetManager", 52),
	})
}

// FleetManagerSystem monitors fleet composition and flags waste.
// AI factions tend to build too many Colony ships and not enough
// Cargo ships, leading to fuel drain without logistics capability.
//
// Checks:
//   - Colony ships that have already colonized (Colonists=0): flag for scrap
//   - Idle Cargo ships with no route: suggest route assignment
//   - Ships stranded with 0 fuel far from owned planets: flag for rescue
//   - Fleet composition ratio warnings
//
// Also provides fleet summary events so LLM agents can make decisions.
type FleetManagerSystem struct {
	*BaseSystem
	lastReport map[string]int64 // playerName → last report tick
}

func (fms *FleetManagerSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := fms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if fms.lastReport == nil {
		fms.lastReport = make(map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil || len(player.OwnedShips) < 3 {
			continue
		}

		// Rate limit: one report per 10000 ticks per faction
		if tick-fms.lastReport[player.Name] < 10000 {
			continue
		}

		fms.analyzeFleet(tick, player, systems, game)
	}
}

func (fms *FleetManagerSystem) analyzeFleet(tick int64, player *entities.Player, systems []*entities.System, game GameProvider) {
	scouts, cargo, colony, frigates, destroyers, cruisers := 0, 0, 0, 0, 0, 0
	emptyColonies := 0
	idleCargo := 0
	strandedShips := 0

	// Identify owned system IDs
	ownedSystems := make(map[int]bool)
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if p, ok := e.(*entities.Planet); ok && p.Owner == player.Name {
				ownedSystems[sys.ID] = true
			}
		}
	}

	for _, ship := range player.OwnedShips {
		if ship == nil {
			continue
		}

		switch ship.ShipType {
		case entities.ShipTypeScout:
			scouts++
		case entities.ShipTypeCargo:
			cargo++
			if ship.Status != entities.ShipStatusMoving && ship.GetTotalCargo() == 0 && ship.DeliveryID == 0 {
				idleCargo++
			}
		case entities.ShipTypeColony:
			colony++
			if ship.Colonists == 0 {
				emptyColonies++
			}
		case entities.ShipTypeFrigate:
			frigates++
		case entities.ShipTypeDestroyer:
			destroyers++
		case entities.ShipTypeCruiser:
			cruisers++
		}

		// Stranded: 0 fuel, not in owned system, not moving
		if ship.CurrentFuel == 0 && ship.Status != entities.ShipStatusMoving && !ownedSystems[ship.CurrentSystem] {
			strandedShips++
		}
	}

	// Only report if there's something worth flagging
	issues := make([]string, 0)

	if emptyColonies > 3 {
		issues = append(issues, fmt.Sprintf("%d empty Colony ships (scrap for resources!)", emptyColonies))
	}
	if idleCargo > 2 {
		issues = append(issues, fmt.Sprintf("%d idle Cargo ships (assign routes!)", idleCargo))
	}
	if strandedShips > 0 {
		issues = append(issues, fmt.Sprintf("%d ships stranded (0 fuel, foreign system)", strandedShips))
	}
	if cargo == 0 && colony > 5 {
		issues = append(issues, "no Cargo ships! Build freighters for trade")
	}

	if len(issues) > 0 {
		fms.lastReport[player.Name] = tick
		msg := fmt.Sprintf("🚢 %s Fleet Report: %d ships [%dS/%dC/%dCol/%dF/%dD/%dCr]",
			player.Name, len(player.OwnedShips), scouts, cargo, colony, frigates, destroyers, cruisers)
		for _, issue := range issues {
			msg += " | " + issue
		}
		game.LogEvent("logistics", player.Name, msg)
	}
}
