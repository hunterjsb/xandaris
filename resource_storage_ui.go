package main

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/entities"
)

// ResourceStorageUI displays stored resources on a planet
type ResourceStorageUI struct {
	game   *Game
	planet *entities.Planet
	x      int
	y      int
	width  int
	height int
}

// NewResourceStorageUI creates a new resource storage UI
func NewResourceStorageUI(game *Game) *ResourceStorageUI {
	return &ResourceStorageUI{
		game:   game,
		x:      220, // Position on left side to avoid construction queue overlap
		y:      screenHeight - 180,
		width:  290,
		height: 170,
	}
}

// SetPlanet sets the planet to display resources for
func (rsu *ResourceStorageUI) SetPlanet(planet *entities.Planet) {
	rsu.planet = planet
}

// Update updates the resource storage UI
func (rsu *ResourceStorageUI) Update() {
	// No input handling needed for now
}

// Draw renders the resource storage panel
func (rsu *ResourceStorageUI) Draw(screen *ebiten.Image) {
	// Only show if planet is owned by player
	if rsu.planet == nil || rsu.game.humanPlayer == nil {
		return
	}

	if rsu.planet.Owner != rsu.game.humanPlayer.Name {
		return
	}

	// Show the panel even with no resources to provide feedback
	hasResources := len(rsu.planet.StoredResources) > 0

	// Draw background panel
	panel := NewUIPanel(rsu.x, rsu.y, rsu.width, rsu.height)
	panel.Draw(screen)

	// Draw title
	titleY := rsu.y + 15
	DrawText(screen, "Resource Storage", rsu.x+10, titleY, UITextPrimary)

	// Draw storage utilization bar
	utilizationY := titleY + 20
	rsu.drawStorageBar(screen, utilizationY)

	// Draw separator
	separatorY := utilizationY + 20
	DrawLine(screen, rsu.x+10, separatorY, rsu.x+rsu.width-10, separatorY, UIPanelBorder)

	// Draw stored resources or empty message
	resourceY := separatorY + 8

	if !hasResources {
		// Show message when no resources are stored
		emptyText := "No resources stored"
		DrawText(screen, emptyText, rsu.x+10, resourceY, UITextSecondary)

		// Show debug info about owned resources on planet
		debugY := resourceY + 20
		ownedCount := 0
		for _, resourceEntity := range rsu.planet.Resources {
			if resource, ok := resourceEntity.(*entities.Resource); ok {
				if resource.Owner == rsu.game.humanPlayer.Name {
					ownedCount++
				}
			}
		}
		debugText := fmt.Sprintf("Owned resources: %d", ownedCount)
		DrawText(screen, debugText, rsu.x+10, debugY, UITextSecondary)
		return
	}

	count := 0
	maxVisible := 6

	// Create sorted list of resources to prevent flickering
	var sortedResources []struct {
		resourceType string
		storage      *entities.ResourceStorage
	}

	for resourceType, storage := range rsu.planet.StoredResources {
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
	if len(rsu.planet.StoredResources) > maxVisible {
		moreY := resourceY + 3
		moreText := fmt.Sprintf("...and %d more", len(rsu.planet.StoredResources)-maxVisible)
		DrawText(screen, moreText, rsu.x+10, moreY, UITextSecondary)
	}
}

// drawStorageBar draws the overall storage utilization bar
func (rsu *ResourceStorageUI) drawStorageBar(screen *ebiten.Image, y int) {
	// Calculate utilization
	used := rsu.planet.GetTotalStorageUsed()
	capacity := rsu.planet.StorageCapacity
	utilization := 0.0
	if capacity > 0 {
		utilization = float64(used) / float64(capacity)
	}

	// Draw text showing usage (simplified - no bar for now)
	textY := y + 5
	usageText := fmt.Sprintf("Storage: %d / %d (%.0f%%)", used, capacity, utilization*100)
	DrawText(screen, usageText, rsu.x+10, textY, UITextPrimary)
}

// drawResourceEntry draws a single resource entry
func (rsu *ResourceStorageUI) drawResourceEntry(screen *ebiten.Image, resourceType string, storage *entities.ResourceStorage, y int) {
	textX := rsu.x + 15

	// Simplified display - just show resource and amount on one line
	entryText := fmt.Sprintf("  %s: %d", resourceType, storage.Amount)
	DrawText(screen, entryText, textX, y, UITextPrimary)
}

// IsVisible returns whether the UI should be visible
func (rsu *ResourceStorageUI) IsVisible() bool {
	if rsu.planet == nil || rsu.game.humanPlayer == nil {
		return false
	}
	// Always show for owned planets to provide feedback
	return rsu.planet.Owner == rsu.game.humanPlayer.Name
}
