package tickable

import (
	"fmt"
	"image/color"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SpaceStationSystem{
		BaseSystem: NewBaseSystem("SpaceStations", 39),
	})
}

// SpaceStationSystem manages orbital space stations.
// Stations are system-level structures (not planet-level) that provide
// bonuses to ALL ships and planets in their system.
//
// Station types (detected from Station entities in systems):
//   Refueling Station: auto-refuels all ships in system (+10 fuel/interval)
//   Defense Platform:  +50% defense vs pirate raids in this system
//   Trade Hub:         +25% trade revenue for all TPs in system
//   Sensor Array:      auto-explores system (no scout needed)
//
// Stations are built by factions and benefit everyone in the system,
// making them cooperative infrastructure investments.
type SpaceStationSystem struct {
	*BaseSystem
}

func (sss *SpaceStationSystem) OnTick(tick int64) {
	if tick%50 != 0 {
		return
	}

	ctx := sss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		// Check for stations in this system
		for _, e := range sys.Entities {
			station, ok := e.(*entities.Station)
			if !ok || station.Owner == "" {
				continue
			}

			switch station.StationType {
			case "Refueling":
				sss.processRefueling(sys, players)
			case "Defense":
				// Defense handled by pirate system checking for stations
			case "Trade":
				sss.processTradeHub(sys, players, game)
			}
		}
	}
}

// processRefueling auto-refuels all ships in a system with a refueling station
func (sss *SpaceStationSystem) processRefueling(sys *entities.System, players []*entities.Player) {
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.CurrentSystem != sys.ID {
				continue
			}
			if ship.CurrentFuel < ship.MaxFuel {
				refuel := 10
				if ship.CurrentFuel+refuel > ship.MaxFuel {
					refuel = ship.MaxFuel - ship.CurrentFuel
				}
				ship.CurrentFuel += refuel
			}
		}
	}
}

// processTradeHub gives bonus credits to TP owners in the system
func (sss *SpaceStationSystem) processTradeHub(sys *entities.System, players []*entities.Player, game GameProvider) {
	playerByName := make(map[string]*entities.Player)
	for _, p := range players {
		if p != nil {
			playerByName[p.Name] = p
		}
	}

	for _, e := range sys.Entities {
		planet, ok := e.(*entities.Planet)
		if !ok || planet.Owner == "" {
			continue
		}

		for _, be := range planet.Buildings {
			b, ok := be.(*entities.Building)
			if !ok || b.BuildingType != entities.BuildingTradingPost || !b.IsOperational {
				continue
			}
			// +25% revenue = +5 credits per TP level
			bonus := 5 * b.Level // Building.Level, not Station.Level
			if player := playerByName[planet.Owner]; player != nil {
				player.Credits += bonus
			}
		}
	}
}

// BuildStation is a helper to create a station entity in a system.
func BuildStation(sys *entities.System, stationType, owner string, systemID int) {
	station := entities.NewStation(
		systemID*10000+len(sys.Entities),
		fmt.Sprintf("%s %s Station", owner, stationType),
		stationType,
		30+float64(len(sys.Entities)),
		0,
		color.RGBA{200, 200, 255, 255},
	)
	station.Owner = owner
	sys.Entities = append(sys.Entities, station)
	fmt.Printf("[Station] %s built %s station in %s\n", owner, stationType, sys.Name)
}
