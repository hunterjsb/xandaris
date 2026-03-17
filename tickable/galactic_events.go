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
		ges.tradeBoom(game, players, systems)
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
	shielded := 0
	for _, e := range sys.Entities {
		if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
			fuel := planet.GetStoredAmount(entities.ResFuel)
			if fuel <= 0 {
				continue
			}
			// Check for shield
			hasShield := false
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingPlanetShield && b.IsOperational {
					hasShield = true
					break
				}
			}
			if hasShield {
				shielded++
				continue // shield protects against solar flare
			}
			drain := fuel / 2
			planet.RemoveStoredResource(entities.ResFuel, drain)
			affected++
		}
	}
	if affected > 0 || shielded > 0 {
		msg := fmt.Sprintf("☀️ Solar flare in %s! %d planets affected", sys.Name, affected)
		if shielded > 0 {
			msg += fmt.Sprintf(" (%d shielded)", shielded)
		}
		game.LogEvent("event", "", msg)
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

// tradeBoom: demand surge rewards factions with stockpiles
func (ges *GalacticEventSystem) tradeBoom(game GameProvider, players []*entities.Player, systems []*entities.System) {
	market := game.GetMarketEngine()
	if market == nil {
		return
	}
	resources := []string{entities.ResOil, entities.ResHelium3, entities.ResElectronics,
		entities.ResRareMetals, entities.ResWater}
	res := resources[rand.Intn(len(resources))]

	// Spike demand
	market.AddTradeVolume(res, 500+rand.Intn(1000), true)

	// Reward factions who have this resource stocked — galactic buyers pay premium
	price := market.GetSellPrice(res)
	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}
			stored := planet.GetStoredAmount(res)
			if stored <= 50 {
				continue
			}
			// Sell up to 50 units at 2x price to "galactic demand"
			sellQty := 50
			if sellQty > stored-20 {
				sellQty = stored - 20 // keep 20 buffer
			}
			if sellQty <= 0 {
				continue
			}
			credits := int(price * 2.0 * float64(sellQty))
			planet.RemoveStoredResource(res, sellQty)
			for _, p := range players {
				if p != nil && p.Name == planet.Owner {
					p.Credits += credits
					break
				}
			}
		}
	}

	game.LogEvent("event", "",
		fmt.Sprintf("📈 Trade boom! Galactic demand for %s surges — stockpilers rewarded at 2x price!", res))
}

// asteroidImpact: a planet loses some stored resources (reduced by shield)
func (ges *GalacticEventSystem) asteroidImpact(game GameProvider, systems []*entities.System) {
	planet := randomOwnedPlanet(systems)
	if planet == nil {
		return
	}

	// Check for Planetary Shield — reduces damage
	shieldLevel := 0
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingPlanetShield && b.IsOperational {
			shieldLevel = b.Level
			break
		}
	}

	resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
		entities.ResRareMetals}
	res := resources[rand.Intn(len(resources))]
	stored := planet.GetStoredAmount(res)
	if stored <= 10 {
		return
	}

	// Base loss 20-40%, reduced 15% per shield level (L5 = 75% reduction)
	baseLoss := 0.2 + rand.Float64()*0.2
	shieldReduction := float64(shieldLevel) * 0.15
	if shieldReduction > 0.90 {
		shieldReduction = 0.90
	}
	actualLoss := baseLoss * (1.0 - shieldReduction)
	loss := int(float64(stored) * actualLoss)
	if loss <= 0 {
		game.LogEvent("event", planet.Owner,
			fmt.Sprintf("☄️ Asteroid deflected by %s's Planetary Shield!", planet.Name))
		return
	}

	planet.RemoveStoredResource(res, loss)
	if shieldLevel > 0 {
		game.LogEvent("event", planet.Owner,
			fmt.Sprintf("☄️ Asteroid hit %s! Shield reduced damage — lost %d %s (%.0f%% blocked)",
				planet.Name, loss, res, shieldReduction*100))
	} else {
		game.LogEvent("event", planet.Owner,
			fmt.Sprintf("☄️ Asteroid impact on %s! Lost %d %s (build a Planetary Shield!)",
				planet.Name, loss, res))
	}
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
