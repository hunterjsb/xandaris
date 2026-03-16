package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&StationSystem{
		BaseSystem: NewBaseSystem("Stations", 26),
	})
}

// StationSystem simulates NPC stations: refueling ships, collecting docking fees,
// and growing/shrinking population based on traffic. Stations act as neutral
// trade hubs and service points that any faction can use.
type StationSystem struct {
	*BaseSystem
}

func (ss *StationSystem) OnTick(tick int64) {
	// Run every 50 ticks (~5 seconds)
	if tick%50 != 0 {
		return
	}

	ctx := ss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()

	for _, sys := range game.GetSystems() {
		// Find stations in this system
		var stations []*entities.Station
		for _, e := range sys.Entities {
			if station, ok := e.(*entities.Station); ok {
				stations = append(stations, station)
			}
		}
		if len(stations) == 0 {
			continue
		}

		// Find ships in this system (from players' owned ships)
		for _, station := range stations {
			ss.processStation(station, sys, players, game, tick)
		}
	}
}

func (ss *StationSystem) processStation(station *entities.Station, sys *entities.System, players []*entities.Player, game GameProvider, tick int64) {
	if station.IsHostile() {
		return
	}

	// Auto-refuel ships near this station (if station offers Fuel service)
	hasFuelService := false
	for _, svc := range station.Services {
		if svc == "Fuel" {
			hasFuelService = true
			break
		}
	}

	if hasFuelService {
		for _, player := range players {
			if player == nil {
				continue
			}
			for _, ship := range player.OwnedShips {
				if ship == nil || ship.CurrentSystem != sys.ID {
					continue
				}
				if ship.Status == entities.ShipStatusMoving {
					continue
				}
				if ship.CurrentFuel >= ship.MaxFuel {
					continue
				}

				// Refuel up to 10 units per interval (costs credits)
				needed := ship.MaxFuel - ship.CurrentFuel
				refuelAmt := needed
				if refuelAmt > 10 {
					refuelAmt = 10
				}

				// Station fuel costs 3 credits per unit (more expensive than planet fuel)
				cost := refuelAmt * 3
				if player.Credits >= cost {
					player.Credits -= cost
					ship.Refuel(refuelAmt)
					// Station earns revenue (grows population from economic activity)
					station.CurrentPop += refuelAmt / 5
					if station.CurrentPop > station.Capacity {
						station.CurrentPop = station.Capacity
					}
				}
			}
		}
	}

	// Station population slowly decays if no activity (attrition)
	if tick%500 == 0 && station.CurrentPop > 100 {
		station.CurrentPop -= station.CurrentPop / 50 // lose 2%
		if station.CurrentPop < 100 {
			station.CurrentPop = 100 // minimum skeleton crew
		}
	}

	// Trading stations periodically log market activity
	if station.StationType == "Trading" && tick%1000 == 0 && station.CurrentPop > 200 {
		game.LogEvent("trade", station.Owner,
			fmt.Sprintf("%s reports active commerce (%d residents)", station.Name, station.CurrentPop))
	}
}
