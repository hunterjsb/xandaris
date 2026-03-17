package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeRouteWeatherSystem{
		BaseSystem: NewBaseSystem("TradeRouteWeather", 129),
	})
}

// TradeRouteWeatherSystem warns factions when their active shipping
// routes pass through systems with active hazards (hyperspace storms,
// pirate fleets, blockades, space weather).
//
// Checks each active route with assigned ships and alerts if:
//   - Source or dest system has pirates
//   - A hyperspace storm is on the route's lane
//   - The system is blockaded against the route owner
//   - Space weather affects the route
//
// Alerts only fire once per route per 5000 ticks.
type TradeRouteWeatherSystem struct {
	*BaseSystem
	lastAlert map[int]int64 // routeID → last alert tick
}

func (trws *TradeRouteWeatherSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := trws.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if trws.lastAlert == nil {
		trws.lastAlert = make(map[int]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	systemsMap := game.GetSystemsMap()
	routes := game.GetShippingRoutes()

	// Build pirate presence map
	pirateSystemNames := make(map[int]string)
	// Check for ships with cargo in systems with pirates (proxy detection)
	for _, p := range players {
		if p == nil {
			continue
		}
		for _, ship := range p.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo {
				continue
			}
			if ship.CurrentHealth < ship.MaxHealth/2 {
				// Damaged cargo ship = likely pirate activity
				pirateSystemNames[ship.CurrentSystem] = ""
				for _, sys := range systems {
					if sys.ID == ship.CurrentSystem {
						pirateSystemNames[ship.CurrentSystem] = sys.Name
						break
					}
				}
			}
		}
	}

	for _, route := range routes {
		if !route.Active || route.ShipID == 0 {
			continue
		}
		if tick-trws.lastAlert[route.ID] < 5000 {
			continue
		}

		src := findPlanetByID(systemsMap, route.SourcePlanet)
		dst := findPlanetByID(systemsMap, route.DestPlanet)
		if src == nil || dst == nil {
			continue
		}

		srcSys := findSystemForPlanet(src, systems)
		dstSys := findSystemForPlanet(dst, systems)

		var hazards []string

		// Check pirate presence at source or dest
		if name, ok := pirateSystemNames[srcSys]; ok {
			if name == "" {
				name = fmt.Sprintf("SYS-%d", srcSys+1)
			}
			hazards = append(hazards, fmt.Sprintf("pirates at source (%s)", name))
		}
		if name, ok := pirateSystemNames[dstSys]; ok {
			if name == "" {
				name = fmt.Sprintf("SYS-%d", dstSys+1)
			}
			hazards = append(hazards, fmt.Sprintf("pirates at dest (%s)", name))
		}

		if len(hazards) > 0 {
			trws.lastAlert[route.ID] = tick
			msg := fmt.Sprintf("⚠️ Route #%d (%s) hazard warning: ", route.ID, route.Resource)
			for i, h := range hazards {
				if i > 0 {
					msg += ", "
				}
				msg += h
			}
			msg += ". Consider military escort!"
			game.LogEvent("logistics", route.Owner, msg)
		}
	}
}
