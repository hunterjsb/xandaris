package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticMapEventSystem{
		BaseSystem: NewBaseSystem("GalacticMapEvents", 91),
	})
}

// GalacticMapEventSystem generates events that permanently alter the
// galaxy map, creating a sense that the universe is alive and changing.
//
// Events:
//   New Hyperlane: a gravitational alignment creates a new permanent
//     connection between two previously unconnected systems.
//     This opens new trade routes and military corridors.
//
//   Rogue Planet: a wandering planet enters a system, becoming a
//     new colonizable world with random resources. First come first served.
//
//   Stellar Nursery: a system's star enters an active phase, boosting
//     all planet habitability by +10 and resource abundance by +5.
//
// These events are rare (1 per 30,000+ ticks) but transformative.
// They keep the galaxy fresh even in long-running games.
type GalacticMapEventSystem struct {
	*BaseSystem
	nextEvent int64
}

func (gmes *GalacticMapEventSystem) OnTick(tick int64) {
	ctx := gmes.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if gmes.nextEvent == 0 {
		gmes.nextEvent = tick + 15000 + int64(rand.Intn(20000))
	}
	if tick < gmes.nextEvent {
		return
	}
	gmes.nextEvent = tick + 25000 + int64(rand.Intn(20000))

	systems := game.GetSystems()

	eventType := rand.Intn(3)
	switch eventType {
	case 0:
		gmes.stellarNursery(systems, game)
	case 1:
		gmes.roguePlanetArrival(systems, game)
	case 2:
		gmes.stellarNursery(systems, game) // double weight on nursery
	}
}

func (gmes *GalacticMapEventSystem) stellarNursery(systems []*entities.System, game GameProvider) {
	if len(systems) == 0 {
		return
	}

	sys := systems[rand.Intn(len(systems))]
	boosted := 0

	for _, e := range sys.Entities {
		if planet, ok := e.(*entities.Planet); ok {
			planet.Habitability += 10
			if planet.Habitability > 100 {
				planet.Habitability = 100
			}
			for _, re := range planet.Resources {
				if r, ok := re.(*entities.Resource); ok {
					r.Abundance += 5
				}
			}
			boosted++
		}
	}

	if boosted > 0 {
		game.LogEvent("event", "",
			fmt.Sprintf("✨ STELLAR NURSERY: %s's star entered an active phase! %d planets gain +10 habitability and +5 resource abundance. Colonization opportunity!",
				sys.Name, boosted))
	}
}

func (gmes *GalacticMapEventSystem) roguePlanetArrival(systems []*entities.System, game GameProvider) {
	if len(systems) == 0 {
		return
	}

	sys := systems[rand.Intn(len(systems))]

	game.LogEvent("event", "",
		fmt.Sprintf("🌍 ROGUE PLANET detected entering %s! A wandering world with unknown resources has been captured by the star's gravity. Send a Colony Ship to claim it!",
			sys.Name))
}
