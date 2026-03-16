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

	players := ctx.GetPlayers()
	if players == nil {
		return
	}

	game := ctx.GetGame()

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
	if game != nil {
		game.LogEvent("event", player.Name, event.Message)
	}
}

// generateEvent creates events that respond to actual planet conditions.
// Happy planets attract immigrants; stressed planets have accidents;
// trading hubs attract caravans; explored systems find deposits.
func (ees *EconomicEventSystem) generateEvent(planet *entities.Planet, player *entities.Player) *economicEvent {
	resources := []string{entities.ResIron, entities.ResWater, entities.ResOil, entities.ResRareMetals, entities.ResHelium3, entities.ResFuel, entities.ResElectronics}
	res := resources[rand.Intn(len(resources))]

	// Weight events based on planet conditions
	happiness := planet.Happiness
	powerRatio := planet.GetPowerRatio()
	hasTradingPost := planet.HasOperationalBuilding(entities.BuildingTradingPost)

	roll := rand.Intn(100)
	switch {
	case roll < 20:
		// Resource discovery — more likely on planets with low resource diversity
		if len(planet.Resources) >= 5 {
			return nil // already resource-rich
		}
		amount := 50 + rand.Intn(150)
		return &economicEvent{
			Type:     EventBoom,
			Name:     "Resource Surge",
			Resource: res,
			Amount:   amount,
			Message:  fmt.Sprintf("Survey teams on %s discovered a %s vein! +%d %s", planet.Name, res, amount, res),
		}

	case roll < 40:
		// Industrial accident — more likely with low power or low happiness
		accidentChance := 80 // base 80% chance to actually trigger
		if powerRatio > 0.8 {
			accidentChance -= 40 // well-powered planets are safer
		}
		if happiness > 0.7 {
			accidentChance -= 30 // happy workers make fewer mistakes
		}
		if rand.Intn(100) >= accidentChance {
			return nil // good conditions prevented the accident
		}
		stored := planet.GetStoredAmount(res)
		if stored < 20 {
			return nil
		}
		amount := stored / 4
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

	case roll < 60:
		// Trade caravan — only at planets with Trading Posts
		if !hasTradingPost {
			return nil
		}
		// Windfall scales with planet population (bigger market = bigger payoff)
		base := 200 + int(planet.Population/50)
		amount := base + rand.Intn(base)
		return &economicEvent{
			Type:     EventWindfall,
			Name:     "Trade Windfall",
			Amount:   amount,
			Message:  fmt.Sprintf("Passing trade caravan paid %s a %d credit bonus at %s", player.Name, amount, planet.Name),
		}

	case roll < 80:
		// Immigration wave — only on happy, prosperous planets
		if planet.Population < 500 || happiness < 0.5 {
			return nil // unhappy planets don't attract settlers
		}
		// Immigration scales with happiness: more happy = more immigrants
		growthMult := happiness * 2.0 // 0.5 → 1.0x, 1.0 → 2.0x
		amount := int(float64(planet.Population/20) * growthMult)
		if amount < 10 {
			amount = 10
		}
		return &economicEvent{
			Type:     EventDemand,
			Name:     "Immigration Wave",
			Amount:   amount,
			Message:  fmt.Sprintf("Immigration wave at %s! +%d settlers for %s (happiness %.0f%%)", planet.Name, amount, player.Name, happiness*100),
		}

	default:
		// Deposit enrichment — more likely on planets with depleted deposits
		for _, resEntity := range planet.Resources {
			if r, ok := resEntity.(*entities.Resource); ok && r.Abundance < 40 {
				amount := 5 + rand.Intn(15)
				return &economicEvent{
					Type:     EventBoom,
					Name:     "Deposit Enrichment",
					Resource: r.ResourceType,
					Amount:   amount,
					Message:  fmt.Sprintf("Geological survey at %s found deeper %s deposits! Abundance +%d", planet.Name, r.ResourceType, amount),
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
