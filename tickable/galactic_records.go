package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticRecordsSystem{
		BaseSystem: NewBaseSystem("GalacticRecords", 81),
	})
}

// GalacticRecordsSystem tracks and announces galactic "firsts" and
// records. These are permanent achievements that create emergent
// narrative and bragging rights.
//
// Records tracked:
//   - First to reach tech level 4.0
//   - First to own 10+ planets
//   - First to complete a shipping route trip
//   - First to build a Dyson Collector
//   - Wealthiest faction ever (peak credits)
//   - Largest battle (most ships involved)
//   - Most planets controlled simultaneously
//   - First to form a trade agreement
//
// Records are announced once and never change (they're historical firsts).
// This creates legacy beyond current game state.
type GalacticRecordsSystem struct {
	*BaseSystem
	records   map[string]string // recordName → faction that holds it
	peakWealth map[string]int   // factionName → peak credits
	nextCheck int64
}

func (grs *GalacticRecordsSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := grs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if grs.records == nil {
		grs.records = make(map[string]string)
		grs.peakWealth = make(map[string]int)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Track peak wealth
		if player.Credits > grs.peakWealth[player.Name] {
			grs.peakWealth[player.Name] = player.Credits
		}

		// Count planets
		planetCount := 0
		bestTech := 0.0
		hasDyson := false
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
					planetCount++
					if planet.TechLevel > bestTech {
						bestTech = planet.TechLevel
					}
					for _, be := range planet.Buildings {
						if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingDysonCollector && b.IsOperational {
							hasDyson = true
						}
					}
				}
			}
		}

		// Check records
		grs.checkRecord("First to Tech 4.0", player.Name, bestTech >= 4.0, game)
		grs.checkRecord("First to 10 Planets", player.Name, planetCount >= 10, game)
		grs.checkRecord("First Dyson Collector", player.Name, hasDyson, game)
		grs.checkRecord("First Millionaire", player.Name, player.Credits >= 1000000, game)
		grs.checkRecord("First to 5 Million", player.Name, player.Credits >= 5000000, game)

		// Ship records
		cargoShips := 0
		militaryShips := 0
		for _, ship := range player.OwnedShips {
			if ship == nil {
				continue
			}
			if ship.ShipType == entities.ShipTypeCargo {
				cargoShips++
			}
			if ship.ShipType == entities.ShipTypeFrigate ||
				ship.ShipType == entities.ShipTypeDestroyer ||
				ship.ShipType == entities.ShipTypeCruiser {
				militaryShips++
			}
		}
		grs.checkRecord("First Fleet of 10+ Cargo", player.Name, cargoShips >= 10, game)
		grs.checkRecord("First Navy of 10+ Warships", player.Name, militaryShips >= 10, game)
	}

	// Shipping route records
	routes := game.GetShippingRoutes()
	for _, route := range routes {
		if route.TripsComplete >= 1 {
			grs.checkRecord("First Shipping Route Trip", route.Owner, true, game)
		}
		if route.TripsComplete >= 10 {
			grs.checkRecord("First 10 Shipping Trips", route.Owner, true, game)
		}
	}

	// Announce records periodically
	if rand.Intn(20) == 0 && len(grs.records) > 0 {
		grs.announceRecords(game)
	}
}

func (grs *GalacticRecordsSystem) checkRecord(name, faction string, condition bool, game GameProvider) {
	if !condition {
		return
	}
	if _, exists := grs.records[name]; exists {
		return // already claimed
	}
	grs.records[name] = faction
	game.LogEvent("event", faction,
		fmt.Sprintf("🏆 GALACTIC RECORD: %s — %s! This achievement is permanent and can never be taken!",
			name, faction))
}

func (grs *GalacticRecordsSystem) announceRecords(game GameProvider) {
	if len(grs.records) == 0 {
		return
	}
	msg := "📚 Hall of Records: "
	count := 0
	for name, faction := range grs.records {
		msg += fmt.Sprintf("🏆 %s: %s ", name, faction)
		count++
		if count >= 3 {
			break
		}
	}
	game.LogEvent("intel", "", msg)
}

// GetRecords returns all galactic records.
func (grs *GalacticRecordsSystem) GetRecords() map[string]string {
	if grs.records == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range grs.records {
		result[k] = v
	}
	return result
}
