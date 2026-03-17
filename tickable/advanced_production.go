package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AdvancedProductionSystem{
		BaseSystem: NewBaseSystem("AdvancedProduction", 16),
	})
}

// AdvancedProductionSystem handles production from advanced buildings:
//
// Research Lab (tech 2.5): generates 1 Electronics per interval from 1 Rare Metals
//   (passive Electronics production without a Factory's Iron requirement)
//
// Orbital Dock (tech 3.0): converts 5 Iron + 2 Electronics → repairs 50 ship HP
//   for all ships in the system (auto-repair)
//
// Trade Nexus (tech 3.5): generates 1% of galaxy trade volume as bonus credits
//   (passive income from being a trade hub)
type AdvancedProductionSystem struct {
	*BaseSystem
}

func (aps *AdvancedProductionSystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	ctx := aps.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()
	playerByName := make(map[string]*entities.Player)
	for _, p := range players {
		if p != nil {
			playerByName[p.Name] = p
		}
	}

	for _, sys := range game.GetSystems() {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			player := playerByName[planet.Owner]
			if player == nil {
				continue
			}

			for _, be := range planet.Buildings {
				b, ok := be.(*entities.Building)
				if !ok || !b.IsOperational || b.GetStaffingRatio() <= 0 {
					continue
				}

				switch b.BuildingType {
				case entities.BuildingResearchLab:
					aps.processResearchLab(planet, b)

				case entities.BuildingOrbitalDock:
					aps.processOrbitalDock(planet, b, player, sys)

				case entities.BuildingTradeNexus:
					aps.processTradeNexus(planet, b, player, game)
				}
			}
		}
	}
}

// Research Lab: 1 Rare Metals → 1 Electronics (slower than Factory but no Iron needed)
func (aps *AdvancedProductionSystem) processResearchLab(planet *entities.Planet, lab *entities.Building) {
	levelMult := 1.0 + float64(lab.Level-1)*0.3
	powerFactor := 0.25 + 0.75*planet.GetPowerRatio()
	staffing := lab.GetStaffingRatio()

	rmNeeded := int(1.0 * levelMult * staffing * powerFactor)
	elecProduced := int(1.0 * levelMult * staffing * powerFactor)

	if rmNeeded < 1 || elecProduced < 1 {
		return
	}

	if planet.GetStoredAmount(entities.ResRareMetals) < rmNeeded {
		return
	}

	planet.RemoveStoredResource(entities.ResRareMetals, rmNeeded)
	planet.AddStoredResource(entities.ResElectronics, elecProduced)
}

// Orbital Dock: auto-repairs ships in the system (5 Iron + 2 Electronics → 50 HP)
func (aps *AdvancedProductionSystem) processOrbitalDock(planet *entities.Planet, dock *entities.Building, player *entities.Player, sys *entities.System) {
	iron := planet.GetStoredAmount(entities.ResIron)
	elec := planet.GetStoredAmount(entities.ResElectronics)
	if iron < 5 || elec < 2 {
		return
	}

	// Find damaged ships in this system
	for _, ship := range player.OwnedShips {
		if ship == nil || ship.CurrentSystem != sys.ID {
			continue
		}
		if ship.CurrentHealth >= ship.MaxHealth {
			continue
		}

		// Repair
		planet.RemoveStoredResource(entities.ResIron, 5)
		planet.RemoveStoredResource(entities.ResElectronics, 2)
		repair := 50 * dock.Level
		ship.CurrentHealth += repair
		if ship.CurrentHealth > ship.MaxHealth {
			ship.CurrentHealth = ship.MaxHealth
		}

		fmt.Printf("[OrbitalDock] Repaired %s for %d HP at %s\n",
			ship.Name, repair, planet.Name)
		return // one repair per tick
	}
}

// Trade Nexus: generates bonus credits from galaxy trade volume
func (aps *AdvancedProductionSystem) processTradeNexus(planet *entities.Planet, nexus *entities.Building, player *entities.Player, game GameProvider) {
	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	tradeVol := market.GetTradeVolume()
	// 1% of trade volume per level, scaled by staffing
	income := int(tradeVol * 0.01 * float64(nexus.Level) * nexus.GetStaffingRatio())
	if income > 0 {
		player.Credits += income
	}
}
