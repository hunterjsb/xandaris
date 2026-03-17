package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SpaceWeatherSystem{
		BaseSystem: NewBaseSystem("SpaceWeather", 80),
	})
}

// SpaceWeatherSystem generates system-level environmental effects
// that modify production, trade, and movement in affected systems.
//
// Weather types (one per system, lasts 3000-8000 ticks):
//   Solar Wind:   +20% mine production, +10% solar power
//   Cosmic Ray Burst: -10% population growth, +20% Electronics output
//   Stellar Quiet: -30% solar power, but +15% ship fuel efficiency
//   Ion Cloud:    Ships travel 20% slower, but +30% Helium-3 production
//   Magnetic Lull: All sensors enhanced — exploration discoveries +50%
//
// Weather shifts naturally. Factions can't control it but can plan
// around it. Knowing the weather forecast helps optimize trade routes
// and production schedules.
type SpaceWeatherSystem struct {
	*BaseSystem
	weather   map[int]*SystemWeather // systemID → active weather
	nextShift int64
}

// SystemWeather represents weather in a star system.
type SystemWeather struct {
	SystemID  int
	SysName   string
	Type      string
	Effect    string
	TicksLeft int
}

var weatherTypes = []struct {
	name   string
	effect string
}{
	{"Solar Wind", "+20% mining, +10% power"},
	{"Cosmic Ray Burst", "-10% pop growth, +20% Electronics"},
	{"Stellar Quiet", "-30% solar power, +15% fuel efficiency"},
	{"Ion Cloud", "-20% ship speed, +30% Helium-3"},
	{"Magnetic Lull", "+50% exploration discoveries"},
}

func (sws *SpaceWeatherSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := sws.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if sws.weather == nil {
		sws.weather = make(map[int]*SystemWeather)
	}

	if sws.nextShift == 0 {
		sws.nextShift = tick + 3000 + int64(rand.Intn(5000))
	}

	systems := game.GetSystems()

	// Decay existing weather
	for sysID, w := range sws.weather {
		w.TicksLeft -= 500
		if w.TicksLeft <= 0 {
			delete(sws.weather, sysID)
			game.LogEvent("event", "",
				fmt.Sprintf("🌤️ Weather in %s clearing — %s has passed",
					w.SysName, w.Type))
		}
	}

	// Apply weather effects
	for _, w := range sws.weather {
		sws.applyWeatherEffects(w, systems, game)
	}

	// Generate new weather
	if tick >= sws.nextShift {
		sws.nextShift = tick + 5000 + int64(rand.Intn(8000))

		// Max 3 systems with active weather
		if len(sws.weather) >= 3 {
			return
		}

		sws.generateWeather(game, systems)
	}
}

func (sws *SpaceWeatherSystem) applyWeatherEffects(w *SystemWeather, systems []*entities.System, game GameProvider) {
	for _, sys := range systems {
		if sys.ID != w.SystemID {
			continue
		}

		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			switch w.Type {
			case "Solar Wind":
				// Bonus production applied via small resource injection
				if rand.Intn(5) == 0 {
					planet.AddStoredResource(entities.ResIron, 2)
				}
			case "Cosmic Ray Burst":
				// Electronics bonus
				if rand.Intn(8) == 0 {
					planet.AddStoredResource(entities.ResElectronics, 1)
				}
			case "Ion Cloud":
				// Helium-3 bonus
				if rand.Intn(5) == 0 {
					planet.AddStoredResource(entities.ResHelium3, 1)
				}
			}
		}
		break
	}
}

func (sws *SpaceWeatherSystem) generateWeather(game GameProvider, systems []*entities.System) {
	if len(systems) == 0 {
		return
	}

	// Pick a system without active weather
	sys := systems[rand.Intn(len(systems))]
	if _, exists := sws.weather[sys.ID]; exists {
		return
	}

	wt := weatherTypes[rand.Intn(len(weatherTypes))]
	duration := 3000 + rand.Intn(5000)

	sws.weather[sys.ID] = &SystemWeather{
		SystemID:  sys.ID,
		SysName:   sys.Name,
		Type:      wt.name,
		Effect:    wt.effect,
		TicksLeft: duration,
	}

	emoji := "🌊"
	switch wt.name {
	case "Cosmic Ray Burst":
		emoji = "☢️"
	case "Stellar Quiet":
		emoji = "🌑"
	case "Ion Cloud":
		emoji = "☁️"
	case "Magnetic Lull":
		emoji = "🧭"
	}

	game.LogEvent("event", "",
		fmt.Sprintf("%s Space weather in %s: %s! Effect: %s (~%d min)",
			emoji, sys.Name, wt.name, wt.effect, duration/600))
}

// GetWeather returns the active weather for a system.
func (sws *SpaceWeatherSystem) GetWeather(systemID int) *SystemWeather {
	if sws.weather == nil {
		return nil
	}
	return sws.weather[systemID]
}

// GetAllWeather returns all active weather events.
func (sws *SpaceWeatherSystem) GetAllWeather() []*SystemWeather {
	var result []*SystemWeather
	for _, w := range sws.weather {
		result = append(result, w)
	}
	return result
}
