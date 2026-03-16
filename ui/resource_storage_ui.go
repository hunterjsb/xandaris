package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/ui/widgets"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hunterjsb/xandaris/views"
)

// ResourceStorageUI displays stored resources on a planet
type ResourceStorageUI struct {
	ctx      UIContext
	provider *PlanetDataProvider
	planet   *entities.Planet
	x        int
	y        int
	width    int
	height   int
}

// NewResourceStorageUI creates a new resource storage UI
func NewResourceStorageUI(ctx UIContext, provider *PlanetDataProvider) *ResourceStorageUI {
	return &ResourceStorageUI{
		ctx:      ctx,
		provider: provider,
		x:        10,
		y:        views.ScreenHeight - 450,
		width:    25 * utils.CharWidth(),
		height:   400,
	}
}

// SetPlanet sets the planet to display resources for
func (rsu *ResourceStorageUI) SetPlanet(planet *entities.Planet) {
	rsu.planet = planet
}

// Update updates the resource storage UI
func (rsu *ResourceStorageUI) Update() {
	// No input handling needed
}

// Draw renders the resource storage panel
func (rsu *ResourceStorageUI) Draw(screen *ebiten.Image) {
	pd := rsu.provider.GetPlanetData()
	if pd == nil || rsu.ctx.GetState().HumanPlayer == nil {
		return
	}

	if pd.Owner != rsu.ctx.GetState().HumanPlayer.Name {
		return
	}

	// Build outer panel using widgets.Panel for the container frame + title
	p := widgets.NewPanel(widgets.AnchorManual, 25)
	p.X = rsu.x
	p.Y = rsu.y

	// Title line with power status on the right
	titleLeft := "Resource Storage"
	titleRight := ""
	titleRightColor := utils.Theme.Accent
	if pd.PowerConsumed > 0 || pd.PowerRatio < 1.0 {
		titleRightColor = utils.SystemGreen
		if pd.PowerRatio < 0.5 {
			titleRightColor = utils.SystemRed
		} else if pd.PowerRatio < 0.8 {
			titleRightColor = utils.SystemOrange
		}
		titleRight = fmt.Sprintf("Power: %.0f%%", pd.PowerRatio*100)
	}
	if titleRight != "" {
		p.LinePair(titleLeft, utils.Theme.Accent, titleRight, titleRightColor)
	} else {
		p.Line(titleLeft, utils.Theme.Accent)
	}

	// Happiness indicator
	if pd.Population > 0 {
		happyColor := utils.SystemGreen
		if pd.Happiness < 0.4 {
			happyColor = utils.SystemRed
		} else if pd.Happiness < 0.7 {
			happyColor = utils.SystemOrange
		}
		happyStr := fmt.Sprintf("%.0f%% happy", pd.Happiness*100)
		p.LineRight(happyStr, happyColor)
	}

	p.Sep()

	// Count lines for resources (to size the panel correctly)
	maxVisible := 12
	resourceCount := len(pd.StoredResources)
	visibleCount := resourceCount
	if visibleCount > maxVisible {
		visibleCount = maxVisible
	}

	// Add placeholder lines for resources so the panel sizes correctly
	for i := 0; i < visibleCount; i++ {
		p.Line("", utils.Theme.TextDim) // placeholder for manual resource drawing
	}
	if resourceCount > maxVisible {
		p.Line(fmt.Sprintf("...and %d more", resourceCount-maxVisible), utils.TextSecondary)
	}
	if resourceCount == 0 {
		p.Line("No resources stored", utils.TextSecondary)
	}

	p.Draw(screen)

	// Now draw the resource entries manually inside the panel bounds
	if resourceCount > 0 {
		px, py, _, _ := p.GetBounds()
		lineH := widgets.LineH()
		cw := utils.CharWidth()
		pad := cw // 1 char padding

		// Calculate starting Y: skip title line + happiness (if shown) + separator
		resourceY := py + pad
		resourceY += lineH // title
		if pd.Population > 0 {
			resourceY += lineH // happiness line
		}
		resourceY += lineH / 2 // separator

		// Get net flows
		rates := rsu.provider.GetRatesData()
		var netFlow map[string]float64
		if rates != nil {
			netFlow = rates.NetFlow
		}

		// Override rsu.x and rsu.width for resource entries to match panel bounds
		origX := rsu.x
		origW := rsu.width
		rsu.x = px
		rsu.width = 25 * cw

		count := 0
		for _, entry := range pd.StoredResources {
			if count >= maxVisible {
				break
			}

			flow := 0.0
			if netFlow != nil {
				flow = netFlow[entry.ResourceType]
			}
			rsu.drawResourceEntry(screen, entry.ResourceType, entry.Amount, entry.Capacity, flow, resourceY)
			resourceY += lineH
			count++
		}

		rsu.x = origX
		rsu.width = origW
	}
}

// drawResourceEntry draws a single resource entry with capacity bar and flow indicator
func (rsu *ResourceStorageUI) drawResourceEntry(screen *ebiten.Image, resourceType string, amount, capacity int, flow float64, y int) {
	textX := rsu.x + 10

	amtColor := utils.TextPrimary
	if amount == 0 {
		amtColor = utils.SystemRed
	} else if capacity > 0 && amount >= capacity-10 {
		amtColor = utils.SystemOrange
	}

	label := resourceType
	views.DrawText(screen, label, textX, y, amtColor)

	// Net flow indicator
	flowX := textX + len(label)*utils.CharWidth() + 2
	if flow > 0.5 {
		flowStr := fmt.Sprintf("+%.0f", flow)
		views.DrawText(screen, flowStr, flowX, y, utils.SystemGreen)
	} else if flow < -0.5 {
		flowStr := fmt.Sprintf("%.0f", flow)
		views.DrawText(screen, flowStr, flowX, y, utils.SystemRed)
	}

	// Amount / capacity on the right
	amtStr := fmt.Sprintf("%d/%d", amount, capacity)
	amtWidth := len(amtStr) * utils.CharWidth()
	views.DrawText(screen, amtStr, rsu.x+rsu.width-amtWidth-15, y, amtColor)

	// Capacity bar
	barX := textX + 100
	barW := rsu.width - 100 - amtWidth - 25
	if barW > 20 {
		barY := y + 3
		barH := 4

		barBg := &views.UIPanel{X: barX, Y: barY, Width: barW, Height: barH,
			BgColor: utils.PanelBg, BorderColor: utils.PanelBorder}
		barBg.Draw(screen)

		if capacity > 0 {
			fillW := int(float64(barW) * float64(amount) / float64(capacity))
			if fillW > 2 {
				fillColor := utils.SystemGreen
				pct := float64(amount) / float64(capacity)
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
	pd := rsu.provider.GetPlanetData()
	if pd == nil || rsu.ctx.GetState().HumanPlayer == nil {
		return false
	}
	return pd.Owner == rsu.ctx.GetState().HumanPlayer.Name
}
