package ui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/ui/widgets"
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
		x:            views.ScreenWidth - 28*utils.CharWidth(),
		y:            50,
		width:        26 * utils.CharWidth(),
		itemHeight:   int(80.0 * utils.UIScale),
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

	// Build outer container using widgets.Panel for the title/header area
	p := widgets.NewPanel(widgets.AnchorManual, 26)
	p.X = cq.x
	p.Y = cq.y

	// Title
	p.Line("Construction Queue", utils.Theme.Accent)

	// Count
	countText := fmt.Sprintf("%d building", len(items))
	if len(items) > 1 {
		countText = fmt.Sprintf("%d buildings", len(items))
	}
	p.Line(countText, utils.TextSecondary)

	p.Sep()

	// Add placeholder lines for each visible item so the panel sizes correctly
	visibleCount := len(items)
	if visibleCount > cq.maxVisible {
		visibleCount = cq.maxVisible
	}
	// Each item takes itemHeight pixels; convert to line count
	lh := widgets.LineH()
	linesPerItem := (cq.itemHeight + 5) / lh
	if linesPerItem < 1 {
		linesPerItem = 1
	}
	for i := 0; i < visibleCount*linesPerItem; i++ {
		p.Line("", utils.Theme.TextDim) // placeholder
	}

	// Scroll indicator placeholder
	if len(items) > cq.maxVisible {
		remaining := len(items) - cq.scrollOffset - cq.maxVisible
		if remaining > 0 {
			p.Line(fmt.Sprintf("(%d more...)", remaining), utils.TextSecondary)
		}
	}

	p.Draw(screen)

	// Now draw construction items manually inside the panel bounds
	px, py, pw, _ := p.GetBounds()
	cw := utils.CharWidth()
	pad := cw

	// Items start after title + count + separator
	itemY := py + pad
	itemY += lh     // title line
	itemY += lh     // count line
	itemY += lh / 2 // separator

	// Temporarily adjust cq.x and cq.width to match panel bounds for item rendering
	origX := cq.x
	origW := cq.width
	cq.x = px
	cq.width = pw

	displayed := 0
	for i := cq.scrollOffset; i < len(items) && displayed < cq.maxVisible; i++ {
		item := items[i]
		cq.drawConstructionItem(screen, item, itemY)
		itemY += cq.itemHeight + 5
		displayed++
	}

	cq.x = origX
	cq.width = origW
}

// drawConstructionItem draws a single construction item
func (cq *ConstructionQueueUI) drawConstructionItem(screen *ebiten.Image, item ConstructionItemData, y int) {
	itemX := cq.x + 10
	itemW := cq.width - 20

	// Item background
	itemPanel := &views.UIPanel{
		X: itemX, Y: y, Width: itemW, Height: cq.itemHeight,
		BgColor: utils.Theme.PanelBgLight, BorderColor: utils.Theme.PanelBorder,
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

	// Match the widget panel sizing: header (title + count + sep) + item lines
	lh := widgets.LineH()
	cw := utils.CharWidth()
	pad := cw
	headerH := pad + lh + lh + lh/2 // pad + title + count + separator
	linesPerItem := (cq.itemHeight + 5) / lh
	if linesPerItem < 1 {
		linesPerItem = 1
	}
	panelHeight := headerH + visibleCount*linesPerItem*lh + pad/2

	return mx >= cq.x && mx < cq.x+cq.width &&
		my >= cq.y && my < cq.y+panelHeight
}

// handleRightClick handles right-click to cancel construction
func (cq *ConstructionQueueUI) handleRightClick(mx, my int) {
	if !cq.isMouseOver(mx, my) {
		return
	}

	items := cq.provider.GetConstructionItems()
	// Match the Draw layout: items start after pad + title + count + separator
	lh := widgets.LineH()
	cw := utils.CharWidth()
	pad := cw
	itemY := cq.y + pad + lh + lh + lh/2

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
