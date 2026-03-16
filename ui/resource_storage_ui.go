package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/entities"
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
		y:        450,
		width:    300,
		height:   260,
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

	// Dark theme panel
	panel := &views.UIPanel{
		X: rsu.x, Y: rsu.y, Width: rsu.width, Height: rsu.height,
		BgColor: utils.Theme.PanelBg, BorderColor: utils.Theme.PanelBorder,
	}
	panel.Draw(screen)

	// Title
	titleY := rsu.y + 15
	views.DrawText(screen, "Resource Storage", rsu.x+10, titleY, utils.Theme.Accent)

	// Power status (right of title)
	if pd.PowerConsumed > 0 || pd.PowerRatio < 1.0 {
		pwrColor := utils.SystemGreen
		if pd.PowerRatio < 0.5 {
			pwrColor = utils.SystemRed
		} else if pd.PowerRatio < 0.8 {
			pwrColor = utils.SystemOrange
		}
		pwrStr := fmt.Sprintf("Power: %.0f%%", pd.PowerRatio*100)
		views.DrawText(screen, pwrStr, rsu.x+rsu.width-len(pwrStr)*6-10, titleY, pwrColor)
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
		views.DrawText(screen, happyStr, rsu.x+rsu.width-len(happyStr)*6-10, titleY-12, happyColor)
	}

	resourceY := titleY + 20

	if len(pd.StoredResources) == 0 {
		views.DrawText(screen, "No resources stored", rsu.x+10, resourceY, utils.TextSecondary)
		return
	}

	// Get net flows
	rates := rsu.provider.GetRatesData()
	var netFlow map[string]float64
	if rates != nil {
		netFlow = rates.NetFlow
	}

	maxVisible := 12
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
		resourceY += 18
		count++
	}

	if len(pd.StoredResources) > maxVisible {
		moreY := resourceY + 3
		moreText := fmt.Sprintf("...and %d more", len(pd.StoredResources)-maxVisible)
		views.DrawText(screen, moreText, rsu.x+10, moreY, utils.TextSecondary)
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
	flowX := textX + len(label)*6 + 2
	if flow > 0.5 {
		flowStr := fmt.Sprintf("+%.0f", flow)
		views.DrawText(screen, flowStr, flowX, y, utils.SystemGreen)
	} else if flow < -0.5 {
		flowStr := fmt.Sprintf("%.0f", flow)
		views.DrawText(screen, flowStr, flowX, y, utils.SystemRed)
	}

	// Amount / capacity on the right
	amtStr := fmt.Sprintf("%d/%d", amount, capacity)
	amtWidth := len(amtStr) * 6
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
