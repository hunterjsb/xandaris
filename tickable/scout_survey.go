package tickable

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ScoutSurveySystem{
		BaseSystem: NewBaseSystem("ScoutSurvey", 45),
	})
}

// ScoutSurveySystem lets idle Scout ships discover new resource deposits
// on planets in their current system. Creates exploration gameplay.
type ScoutSurveySystem struct {
	*BaseSystem
}

func (sss *ScoutSurveySystem) OnTick(tick int64) {
	// Check every 200 ticks (~20 seconds)
	if tick%200 != 0 {
		return
	}

	ctx := sss.GetContext()
	if ctx == nil {
		return
	}

	players := ctx.GetPlayers()

	game := ctx.GetGame()
	if game == nil {
		return
	}
	systems := game.GetSystems()

	// Build system lookup
	systemByID := make(map[int]*entities.System)
	for _, sys := range systems {
		systemByID[sys.ID] = sys
	}

	for _, player := range players {
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeScout {
				continue
			}
			// Only idle/orbiting scouts can survey
			if ship.Status != entities.ShipStatusIdle && ship.Status != entities.ShipStatusOrbiting && ship.Status != entities.ShipStatusDocked {
				continue
			}

			sys := systemByID[ship.CurrentSystem]
			if sys == nil {
				continue
			}

			// 15% chance per check to discover something
			if rand.Intn(100) > 15 {
				continue
			}

			sss.surveySystem(sys, player, ship, game)
		}
	}
}

var surveyableResources = []struct {
	resType string
	weight  int
}{
	{"Iron", 30},
	{"Water", 25},
	{"Oil", 20},
	{"Rare Metals", 10},
	{"Helium-3", 5},
}

func (sss *ScoutSurveySystem) surveySystem(sys *entities.System, player *entities.Player, ship *entities.Ship, logger GameProvider) {
	// Find planets in this system
	for _, e := range sys.Entities {
		planet, ok := e.(*entities.Planet)
		if !ok {
			continue
		}

		// Only survey unclaimed or own planets
		if planet.Owner != "" && planet.Owner != player.Name {
			continue
		}

		// Check if planet already has many deposits (max 5)
		if len(planet.Resources) >= 5 {
			continue
		}

		// Pick a random resource type (weighted)
		totalWeight := 0
		for _, r := range surveyableResources {
			totalWeight += r.weight
		}
		roll := rand.Intn(totalWeight)
		cumulative := 0
		selectedType := "Iron"
		for _, r := range surveyableResources {
			cumulative += r.weight
			if roll < cumulative {
				selectedType = r.resType
				break
			}
		}

		// Check if this resource type already exists on the planet
		alreadyHas := false
		for _, res := range planet.Resources {
			if r, ok := res.(*entities.Resource); ok && r.ResourceType == selectedType {
				alreadyHas = true
				break
			}
		}
		if alreadyHas {
			continue
		}

		// Create the new deposit
		abundance := 30 + rand.Intn(40) // 30-70
		extractionRate := 0.5 + rand.Float64()*0.5
		nodePos := rand.Float64() * 2 * math.Pi

		deposit := &entities.Resource{
			BaseEntity: entities.BaseEntity{
				ID:           sys.ID*100000 + rand.Intn(10000),
				Name:         fmt.Sprintf("%s Deposit", selectedType),
				Type:         entities.EntityTypeResource,
				SubType:      selectedType,
				Color:        resourceColor(selectedType),
				OrbitDistance: 8 + rand.Float64()*4,
				OrbitAngle:   nodePos,
			},
			ResourceType:   selectedType,
			Abundance:      abundance,
			ExtractionRate: math.Round(extractionRate*10) / 10,
			Rarity:         resourceRarity(selectedType),
			Size:           3,
			Quality:        50 + rand.Intn(50),
			Owner:          planet.Owner,
			NodePosition:   nodePos,
		}

		planet.Resources = append(planet.Resources, deposit)

		msg := fmt.Sprintf("%s's scout discovered %s (a%d) on %s!", player.Name, selectedType, abundance, planet.Name)
		fmt.Printf("[Survey] %s\n", msg)
		if logger != nil {
			logger.LogEvent("event", player.Name, msg)
		}
		return // One discovery per survey check
	}
}

func resourceColor(resType string) color.RGBA {
	switch resType {
	case "Iron":
		return color.RGBA{180, 120, 80, 255}
	case "Water":
		return color.RGBA{80, 140, 200, 255}
	case "Oil":
		return color.RGBA{60, 60, 60, 255}
	case "Rare Metals":
		return color.RGBA{200, 180, 100, 255}
	case "Helium-3":
		return color.RGBA{180, 220, 255, 255}
	default:
		return color.RGBA{150, 150, 150, 255}
	}
}

func resourceRarity(resType string) string {
	switch resType {
	case "Iron", "Water":
		return "Common"
	case "Oil":
		return "Uncommon"
	case "Rare Metals":
		return "Rare"
	case "Helium-3":
		return "Very Rare"
	default:
		return "Common"
	}
}
