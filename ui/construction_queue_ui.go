package ui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hunterjsb/xandaris/views"
)

// ConstructionQueueUI displays all active construction items
type ConstructionQueueUI struct {
	ctx          UIContext
	provider     *PlanetDataProvider
	x            int
	y            int
	width        int
	itemHeight   int
	maxVisible   int
	scrollOffset int
}

// NewConstructionQueueUI creates a new construction queue UI
func NewConstructionQueueUI(ctx UIContext, provider *PlanetDataProvider) *ConstructionQueueUI {
	return &ConstructionQueueUI{
		ctx:          ctx,
		provider:     provider,
		x:            970,
		y:            50,
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
	items := cq.provider.GetConstructionItems()
	if len(items) == 0 {
		return
	}

	// Dark theme colors
	bgColor := color.RGBA{12, 16, 28, 220}
	borderColor := color.RGBA{30, 40, 68, 255}
	accentColor := color.RGBA{127, 219, 202, 255}

	// Calculate panel height based on number of items
	visibleCount := len(items)
	if visibleCount > cq.maxVisible {
		visibleCount = cq.maxVisible
	}
	panelHeight := 50 + (visibleCount * (cq.itemHeight + 5))

	// Draw background panel
	panel := &views.UIPanel{
		X: cq.x, Y: cq.y, Width: cq.width, Height: panelHeight,
		BgColor: bgColor, BorderColor: borderColor,
	}
	panel.Draw(screen)

	// Draw title
	titleY := cq.y + 15
	views.DrawText(screen, "Construction Queue", cq.x+10, titleY, accentColor)

	// Draw total count
	countY := titleY + 15
	countText := fmt.Sprintf("%d building", len(items))
	if len(items) > 1 {
		countText = fmt.Sprintf("%d buildings", len(items))
	}
	views.DrawText(screen, countText, cq.x+10, countY, utils.TextSecondary)

	// Draw separator
	separatorY := countY + 10
	views.DrawLine(screen, cq.x+10, separatorY, cq.x+cq.width-10, separatorY, borderColor)

	// Draw construction items
	itemY := separatorY + 10
	displayed := 0

	for i := cq.scrollOffset; i < len(items) && displayed < cq.maxVisible; i++ {
		item := items[i]
		cq.drawConstructionItem(screen, item, itemY)
		itemY += cq.itemHeight + 5
		displayed++
	}

	// Draw scroll indicator if needed
	if len(items) > cq.maxVisible {
		remaining := len(items) - cq.scrollOffset - cq.maxVisible
		if remaining > 0 {
			scrollText := fmt.Sprintf("(%d more...)", remaining)
			scrollY := cq.y + panelHeight - 15
			views.DrawText(screen, scrollText, cq.x+10, scrollY, utils.TextSecondary)
		}
	}
}

// drawConstructionItem draws a single construction item
func (cq *ConstructionQueueUI) drawConstructionItem(screen *ebiten.Image, item ConstructionItemData, y int) {
	itemX := cq.x + 10
	itemW := cq.width - 20

	// Item background
	itemPanel := &views.UIPanel{
		X: itemX, Y: y, Width: itemW, Height: cq.itemHeight,
		BgColor: color.RGBA{18, 22, 38, 230}, BorderColor: color.RGBA{30, 40, 68, 255},
	}
	itemPanel.Draw(screen)

	// Construction name
	nameY := y + 10
	views.DrawText(screen, item.Name, itemX+5, nameY, utils.TextPrimary)

	// Location
	locationY := nameY + 15
	locationText := fmt.Sprintf("Location: %s", cq.getLocationName(item.Location))
	views.DrawText(screen, locationText, itemX+5, locationY, utils.TextSecondary)

	// Progress bar with accent color
	progressY := locationY + 15
	progressBarWidth := itemW - 10
	progressBarHeight := 10

	bar := views.NewUIProgressBar(itemX+5, progressY, progressBarWidth, progressBarHeight)
	bar.SetValue(float64(item.Progress), 100.0)
	bar.FillColor = color.RGBA{100, 200, 140, 255}
	bar.BgColor = color.RGBA{18, 22, 38, 255}
	bar.Draw(screen)

	// Progress text
	progressTextY := progressY + progressBarHeight + 12
	timeRemaining := cq.formatTimeRemaining(item.RemainingTicks)
	progressText := fmt.Sprintf("%d%% - %s remaining", item.Progress, timeRemaining)
	views.DrawText(screen, progressText, itemX+5, progressTextY, utils.TextSecondary)

	// Cancel hint on hover
	mx, my := ebiten.CursorPosition()
	if mx >= itemX && mx < itemX+itemW && my >= y && my < y+cq.itemHeight {
		cancelY := y + cq.itemHeight - 12
		views.DrawText(screen, "[Right-click to cancel]", itemX+5, cancelY, color.RGBA{200, 200, 100, 255})
	}
}

// getLocationName gets a friendly name for a location ID
func (cq *ConstructionQueueUI) getLocationName(locationID string) string {
	for _, system := range cq.ctx.GetState().Systems {
		for _, entity := range system.Entities {
			if fmt.Sprintf("%d", entity.GetID()) == locationID {
				return entity.GetName()
			}

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
	effectiveSpeed := cq.ctx.GetTickManager().GetEffectiveTicksPerSecond()

	if effectiveSpeed == 0 {
		return "Paused"
	}

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
	items := cq.provider.GetConstructionItems()
	if len(items) == 0 {
		return false
	}

	visibleCount := len(items)
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

	items := cq.provider.GetConstructionItems()
	itemY := cq.y + 50 + 10

	for i := cq.scrollOffset; i < len(items) && i < cq.scrollOffset+cq.maxVisible; i++ {
		itemX := cq.x + 10
		itemW := cq.width - 20

		if mx >= itemX && mx < itemX+itemW && my >= itemY && my < itemY+cq.itemHeight {
			item := items[i]
			// Send cancel command through the command channel (works in both local and remote mode)
			cq.ctx.GetCommandChannel() <- game.GameCommand{
				Type: game.CmdCancelConstruction,
				Data: game.CancelConstructionCommandData{
					ConstructionID: item.ID,
				},
			}
			cq.provider.ForceRefresh()
			return
		}

		itemY += cq.itemHeight + 5
	}
}
