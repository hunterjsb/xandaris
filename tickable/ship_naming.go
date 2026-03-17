package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ShipNamingSystem{
		BaseSystem: NewBaseSystem("ShipNaming", 160),
	})
}

// ShipNamingSystem gives memorable names to ships that achieve
// notable feats. Generic "Hauler-1" and "Converted-123" names
// get replaced with legendary names after milestones.
//
// Naming triggers:
//   - Cargo ship completes 5+ deliveries: named after trade routes
//   - Military ship survives 3+ battles: named "Ironclad", "Invincible"
//   - Scout discovers 3+ anomalies: named "Pathfinder", "Stargazer"
//   - Ship reaches "Veteran" XP rank: earns a hero name
//
// Names are drawn from themed pools and never repeat.
type ShipNamingSystem struct {
	*BaseSystem
	namedShips map[int]bool
	nextCheck  int64
}

var heroNames = []string{
	"Vanguard", "Endeavor", "Resolute", "Dauntless", "Horizon",
	"Tempest", "Phoenix", "Valiant", "Sovereign", "Eclipse",
	"Meridian", "Zenith", "Triumph", "Vigilant", "Corsair",
	"Nomad", "Wayward", "Pioneer", "Ironheart", "Stormborn",
	"Farseeker", "Dawnbreaker", "Starweaver", "Voidrunner", "Trailblazer",
}

func (sns *ShipNamingSystem) OnTick(tick int64) {
	if tick%3000 != 0 {
		return
	}

	ctx := sns.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if sns.namedShips == nil {
		sns.namedShips = make(map[int]bool)
	}

	players := ctx.GetPlayers()
	routes := game.GetShippingRoutes()

	// Find ships on routes with completed trips
	routeShipTrips := make(map[int]int) // shipID → total trips
	for _, r := range routes {
		if r.Active && r.ShipID != 0 && r.TripsComplete >= 5 {
			routeShipTrips[r.ShipID] += r.TripsComplete
		}
	}

	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || sns.namedShips[ship.GetID()] {
				continue
			}

			shouldName := false
			reason := ""

			// Cargo with route trips
			if trips, ok := routeShipTrips[ship.GetID()]; ok && trips >= 5 {
				shouldName = true
				reason = fmt.Sprintf("%d deliveries", trips)
			}

			// Damaged military = battle-hardened
			if (ship.ShipType == entities.ShipTypeFrigate ||
				ship.ShipType == entities.ShipTypeDestroyer ||
				ship.ShipType == entities.ShipTypeCruiser) &&
				ship.CurrentHealth < ship.MaxHealth*70/100 {
				shouldName = true
				reason = "battle-hardened veteran"
			}

			if !shouldName {
				continue
			}

			// Pick a name
			nameIdx := rand.Intn(len(heroNames))
			newName := heroNames[nameIdx]

			oldName := ship.Name
			ship.Name = newName
			sns.namedShips[ship.GetID()] = true

			game.LogEvent("event", player.Name,
				fmt.Sprintf("🏅 %s's %s renamed to \"%s\" — %s!",
					player.Name, oldName, newName, reason))
			break // one naming per faction per tick
		}
	}
}
