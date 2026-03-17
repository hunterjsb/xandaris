package tickable

import (
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ResearchProductionSystem{
		BaseSystem: NewBaseSystem("ResearchProduction", 17),
	})
}

// ResearchProductionSystem handles Research Labs passively generating Electronics.
// Unlike Factories, Research Labs don't consume input resources — they represent
// pure scientific output. Production is low (1 Electronics/interval base) but
// provides an alternative Electronics source for planets without Factory inputs.
type ResearchProductionSystem struct {
	*BaseSystem
}

func (rps *ResearchProductionSystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	ctx := rps.GetContext()
	if ctx == nil {
		return
	}

	// Use system entity planets (authoritative)
	game := ctx.GetGame()
	if game == nil {
		return
	}
	for _, sys := range game.GetSystems() {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				rps.processResearchLabs(planet)
			}
		}
	}
}

func (rps *ResearchProductionSystem) processResearchLabs(planet *entities.Planet) {
	for _, be := range planet.Buildings {
		b, ok := be.(*entities.Building)
		if !ok || b.BuildingType != entities.BuildingResearchLab || !b.IsOperational {
			continue
		}
		staffing := b.GetStaffingRatio()
		if staffing <= 0 {
			continue
		}

		// Base: 1 Electronics per interval, scaling with level and tech
		levelMult := 1.0 + float64(b.Level-1)*0.3
		techBonus := 1.0 + planet.TechLevel*0.03
		powerFactor := 0.25 + 0.75*planet.GetPowerRatio()

		output := int(1.0 * levelMult * staffing * techBonus * powerFactor)
		if output < 1 {
			output = 1
		}

		// Ensure Electronics storage exists
		if _, has := planet.StoredResources[entities.ResElectronics]; !has {
			planet.AddStoredResource(entities.ResElectronics, 0)
		}

		planet.AddStoredResource(entities.ResElectronics, output)
	}
}
