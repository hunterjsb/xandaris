package ui

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
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
		y:      720 - 180,
		width:  290,
		height: 170,
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

	// Draw storage utilization bar
	utilizationY := titleY + 20
	rsu.drawStorageBarForPlanet(screen, utilizationY, planet)

	// Draw separator
	separatorY := utilizationY + 20
	views.DrawLine(screen, rsu.x+10, separatorY, rsu.x+rsu.width-10, separatorY, utils.PanelBorder)

	// Draw stored resources or empty message
	resourceY := separatorY + 8

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
	maxVisible := 6

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

	for _, resource := range sortedResources {
		if count >= maxVisible {
			break
		}

		rsu.drawResourceEntry(screen, resource.resourceType, resource.storage, resourceY)
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

// drawStorageBarForPlanet draws the overall storage utilization bar for a specific planet
func (rsu *ResourceStorageUI) drawStorageBarForPlanet(screen *ebiten.Image, y int, planet *entities.Planet) {
	// Calculate utilization
	used := planet.GetTotalStorageUsed()
	capacity := planet.StorageCapacity
	utilization := 0.0
	if capacity > 0 {
		utilization = float64(used) / float64(capacity)
	}

	// Draw text showing usage (simplified - no bar for now)
	textY := y + 5
	usageText := fmt.Sprintf("Storage: %d / %d (%.0f%%)", used, capacity, utilization*100)
	views.DrawText(screen, usageText, rsu.x+10, textY, utils.TextPrimary)
}

// drawResourceEntry draws a single resource entry
func (rsu *ResourceStorageUI) drawResourceEntry(screen *ebiten.Image, resourceType string, storage *entities.ResourceStorage, y int) {
	textX := rsu.x + 15

	// Simplified display - just show resource and amount on one line
	entryText := fmt.Sprintf("  %s: %d", resourceType, storage.Amount)
	views.DrawText(screen, entryText, textX, y, utils.TextPrimary)
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
