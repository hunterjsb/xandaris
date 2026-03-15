package ui

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/views"
	"github.com/hunterjsb/xandaris/utils"
)

// ResourceStorageUI displays stored resources on a planet
type ResourceStorageUI struct {
	ctx   UIContext
	planet *entities.Planet
	x      int
	y      int
	width  int
	height int
}

// NewResourceStorageUI creates a new resource storage UI
func NewResourceStorageUI(ctx UIContext) *ResourceStorageUI {
	return &ResourceStorageUI{
		ctx:   ctx,
		x:      220,
		y:      720 - 250,
		width:  320,
		height: 180,
	}
}

// SetPlanet sets the planet to display resources for
func (rsu *ResourceStorageUI) SetPlanet(planet *entities.Planet) {
	rsu.planet = planet
}

// getCurrentPlanet gets the actual current planet from game state
// This ensures we always read from the live planet object, not a stale reference
func (rsu *ResourceStorageUI) getCurrentPlanet() *entities.Planet {
	if rsu.planet == nil || rsu.ctx.GetState().HumanPlayer == nil {
		return nil
	}

	// Find the matching planet in the player's owned planets by ID
	planetID := rsu.planet.GetID()
	for _, ownedPlanet := range rsu.ctx.GetState().HumanPlayer.OwnedPlanets {
		if ownedPlanet.GetID() == planetID {
			return ownedPlanet
		}
	}

	// If not found in owned planets, return the reference we have
	return rsu.planet
}

// Update updates the resource storage UI
func (rsu *ResourceStorageUI) Update() {
	// No input handling needed for now
}

// Draw renders the resource storage panel
func (rsu *ResourceStorageUI) Draw(screen *ebiten.Image) {
	// Get the current planet from game state
	planet := rsu.getCurrentPlanet()
	if planet == nil || rsu.ctx.GetState().HumanPlayer == nil {
		return
	}

	if planet.Owner != rsu.ctx.GetState().HumanPlayer.Name {
		return
	}

	// Show the panel even with no resources to provide feedback
	hasResources := len(planet.StoredResources) > 0

	// Draw background panel
	panel := views.NewUIPanel(rsu.x, rsu.y, rsu.width, rsu.height)
	panel.Draw(screen)

	// Draw title
	titleY := rsu.y + 15
	views.DrawText(screen, "Resource Storage", rsu.x+10, titleY, utils.TextPrimary)

	// Power status bar (compact, right of title)
	if planet.PowerConsumed > 0 {
		powerRatio := planet.GetPowerRatio()
		pwrColor := utils.SystemGreen
		if powerRatio < 0.5 {
			pwrColor = utils.SystemRed
		} else if powerRatio < 0.8 {
			pwrColor = utils.SystemOrange
		}
		pwrStr := fmt.Sprintf("Power: %.0f%%", powerRatio*100)
		views.DrawText(screen, pwrStr, rsu.x+rsu.width-len(pwrStr)*6-10, titleY, pwrColor)
	}

	// Happiness indicator
	if planet.Population > 0 {
		happyColor := utils.SystemGreen
		if planet.Happiness < 0.4 {
			happyColor = utils.SystemRed
		} else if planet.Happiness < 0.7 {
			happyColor = utils.SystemOrange
		}
		happyStr := fmt.Sprintf("%.0f%% happy", planet.Happiness*100)
		views.DrawText(screen, happyStr, rsu.x+rsu.width-len(happyStr)*6-10, titleY-12, happyColor)
	}

	// Draw stored resources or empty message
	resourceY := titleY + 20

	if !hasResources {
		// Show message when no resources are stored
		emptyText := "No resources stored"
		views.DrawText(screen, emptyText, rsu.x+10, resourceY, utils.TextSecondary)

		// Show debug info about owned resources on planet
		debugY := resourceY + 20
		ownedCount := 0
		for _, resourceEntity := range planet.Resources {
			if resource, ok := resourceEntity.(*entities.Resource); ok {
				if resource.Owner == rsu.ctx.GetState().HumanPlayer.Name {
					ownedCount++
				}
			}
		}
		debugText := fmt.Sprintf("Owned resources: %d", ownedCount)
		views.DrawText(screen, debugText, rsu.x+10, debugY, utils.TextSecondary)
		return
	}

	count := 0
	maxVisible := 7

	// Create sorted list of resources to prevent flickering
	var sortedResources []struct {
		resourceType string
		storage      *entities.ResourceStorage
	}

	for resourceType, storage := range planet.StoredResources {
		sortedResources = append(sortedResources, struct {
			resourceType string
			storage      *entities.ResourceStorage
		}{resourceType, storage})
	}

	// Sort alphabetically by resource type
	sort.Slice(sortedResources, func(i, j int) bool {
		return sortedResources[i].resourceType < sortedResources[j].resourceType
	})

	// Compute net flow for each resource
	netFlow := rsu.computeNetFlow(planet)

	for _, resource := range sortedResources {
		if count >= maxVisible {
			break
		}

		rsu.drawResourceEntry(screen, resource.resourceType, resource.storage, netFlow[resource.resourceType], resourceY)
		resourceY += 18
		count++
	}

	// Show "and X more" if there are more resources
	if len(planet.StoredResources) > maxVisible {
		moreY := resourceY + 3
		moreText := fmt.Sprintf("...and %d more", len(planet.StoredResources)-maxVisible)
		views.DrawText(screen, moreText, rsu.x+10, moreY, utils.TextSecondary)
	}
}



// computeNetFlow calculates production - consumption for each resource on a planet.
func (rsu *ResourceStorageUI) computeNetFlow(planet *entities.Planet) map[string]float64 {
	flow := make(map[string]float64)

	// Mine production
	for _, resEntity := range planet.Resources {
		res, ok := resEntity.(*entities.Resource)
		if !ok || res.Abundance <= 0 {
			continue
		}
		resIDStr := fmt.Sprintf("%d", res.GetID())
		multiplier := 0.0
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok {
				if b.BuildingType == "Mine" && b.AttachedTo == resIDStr && b.IsOperational {
					multiplier += b.GetStaffingRatio() * b.ProductionBonus
				}
			}
		}
		if multiplier > 0 {
			af := float64(res.Abundance) / 70.0
			if af > 1.0 {
				af = 1.0
			}
			if af < 0.1 {
				af = 0.1
			}
			flow[res.ResourceType] += 8.0 * res.ExtractionRate * multiplier * af
		}
	}

	// Refinery: +Fuel, -Oil
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.BuildingType == "Refinery" && b.IsOperational {
			lm := 1.0 + float64(b.Level-1)*0.3
			flow["Fuel"] += 3.0 * lm
			flow["Oil"] -= 2.0 * lm
		}
	}

	// Factory: +Electronics, -Rare Metals, -Iron
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.BuildingType == "Factory" && b.IsOperational {
			lm := 1.0 + float64(b.Level-1)*0.3
			flow["Electronics"] += 2.0 * lm
			flow["Rare Metals"] -= 2.0 * lm
			flow["Iron"] -= 1.0 * lm
		}
	}

	// Population consumption (from economy.PopulationConsumption — single source of truth)
	for _, rate := range economy.PopulationConsumption {
		flow[rate.ResourceType] -= float64(planet.Population) / rate.PopDivisor * rate.PerPopulation
	}

	// Building upkeep (from economy.BuildingResourceUpkeep — single source of truth)
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.IsOperational {
			if upkeeps, found := economy.BuildingResourceUpkeep[b.BuildingType]; found {
				for _, u := range upkeeps {
					flow[u.ResourceType] -= float64(u.Amount)
				}
			}
		}
	}

	return flow
}

// drawResourceEntry draws a single resource entry with capacity bar and flow indicator
func (rsu *ResourceStorageUI) drawResourceEntry(screen *ebiten.Image, resourceType string, storage *entities.ResourceStorage, flow float64, y int) {
	textX := rsu.x + 10

	// Resource name
	amtColor := utils.TextPrimary
	if storage.Amount == 0 {
		amtColor = utils.SystemRed
	} else if storage.Capacity > 0 && storage.Amount >= storage.Capacity-10 {
		amtColor = utils.SystemOrange
	}

	label := resourceType
	views.DrawText(screen, label, textX, y, amtColor)

	// Net flow indicator after the name (show rate per interval)
	flowX := textX + len(label)*6 + 2
	if flow > 0.5 {
		flowStr := fmt.Sprintf("+%.0f", flow)
		views.DrawText(screen, flowStr, flowX, y, utils.SystemGreen)
	} else if flow < -0.5 {
		flowStr := fmt.Sprintf("%.0f", flow)
		views.DrawText(screen, flowStr, flowX, y, utils.SystemRed)
	}

	// Amount / capacity on the right
	amtStr := fmt.Sprintf("%d/%d", storage.Amount, storage.Capacity)
	amtWidth := len(amtStr) * 6
	views.DrawText(screen, amtStr, rsu.x+rsu.width-amtWidth-15, y, amtColor)

	// Small capacity bar
	barX := textX + 100
	barW := rsu.width - 100 - amtWidth - 25
	if barW > 20 {
		barY := y + 3
		barH := 4

		// Bar background
		barBg := &views.UIPanel{X: barX, Y: barY, Width: barW, Height: barH,
			BgColor: utils.PanelBg, BorderColor: utils.PanelBorder}
		barBg.Draw(screen)

		// Bar fill
		if storage.Capacity > 0 {
			fillW := int(float64(barW) * float64(storage.Amount) / float64(storage.Capacity))
			if fillW > 2 {
				fillColor := utils.SystemGreen
				pct := float64(storage.Amount) / float64(storage.Capacity)
				if pct > 0.8 {
					fillColor = utils.SystemOrange
				}
				if pct > 0.95 {
					fillColor = utils.SystemRed
				}
				barFill := &views.UIPanel{X: barX + 1, Y: barY + 1, Width: fillW - 2, Height: barH - 2,
					BgColor: fillColor, BorderColor: fillColor}
				barFill.Draw(screen)
			}
		}
	}
}

// IsVisible returns whether the UI should be visible
func (rsu *ResourceStorageUI) IsVisible() bool {
	planet := rsu.getCurrentPlanet()
	if planet == nil || rsu.ctx.GetState().HumanPlayer == nil {
		return false
	}
	// Always show for owned planets to provide feedback
	return planet.Owner == rsu.ctx.GetState().HumanPlayer.Name
}
