package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticEventSystem{
		BaseSystem: NewBaseSystem("GalacticEvents", 32),
	})
}

// GalacticEventSystem generates random events that affect the galaxy.
// Events create urgency, opportunity, and emergent gameplay:
//
// - Resource discovery: a planet gains a new resource deposit
// - Solar flare: a system loses power temporarily
// - Pirate raid: a cargo ship loses some cargo
// - Population boom: a happy planet gets bonus population
// - Trade boom: a resource temporarily spikes in price
// - Asteroid impact: a planet loses some stored resources
// - Refugee wave: unclaimed planet gets free colonists
//
// Events fire roughly every 2000 ticks (~3.3 minutes) with randomness.
type GalacticEventSystem struct {
	*BaseSystem
	nextEvent int64
}

func (ges *GalacticEventSystem) OnTick(tick int64) {
	if ges.nextEvent == 0 {
		ges.nextEvent = tick + 1000 + int64(rand.Intn(2000))
	}
	if tick < ges.nextEvent {
		return
	}
	ges.nextEvent = tick + 1500 + int64(rand.Intn(2000))

	ctx := ges.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Pick a random event
	eventType := rand.Intn(7)
	switch eventType {
	case 0:
		ges.resourceDiscovery(game, systems)
	case 1:
		ges.solarFlare(game, systems)
	case 2:
		ges.pirateRaid(game, players)
	case 3:
		ges.populationBoom(game, systems)
	case 4:
		ges.tradeBoom(game)
	case 5:
		ges.asteroidImpact(game, systems)
	case 6:
		ges.refugeeWave(game, systems)
	}
}

// resourceDiscovery: a random owned planet gains abundance on an existing deposit
func (ges *GalacticEventSystem) resourceDiscovery(game GameProvider, systems []*entities.System) {
	planet := randomOwnedPlanet(systems)
	if planet == nil || len(planet.Resources) == 0 {
		return
	}
	// Boost a random resource's abundance
	idx := rand.Intn(len(planet.Resources))
	if res, ok := planet.Resources[idx].(*entities.Resource); ok {
		bonus := 10 + rand.Intn(20)
		res.Abundance += bonus
		game.LogEvent("event", planet.Owner,
			fmt.Sprintf("🔍 Resource discovery on %s! %s abundance increased by %d",
				planet.Name, res.ResourceType, bonus))
	}
}

// solarFlare: a random system's power generation is disrupted (stored fuel depleted)
func (ges *GalacticEventSystem) solarFlare(game GameProvider, systems []*entities.System) {
	if len(systems) == 0 {
		return
	}
	sys := systems[rand.Intn(len(systems))]
	affected := 0
	for _, e := range sys.Entities {
		if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
			// Drain 50% of stored Fuel
			fuel := planet.GetStoredAmount(entities.ResFuel)
			if fuel > 0 {
				drain := fuel / 2
				planet.RemoveStoredResource(entities.ResFuel, drain)
				affected++
			}
		}
	}
	if affected > 0 {
		game.LogEvent("event", "",
			fmt.Sprintf("☀️ Solar flare in %s! Fuel reserves depleted across %d planets",
				sys.Name, affected))
	}
}

// pirateRaid: a random cargo ship loses some cargo
func (ges *GalacticEventSystem) pirateRaid(game GameProvider, players []*entities.Player) {
	for _, p := range players {
		if p == nil {
			continue
		}
		for _, ship := range p.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo || ship.GetTotalCargo() == 0 {
				continue
			}
			// 20% chance per loaded cargo ship
			if rand.Intn(5) != 0 {
				continue
			}
			// Lose 10-30% of cargo
			lossRate := 0.1 + rand.Float64()*0.2
			for res, amt := range ship.CargoHold {
				loss := int(float64(amt) * lossRate)
				if loss > 0 {
					ship.CargoHold[res] -= loss
					if ship.CargoHold[res] <= 0 {
						delete(ship.CargoHold, res)
					}
				}
			}
			game.LogEvent("event", p.Name,
				fmt.Sprintf("🏴‍☠️ Pirates raided %s! Lost %.0f%% of cargo",
					ship.Name, lossRate*100))
			return // only one raid per event
		}
	}
}

// populationBoom: a happy planet gets bonus population
func (ges *GalacticEventSystem) populationBoom(game GameProvider, systems []*entities.System) {
	planet := randomOwnedPlanet(systems)
	if planet == nil || planet.Happiness < 0.6 {
		return // only happy planets get booms
	}
	cap := planet.GetTotalPopulationCapacity()
	if cap <= 0 || planet.Population >= cap {
		return
	}
	bonus := int64(500 + rand.Intn(2000))
	if planet.Population+bonus > cap {
		bonus = cap - planet.Population
	}
	planet.Population += bonus
	game.LogEvent("event", planet.Owner,
		fmt.Sprintf("👶 Population boom on %s! %d new citizens arrived",
			planet.Name, bonus))
}

// tradeBoom: temporarily boost a resource's market price
func (ges *GalacticEventSystem) tradeBoom(game GameProvider) {
	market := game.GetMarketEngine()
	if market == nil {
		return
	}
	resources := []string{entities.ResOil, entities.ResHelium3, entities.ResElectronics,
		entities.ResRareMetals, entities.ResWater}
	res := resources[rand.Intn(len(resources))]
	// Add artificial demand to spike the price
	market.AddTradeVolume(res, 500+rand.Intn(1000), true)
	game.LogEvent("event", "",
		fmt.Sprintf("📈 Trade boom! Demand for %s surges across the galaxy", res))
}

// asteroidImpact: a planet loses some stored resources
func (ges *GalacticEventSystem) asteroidImpact(game GameProvider, systems []*entities.System) {
	planet := randomOwnedPlanet(systems)
	if planet == nil {
		return
	}
	// Destroy 20-40% of a random stored resource
	resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
		entities.ResRareMetals}
	res := resources[rand.Intn(len(resources))]
	stored := planet.GetStoredAmount(res)
	if stored <= 10 {
		return
	}
	loss := int(float64(stored) * (0.2 + rand.Float64()*0.2))
	planet.RemoveStoredResource(res, loss)
	game.LogEvent("event", planet.Owner,
		fmt.Sprintf("☄️ Asteroid impact on %s! Lost %d %s", planet.Name, loss, res))
}

// refugeeWave: an unclaimed habitable planet gets free colonists
func (ges *GalacticEventSystem) refugeeWave(game GameProvider, systems []*entities.System) {
	// Find an unclaimed habitable planet
	var candidates []*entities.Planet
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if p, ok := e.(*entities.Planet); ok && p.Owner == "" && p.IsHabitable() {
				candidates = append(candidates, p)
			}
		}
	}
	if len(candidates) == 0 {
		return
	}
	planet := candidates[rand.Intn(len(candidates))]
	pop := int64(1000 + rand.Intn(5000))
	planet.Population += pop
	game.LogEvent("event", "",
		fmt.Sprintf("🚀 Refugee wave! %d settlers arrived on unclaimed %s",
			pop, planet.Name))
}

func randomOwnedPlanet(systems []*entities.System) *entities.Planet {
	var owned []*entities.Planet
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if p, ok := e.(*entities.Planet); ok && p.Owner != "" && p.Population > 0 {
				owned = append(owned, p)
			}
		}
	}
	if len(owned) == 0 {
		return nil
	}
	return owned[rand.Intn(len(owned))]
}
