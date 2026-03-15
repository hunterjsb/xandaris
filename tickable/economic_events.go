package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&EconomicEventSystem{
		BaseSystem: NewBaseSystem("EconomicEvents", 50),
	})
}

// EconomicEventSystem generates random economic events that create supply shocks,
// trade opportunities, and narrative tension.
type EconomicEventSystem struct {
	*BaseSystem
	lastEventTick int64
}

// EventType categorizes the economic event.
type EventType int

const (
	EventBoom      EventType = iota // Resource deposit surge
	EventBust                       // Resource deposit depletion
	EventDemand                     // Population demand spike
	EventWindfall                   // Credit bonus from trade
	EventShortage                   // Resource destroyed by accident
)

type economicEvent struct {
	Type     EventType
	Name     string
	Resource string
	Amount   int
	Message  string
}

func (ees *EconomicEventSystem) OnTick(tick int64) {
	// Events fire roughly every 500 ticks (~50 seconds at 1x speed)
	if tick%500 != 0 || tick < 500 {
		return
	}

	// 30% chance per check — keeps events unpredictable
	if rand.Intn(100) > 30 {
		return
	}

	ctx := ees.GetContext()
	if ctx == nil {
		return
	}

	playersIface := ctx.GetPlayers()
	if playersIface == nil {
		return
	}
	players, ok := playersIface.([]*entities.Player)
	if !ok {
		return
	}

	logger, _ := ctx.GetGame().(EventLogger)

	// Pick a random player with at least one planet
	var candidates []*entities.Player
	for _, p := range players {
		if p != nil && len(p.OwnedPlanets) > 0 {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return
	}
	player := candidates[rand.Intn(len(candidates))]
	planet := player.OwnedPlanets[rand.Intn(len(player.OwnedPlanets))]
	if planet == nil {
		return
	}

	event := ees.generateEvent(planet, player)
	if event == nil {
		return
	}

	ees.applyEvent(event, planet, player)
	ees.lastEventTick = tick

	fmt.Printf("[Event] %s\n", event.Message)
	if logger != nil {
		logger.LogEvent("event", player.Name, event.Message)
	}
}

func (ees *EconomicEventSystem) generateEvent(planet *entities.Planet, player *entities.Player) *economicEvent {
	resources := []string{"Iron", "Water", "Oil", "Rare Metals", "Helium-3", "Fuel", "Electronics"}
	res := resources[rand.Intn(len(resources))]

	roll := rand.Intn(100)
	switch {
	case roll < 25:
		// Boom: resource deposit found — bonus resources
		amount := 50 + rand.Intn(150)
		return &economicEvent{
			Type:     EventBoom,
			Name:     "Resource Surge",
			Resource: res,
			Amount:   amount,
			Message:  fmt.Sprintf("Survey teams on %s discovered a %s vein! +%d %s", planet.Name, res, amount, res),
		}
	case roll < 45:
		// Shortage: industrial accident destroys some stock
		stored := planet.GetStoredAmount(res)
		if stored < 20 {
			return nil // Nothing to lose
		}
		amount := stored / 4 // Lose 25% of stock
		if amount < 10 {
			amount = 10
		}
		return &economicEvent{
			Type:     EventShortage,
			Name:     "Storage Accident",
			Resource: res,
			Amount:   amount,
			Message:  fmt.Sprintf("Storage malfunction on %s! Lost %d %s", planet.Name, amount, res),
		}
	case roll < 65:
		// Windfall: trade caravan pays bonus credits
		amount := 200 + rand.Intn(800)
		return &economicEvent{
			Type:     EventWindfall,
			Name:     "Trade Windfall",
			Amount:   amount,
			Message:  fmt.Sprintf("Passing trade caravan paid %s a %d credit bonus at %s", player.Name, amount, planet.Name),
		}
	case roll < 80:
		// Demand spike: population growth burst
		if planet.Population < 500 {
			return nil
		}
		amount := int(planet.Population / 20) // 5% population growth
		return &economicEvent{
			Type:     EventDemand,
			Name:     "Immigration Wave",
			Amount:   amount,
			Message:  fmt.Sprintf("Immigration wave at %s! +%d settlers for %s", planet.Name, amount, player.Name),
		}
	default:
		// Deposit enrichment: boost a resource deposit's abundance
		for _, resEntity := range planet.Resources {
			if res, ok := resEntity.(*entities.Resource); ok && res.Abundance < 60 {
				amount := 5 + rand.Intn(15)
				return &economicEvent{
					Type:     EventBoom,
					Name:     "Deposit Enrichment",
					Resource: res.ResourceType,
					Amount:   amount,
					Message:  fmt.Sprintf("Geological survey at %s found deeper %s deposits! Abundance +%d", planet.Name, res.ResourceType, amount),
				}
			}
		}
		return nil
	}
}

func (ees *EconomicEventSystem) applyEvent(event *economicEvent, planet *entities.Planet, player *entities.Player) {
	switch event.Type {
	case EventBoom:
		if event.Name == "Deposit Enrichment" {
			// Boost deposit abundance
			for _, resEntity := range planet.Resources {
				if res, ok := resEntity.(*entities.Resource); ok && res.ResourceType == event.Resource {
					res.Abundance += event.Amount
					if res.Abundance > 100 {
						res.Abundance = 100
					}
					break
				}
			}
		} else {
			// Add bonus resources to storage
			planet.AddStoredResource(event.Resource, event.Amount)
		}
	case EventShortage:
		planet.RemoveStoredResource(event.Resource, event.Amount)
	case EventWindfall:
		player.Credits += event.Amount
	case EventDemand:
		planet.Population += int64(event.Amount)
		cap := planet.GetTotalPopulationCapacity()
		if planet.Population > cap {
			planet.Population = cap
		}
	}
}
