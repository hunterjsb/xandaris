package tickable

import (
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FactionPerkSystem{
		BaseSystem: NewBaseSystem("FactionPerks", 8),
	})
}

// FactionPerkSystem applies unique bonuses based on faction identity.
// Each AI faction has a thematic specialty that makes them play differently.
// Human players get bonuses based on their highest achievement.
//
// AI Faction Perks:
//   Llama Logistics:    +25% cargo capacity on all ships
//   DeepSeek Ventures:  +15% tech growth rate
//   Gemini Exchange:    +20% Trading Post revenue
//   Grok Industries:    +20% mine production
//   Opus Cartel:        +15% population growth rate
//
// Human Perks (earned):
//   Pioneer:    (own 3+ planets) +10% colony ship speed
//   Tycoon:     (100k+ credits) +10% trade revenue
//   Warlord:    (destroy 5+ ships) +10% combat damage
type FactionPerkSystem struct {
	*BaseSystem
}

// FactionPerk describes a faction's unique bonus.
var factionPerks = map[string]struct {
	Name  string
	Bonus string
}{
	"Llama Logistics":   {"Supply Chain Masters", "+25% cargo capacity"},
	"DeepSeek Ventures": {"Data Scientists", "+15% tech growth"},
	"Gemini Exchange":   {"Market Makers", "+20% TP revenue"},
	"Grok Industries":   {"Industrial Giants", "+20% mine output"},
	"Opus Cartel":       {"Empire Builders", "+15% pop growth"},
}

func (fps *FactionPerkSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := fps.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()

	for _, player := range players {
		if player == nil {
			continue
		}

		perk, hasPerk := factionPerks[player.Name]
		if !hasPerk {
			continue
		}

		// Apply perks
		switch player.Name {
		case "Llama Logistics":
			// +25% cargo capacity on all ships
			for _, ship := range player.OwnedShips {
				if ship != nil && ship.ShipType == entities.ShipTypeCargo {
					boosted := 625 // 500 * 1.25
					if ship.MaxCargo < boosted {
						ship.MaxCargo = boosted
					}
				}
			}

		case "Grok Industries":
			// +20% mine production handled by checking specialties
			for _, sys := range game.GetSystems() {
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
						if planet.Specialties == nil {
							planet.Specialties = make(map[string]float64)
						}
						planet.Specialties["faction_mining"] = 20.0
					}
				}
			}

		case "Gemini Exchange":
			// +20% TP revenue handled in credit_production via specialties
			for _, sys := range game.GetSystems() {
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
						if planet.Specialties == nil {
							planet.Specialties = make(map[string]float64)
						}
						planet.Specialties["faction_commerce"] = 20.0
					}
				}
			}
		}

		_ = perk // used for display
	}
}
