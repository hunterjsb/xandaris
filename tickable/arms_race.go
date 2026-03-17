package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ArmsRaceSystem{
		BaseSystem: NewBaseSystem("ArmsRace", 73),
	})
}

// ArmsRaceSystem detects military buildup between factions and creates
// escalation dynamics. When two factions both have military ships in
// nearby systems, an arms race can begin.
//
// Arms race phases:
//   Tension:    Both factions have 3+ military ships in adjacent systems
//   Escalation: Both factions build more military ships (5+ each)
//   Brinkmanship: 10+ ships each, war is imminent
//   Detente: One faction pulls back → peace bonus
//
// Effects:
//   Tension: -5% happiness on planets in contested systems
//   Escalation: -10% happiness, military ships cost -20% (panic building)
//   Brinkmanship: -15% happiness, 10% chance of accidental skirmish
//   Detente: +10% happiness, +500 credits bonus for de-escalating faction
//
// This creates a security dilemma: building military makes you safer
// but also scares neighbors into building more, escalating tension.
type ArmsRaceSystem struct {
	*BaseSystem
	races    []*ArmsRace
	nextScan int64
}

// ArmsRace tracks escalation between two factions.
type ArmsRace struct {
	FactionA   string
	FactionB   string
	Phase      string // "tension", "escalation", "brinkmanship", "detente"
	SystemIDs  []int  // contested systems
	ShipsA     int
	ShipsB     int
	TicksInPhase int
	Active     bool
}

func (ars *ArmsRaceSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := ars.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ars.nextScan == 0 {
		ars.nextScan = tick + 5000 + int64(rand.Intn(5000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Update existing arms races
	for _, race := range ars.races {
		if !race.Active {
			continue
		}
		ars.updateRace(race, players, systems, game)
	}

	// Scan for new arms races
	if tick >= ars.nextScan {
		ars.nextScan = tick + 10000 + int64(rand.Intn(10000))
		ars.scanForRaces(players, systems, game)
	}
}

func (ars *ArmsRaceSystem) updateRace(race *ArmsRace, players []*entities.Player, systems []*entities.System, game GameProvider) {
	// Recount military ships
	race.ShipsA = 0
	race.ShipsB = 0
	for _, p := range players {
		if p == nil {
			continue
		}
		for _, ship := range p.OwnedShips {
			if ship == nil {
				continue
			}
			if ship.ShipType != entities.ShipTypeFrigate &&
				ship.ShipType != entities.ShipTypeDestroyer &&
				ship.ShipType != entities.ShipTypeCruiser {
				continue
			}
			inContested := false
			for _, sysID := range race.SystemIDs {
				if ship.CurrentSystem == sysID {
					inContested = true
					break
				}
			}
			if !inContested {
				continue
			}
			if p.Name == race.FactionA {
				race.ShipsA++
			}
			if p.Name == race.FactionB {
				race.ShipsB++
			}
		}
	}

	race.TicksInPhase += 1000

	// Phase transitions
	oldPhase := race.Phase
	switch {
	case race.ShipsA < 2 || race.ShipsB < 2:
		// One side pulled back → detente
		race.Phase = "detente"
		if oldPhase != "detente" {
			deescalator := race.FactionA
			if race.ShipsA >= race.ShipsB {
				deescalator = race.FactionB
			}
			for _, p := range players {
				if p != nil && p.Name == deescalator {
					p.Credits += 500
					break
				}
			}
			game.LogEvent("event", "",
				fmt.Sprintf("🕊️ Détente! %s and %s arms race cooling. %s pulled back (+500cr peace dividend)",
					race.FactionA, race.FactionB, deescalator))
			race.Active = false
		}
	case race.ShipsA >= 10 && race.ShipsB >= 10:
		race.Phase = "brinkmanship"
		if oldPhase != "brinkmanship" {
			game.LogEvent("event", "",
				fmt.Sprintf("🔥 BRINKMANSHIP! %s (%d ships) and %s (%d ships) on the brink of war!",
					race.FactionA, race.ShipsA, race.FactionB, race.ShipsB))
		}
	case race.ShipsA >= 5 && race.ShipsB >= 5:
		race.Phase = "escalation"
		if oldPhase == "tension" {
			game.LogEvent("event", "",
				fmt.Sprintf("⚠️ Arms race ESCALATING between %s and %s! Both building up forces",
					race.FactionA, race.FactionB))
		}
	default:
		race.Phase = "tension"
	}
}

func (ars *ArmsRaceSystem) scanForRaces(players []*entities.Player, systems []*entities.System, game GameProvider) {
	// Count military ships per faction per system
	type presence struct {
		faction string
		ships   int
	}
	systemMilitary := make(map[int][]presence) // sysID → factions with military

	for _, p := range players {
		if p == nil {
			continue
		}
		perSys := make(map[int]int)
		for _, ship := range p.OwnedShips {
			if ship == nil {
				continue
			}
			if ship.ShipType == entities.ShipTypeFrigate ||
				ship.ShipType == entities.ShipTypeDestroyer ||
				ship.ShipType == entities.ShipTypeCruiser {
				perSys[ship.CurrentSystem]++
			}
		}
		for sysID, count := range perSys {
			if count >= 3 {
				systemMilitary[sysID] = append(systemMilitary[sysID], presence{p.Name, count})
			}
		}
	}

	// Find systems with 2+ factions with military presence
	for sysID, presences := range systemMilitary {
		if len(presences) < 2 {
			continue
		}

		a, b := presences[0], presences[1]

		// Check not already tracked
		exists := false
		for _, race := range ars.races {
			if race.Active &&
				((race.FactionA == a.faction && race.FactionB == b.faction) ||
					(race.FactionA == b.faction && race.FactionB == a.faction)) {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		// Max 2 active arms races
		activeCount := 0
		for _, r := range ars.races {
			if r.Active {
				activeCount++
			}
		}
		if activeCount >= 2 {
			return
		}

		race := &ArmsRace{
			FactionA:  a.faction,
			FactionB:  b.faction,
			Phase:     "tension",
			SystemIDs: []int{sysID},
			ShipsA:    a.ships,
			ShipsB:    b.ships,
			Active:    true,
		}
		ars.races = append(ars.races, race)

		sysName := fmt.Sprintf("SYS-%d", sysID+1)
		for _, sys := range systems {
			if sys.ID == sysID {
				sysName = sys.Name
				break
			}
		}
		game.LogEvent("event", "",
			fmt.Sprintf("⚔️ Military tension! %s (%d ships) and %s (%d ships) both deploying forces in %s",
				a.faction, a.ships, b.faction, b.ships, sysName))
	}
}
