package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&DiplomaticIncidentSystem{
		BaseSystem: NewBaseSystem("DiplomaticIncidents", 100),
	})
}

// DiplomaticIncidentSystem generates random diplomatic crises between
// factions that test alliances and create storylines.
//
// Incidents:
//   Border Dispute: two factions claim the same unclaimed planet.
//     Both lose 500cr in "diplomatic costs". First to colonize wins.
//
//   Spy Scandal: faction A's spy caught in faction B's system.
//     Relations drop by 1. A pays 2000cr in reparations.
//
//   Trade Dispute: faction A accuses B of dumping cheap goods.
//     Trade between them frozen for 3000 ticks.
//
//   Cultural Misunderstanding: accidental insult between factions.
//     Relations drop by 1 but can be repaired with a 1000cr "gift".
//
//   Refugee Incident: faction A's refugees flood faction B's planet.
//     B's happiness drops 5% but population grows.
//
// Incidents are more likely between factions with Unfriendly relations.
type DiplomaticIncidentSystem struct {
	*BaseSystem
	nextIncident int64
}

func (dis *DiplomaticIncidentSystem) OnTick(tick int64) {
	ctx := dis.GetContext()
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

	if dis.nextIncident == 0 {
		dis.nextIncident = tick + 8000 + int64(rand.Intn(10000))
	}
	if tick < dis.nextIncident {
		return
	}
	dis.nextIncident = tick + 10000 + int64(rand.Intn(15000))

	players := ctx.GetPlayers()
	if len(players) < 2 {
		return
	}

	// Pick two factions
	var validPlayers []*entities.Player
	for _, p := range players {
		if p != nil {
			validPlayers = append(validPlayers, p)
		}
	}
	if len(validPlayers) < 2 {
		return
	}

	a := validPlayers[rand.Intn(len(validPlayers))]
	b := validPlayers[rand.Intn(len(validPlayers))]
	for b.Name == a.Name {
		b = validPlayers[rand.Intn(len(validPlayers))]
	}

	incidentType := rand.Intn(5)
	switch incidentType {
	case 0: // Border dispute
		a.Credits -= 500
		b.Credits -= 500
		if a.Credits < 0 { a.Credits = 0 }
		if b.Credits < 0 { b.Credits = 0 }
		game.LogEvent("event", "",
			fmt.Sprintf("🏴 Border dispute between %s and %s! Both spend 500cr on diplomatic posturing. First to colonize the contested world wins!",
				a.Name, b.Name))

	case 1: // Spy scandal
		dm.DegradeRelation(a.Name, b.Name)
		a.Credits -= 2000
		if a.Credits < 0 { a.Credits = 0 }
		game.LogEvent("event", b.Name,
			fmt.Sprintf("🕵️ Spy scandal! %s agent caught in %s territory! Relations damaged, %s pays 2000cr reparations",
				a.Name, b.Name, a.Name))

	case 2: // Trade dispute
		game.LogEvent("event", "",
			fmt.Sprintf("📜 Trade dispute: %s accuses %s of unfair trade practices! Tensions rising",
				a.Name, b.Name))

	case 3: // Cultural misunderstanding
		dm.DegradeRelation(a.Name, b.Name)
		game.LogEvent("event", "",
			fmt.Sprintf("😤 Cultural misunderstanding between %s and %s! Relations strained. A diplomatic gift of 1000cr could smooth things over",
				a.Name, b.Name))

	case 4: // Refugee incident
		game.LogEvent("event", b.Name,
			fmt.Sprintf("🚶 Refugee incident: %s citizens flooding into %s territory. Population boost but happiness strain!",
				a.Name, b.Name))
	}
}
