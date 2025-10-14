package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/views"
	"github.com/hunterjsb/xandaris/utils"
)

// ConstructionQueueUI displays all active construction items
type ConstructionQueueUI struct {
	game         *Game
	x            int
	y            int
	width        int
	itemHeight   int
	maxVisible   int
	scrollOffset int
}

// NewConstructionQueueUI creates a new construction queue UI
func NewConstructionQueueUI(game *Game) *ConstructionQueueUI {
	return &ConstructionQueueUI{
		game:         game,
		x:            screenWidth - 310,
		y:            120,
		width:        300,
		itemHeight:   70,
		maxVisible:   5,
		scrollOffset: 0,
	}
}

// Update handles input for the queue UI
func (cq *ConstructionQueueUI) Update() {
	// Handle scroll wheel for scrolling through queue
	_, scrollY := ebiten.Wheel()
	if scrollY != 0 {
		mx, my := ebiten.CursorPosition()
		// Only scroll if mouse is over the panel
		if cq.isMouseOver(mx, my) {
			cq.scrollOffset -= int(scrollY)
			if cq.scrollOffset < 0 {
				cq.scrollOffset = 0
			}
		}
	}

	// Handle right-click to cancel construction
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		mx, my := ebiten.CursorPosition()
		cq.handleRightClick(mx, my)
	}
}

// Draw renders the construction queue panel
func (cq *ConstructionQueueUI) Draw(screen *ebiten.Image) {
	// Get all constructions
	constructions := cq.getAllConstructions()
	if len(constructions) == 0 {
		return // Don't show panel if nothing is building
	}

	// Calculate panel height based on number of items
	visibleCount := len(constructions)
	if visibleCount > cq.maxVisible {
		visibleCount = cq.maxVisible
	}
	panelHeight := 50 + (visibleCount * (cq.itemHeight + 5))

	// Draw background panel
	panel := views.NewUIPanel(cq.x, cq.y, cq.width, panelHeight)
	panel.Draw(screen)

	// Draw title
	titleY := cq.y + 15
	views.DrawCenteredText(screen, "Construction Queue", cq.x+cq.width/2, titleY)

	// Draw total count
	countY := titleY + 15
	countText := fmt.Sprintf("%d building", len(constructions))
	if len(constructions) > 1 {
		countText = fmt.Sprintf("%d buildings", len(constructions))
	}
	views.DrawCenteredText(screen, countText, cq.x+cq.width/2, countY)

	// Draw separator
	separatorY := countY + 10
	views.DrawLine(screen, cq.x+10, separatorY, cq.x+cq.width-10, separatorY, utils.PanelBorder)

	// Draw construction items
	itemY := separatorY + 10
	displayed := 0

	for i := cq.scrollOffset; i < len(constructions) && displayed < cq.maxVisible; i++ {
		construction := constructions[i]
		cq.drawConstructionItem(screen, construction, itemY, i)
		itemY += cq.itemHeight + 5
		displayed++
	}

	// Draw scroll indicator if needed
	if len(constructions) > cq.maxVisible {
		scrollText := fmt.Sprintf("(%d more...)", len(constructions)-cq.scrollOffset-cq.maxVisible)
		if len(constructions)-cq.scrollOffset-cq.maxVisible > 0 {
			scrollY := cq.y + panelHeight - 15
			views.DrawCenteredText(screen, scrollText, cq.x+cq.width/2, scrollY)
		}
	}
}

// drawConstructionItem draws a single construction item
func (cq *ConstructionQueueUI) drawConstructionItem(screen *ebiten.Image, item *tickable.ConstructionItem, y int, index int) {
	itemX := cq.x + 10
	itemW := cq.width - 20

	// Draw item background
	itemPanel := views.NewUIPanel(itemX, y, itemW, cq.itemHeight)
	itemPanel.BgColor = color.RGBA{25, 25, 50, 230}
	itemPanel.Draw(screen)

	// Draw construction name
	nameY := y + 10
	views.DrawText(screen, item.Name, itemX+5, nameY, utils.TextPrimary)

	// Draw location
	locationY := nameY + 15
	locationText := fmt.Sprintf("Location: %s", cq.getLocationName(item.Location))
	views.DrawText(screen, locationText, itemX+5, locationY, utils.TextSecondary)

	// Calculate progress
	item.Mutex.RLock()
	progress := item.Progress
	remainingTicks := item.RemainingTicks
	item.Mutex.RUnlock()

	// Draw progress bar
	progressY := locationY + 15
	progressBarWidth := itemW - 10
	progressBarHeight := 8

	// Background bar
	progressBg := views.NewUIPanel(itemX+5, progressY, progressBarWidth, progressBarHeight)
	progressBg.BgColor = color.RGBA{20, 20, 40, 255}
	progressBg.BorderColor = utils.PanelBorder
	progressBg.Draw(screen)

	// Progress fill
	fillWidth := int(float64(progressBarWidth) * (float64(progress) / 100.0))
	if fillWidth > 0 {
		progressFill := ebiten.NewImage(fillWidth, progressBarHeight)
		progressFill.Fill(color.RGBA{100, 200, 100, 255})
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(itemX+5), float64(progressY))
		screen.DrawImage(progressFill, opts)
	}

	// Draw progress percentage and time remaining
	progressTextY := progressY + progressBarHeight + 12
	timeRemaining := cq.formatTimeRemaining(remainingTicks)
	progressText := fmt.Sprintf("%d%% - %s remaining", progress, timeRemaining)
	views.DrawText(screen, progressText, itemX+5, progressTextY, utils.TextSecondary)

	// Draw cancel hint on hover
	mx, my := ebiten.CursorPosition()
	if mx >= itemX && mx < itemX+itemW && my >= y && my < y+cq.itemHeight {
		cancelY := y + cq.itemHeight - 12
		views.DrawText(screen, "[Right-click to cancel]", itemX+5, cancelY, color.RGBA{200, 200, 100, 255})
	}
}

// getAllConstructions gets all active construction items sorted by start time
func (cq *ConstructionQueueUI) getAllConstructions() []*tickable.ConstructionItem {
	constructionSystem := tickable.GetSystemByName("Construction")
	if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
		if cq.game.humanPlayer != nil {
			items := cs.GetConstructionsByOwner(cq.game.humanPlayer.Name)

			// Sort by start time to ensure consistent order
			// This prevents flickering from map iteration randomness
			for i := 0; i < len(items)-1; i++ {
				for j := i + 1; j < len(items); j++ {
					if items[i].Started > items[j].Started {
						items[i], items[j] = items[j], items[i]
					}
				}
			}

			return items
		}
	}
	return []*tickable.ConstructionItem{}
}

// getLocationName gets a friendly name for a location ID
func (cq *ConstructionQueueUI) getLocationName(locationID string) string {
	// Search for the location in all systems
	for _, system := range cq.game.systems {
		for _, entity := range system.Entities {
			if fmt.Sprintf("%d", entity.GetID()) == locationID {
				return entity.GetName()
			}

			// Check resources on planets
			if planet, ok := entity.(*entities.Planet); ok {
				for _, resource := range planet.Resources {
					if fmt.Sprintf("%d", resource.GetID()) == locationID {
						return fmt.Sprintf("%s on %s", resource.GetName(), planet.GetName())
					}
				}
			}
		}
	}
	return "Unknown Location"
}

// formatTimeRemaining formats remaining ticks as a time string
func (cq *ConstructionQueueUI) formatTimeRemaining(remainingTicks int) string {
	effectiveSpeed := cq.game.tickManager.GetEffectiveTicksPerSecond()

	if effectiveSpeed == 0 {
		return "Paused"
	}

	// Calculate real seconds remaining
	secondsRemaining := float64(remainingTicks) / effectiveSpeed

	if secondsRemaining < 60 {
		return fmt.Sprintf("%.0fs", secondsRemaining)
	} else if secondsRemaining < 3600 {
		minutes := int(secondsRemaining / 60)
		seconds := int(secondsRemaining) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		hours := int(secondsRemaining / 3600)
		minutes := int(secondsRemaining/60) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

// isMouseOver checks if mouse is over the panel area
func (cq *ConstructionQueueUI) isMouseOver(mx, my int) bool {
	constructions := cq.getAllConstructions()
	if len(constructions) == 0 {
		return false
	}

	visibleCount := len(constructions)
	if visibleCount > cq.maxVisible {
		visibleCount = cq.maxVisible
	}
	panelHeight := 50 + (visibleCount * (cq.itemHeight + 5))

	return mx >= cq.x && mx < cq.x+cq.width &&
		my >= cq.y && my < cq.y+panelHeight
}

// handleRightClick handles right-click to cancel construction
func (cq *ConstructionQueueUI) handleRightClick(mx, my int) {
	if !cq.isMouseOver(mx, my) {
		return
	}

	constructions := cq.getAllConstructions()
	itemY := cq.y + 50 + 10

	for i := cq.scrollOffset; i < len(constructions) && i < cq.scrollOffset+cq.maxVisible; i++ {
		itemX := cq.x + 10
		itemW := cq.width - 20

		if mx >= itemX && mx < itemX+itemW && my >= itemY && my < itemY+cq.itemHeight {
			// Cancel this construction
			construction := constructions[i]
			constructionSystem := tickable.GetSystemByName("Construction")
			if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
				// Refund partial cost based on progress
				refundAmount := int(float64(construction.Cost) * (1.0 - float64(construction.Progress)/100.0))
				cq.game.humanPlayer.Credits += refundAmount

				// Remove from queue
				cs.RemoveFromQueue(construction.Location, construction.ID)
			}
			return
		}

		itemY += cq.itemHeight + 5
	}
}
