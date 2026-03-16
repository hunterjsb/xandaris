package core

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hunterjsb/xandaris/views"
)

// drawTickInfo draws the bottom-left status bar + top-right empire panel.
func (a *App) drawTickInfo(screen *ebiten.Image) {
	if a.viewManager.GetCurrentView().GetType() == views.ViewTypeMainMenu || a.Server.TickManager == nil {
		return
	}

	// Hide status bar when command bar is open to avoid overlap
	if a.commandBar == nil || !a.commandBar.IsOpen() {
		a.drawStatusBar(screen)
	}
	a.drawEmpirePanel(screen)
}

// drawStatusBar renders speed/credits/controls in the bottom-left.
func (a *App) drawStatusBar(screen *ebiten.Image) {
	x := 10
	y := a.screenHeight - 55

	panel := &views.UIPanel{
		X: x, Y: y, Width: 520, Height: 50,
		BgColor:     utils.Theme.PanelBg,
		BorderColor: utils.Theme.PanelBorder,
	}
	panel.Draw(screen)

	textX := x + 10
	textY := y + 12

	// Line 1: Speed + game time + construction indicator
	speedStr := a.Server.TickManager.GetSpeedString()
	timeStr := a.Server.TickManager.GetGameTimeFormatted()
	line1 := fmt.Sprintf("Speed: %s  %s", speedStr, timeStr)
	if a.Server.TickManager.IsPaused() {
		views.DrawText(screen, line1, textX, textY, utils.Theme.TextDim)
		views.DrawText(screen, "PAUSED", textX+len(line1+"  ")*utils.CharWidth(), textY, utils.SystemYellow)
	} else {
		views.DrawText(screen, line1, textX, textY, utils.Theme.TextLight)
	}

	// Construction count indicator (right side of line 1)
	if human := a.Server.State.HumanPlayer; human != nil {
		queueCount := len(a.getConstructionItems(human.Name))
		if queueCount > 0 {
			qLabel := fmt.Sprintf("Building %d", queueCount)
			qX := x + 520 - len(qLabel)*utils.CharWidth() - 10
			views.DrawText(screen, qLabel, qX, textY, utils.SystemGreen)
		}
	}

	// Line 2: Credits with colored net flow + hints
	textY += 15
	if human := a.Server.State.HumanPlayer; human != nil {
		income := 0
		upkeep := 0
		for _, planet := range human.OwnedPlanets {
			if planet == nil {
				continue
			}
			income += int(planet.Population / 100)
			upkeep += int(planet.Population / 1000)
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.IsOperational {
					if cost, found := economy.BuildingCreditUpkeep[b.BuildingType]; found {
						upkeep += cost + (b.Level - 1)
					}
					if b.BuildingType == "Trading Post" {
						income += 2 * b.Level
					}
				}
			}
		}

		credLabel := fmt.Sprintf("Credits: %d ", human.Credits)
		views.DrawText(screen, credLabel, textX, textY, utils.Theme.TextLight)

		// Color-coded net flow
		net := income - upkeep
		flowX := textX + len(credLabel)*utils.CharWidth()
		if net > 0 {
			views.DrawText(screen, fmt.Sprintf("(+%d/s)", net), flowX, textY, utils.SystemGreen)
			flowX += len(fmt.Sprintf("(+%d/s)", net))*utils.CharWidth() + 6
		} else if net < 0 {
			views.DrawText(screen, fmt.Sprintf("(%d/s)", net), flowX, textY, utils.SystemRed)
			flowX += len(fmt.Sprintf("(%d/s)", net))*utils.CharWidth() + 6
		} else {
			views.DrawText(screen, "(0/s)", flowX, textY, utils.Theme.TextDim)
			flowX += len("(0/s)")*utils.CharWidth() + 6
		}

		// Hints after credits
		hints := "[`] Chat"
		if !a.IsRemote() {
			hints += "  [Space] Pause"
		}
		views.DrawText(screen, hints, flowX, textY, utils.Theme.TextDim)
	}
}

// drawEmpirePanel renders a persistent top-right panel with empire vitals.
// Hidden on planet view to avoid overlap with planet-specific UI.
func (a *App) drawEmpirePanel(screen *ebiten.Image) {
	// Don't show on planet view — the planet info panel has all the details
	if a.viewManager.GetCurrentView().GetType() == views.ViewTypePlanet {
		return
	}

	human := a.Server.State.HumanPlayer
	if human == nil || len(human.OwnedPlanets) == 0 {
		return
	}

	// Calculate panel height based on content
	panelW := 300
	perPlanet := 50
	hasPower := false
	var totalPop int64
	for _, planet := range human.OwnedPlanets {
		if planet == nil {
			continue
		}
		totalPop += planet.Population
		if planet.PowerConsumed > 0 || planet.PowerGenerated > 0 {
			hasPower = true
		}
	}
	if hasPower {
		perPlanet = 68
	}
	// +18 for the total pop line, +extra for construction queue
	constructionItems := a.getConstructionItems(human.Name)
	queueItems := len(constructionItems)
	queueHeight := 0
	if queueItems > 0 {
		shown := queueItems
		if shown > 3 {
			shown = 3
		}
		queueHeight = 24 + shown*13
		if queueItems > 3 {
			queueHeight += 13
		}
	}
	panelH := 42 + len(human.OwnedPlanets)*perPlanet + queueHeight + 4
	if panelH > 500 {
		panelH = 500
	}
	x := a.screenWidth - panelW - 10
	y := 10

	panel := &views.UIPanel{
		X: x, Y: y, Width: panelW, Height: panelH,
		BgColor:     utils.Theme.PanelBg,
		BorderColor: utils.Theme.PanelBorder,
	}
	panel.Draw(screen)

	textY := y + 10
	views.DrawText(screen, human.Name, x+10, textY, utils.Theme.Accent)

	// Credits right-aligned
	credStr := formatCredits(human.Credits)
	credW := len(credStr) * utils.CharWidth()
	views.DrawText(screen, credStr, x+panelW-credW-10, textY, utils.Theme.TextLight)
	textY += 14

	// Total population + planet count summary
	popSummary := fmt.Sprintf("Pop: %s  %d planets", utils.FormatInt64WithCommas(totalPop), len(human.OwnedPlanets))
	views.DrawText(screen, popSummary, x+10, textY, utils.Theme.TextDim)
	textY += 16

	a.empirePlanetHits = a.empirePlanetHits[:0] // reset hit regions

	for _, planet := range human.OwnedPlanets {
		if planet == nil || textY > y+panelH-10 {
			continue
		}

		hitStart := textY

		// Planet name (clickable)
		views.DrawText(screen, planet.Name, x+10, textY, utils.Theme.Accent)
		textY += 15

		// Population + happiness on same line
		popCap := planet.GetTotalPopulationCapacity()
		popStr := fmt.Sprintf("Pop: %d", planet.Population)
		if popCap > 0 {
			popStr = fmt.Sprintf("Pop: %d/%d", planet.Population, popCap)
		}
		views.DrawText(screen, popStr, x+14, textY, utils.Theme.TextDim)

		happyStr := fmt.Sprintf("%.0f%%", planet.Happiness*100)
		happyColor := utils.SystemGreen
		if planet.Happiness < 0.4 {
			happyColor = utils.SystemRed
		} else if planet.Happiness < 0.7 {
			happyColor = utils.SystemOrange
		}
		views.DrawText(screen, happyStr, x+panelW-36, textY, happyColor)
		textY += 15

		// Power bar + label
		if planet.PowerConsumed > 0 || planet.PowerGenerated > 0 {
			powerRatio := planet.GetPowerRatio()
			barX := x + 14
			barW := panelW - 28
			barH := 5

			bg := &views.UIPanel{X: barX, Y: textY, Width: barW, Height: barH,
				BgColor: utils.Theme.BarBg, BorderColor: color.RGBA{30, 35, 55, 255}}
			bg.Draw(screen)

			fillW := int(float64(barW) * powerRatio)
			if fillW > 0 {
				fillColor := utils.SystemGreen
				if powerRatio < 0.5 {
					fillColor = utils.SystemRed
				} else if powerRatio < 0.8 {
					fillColor = utils.SystemOrange
				}
				fill := &views.UIPanel{X: barX + 1, Y: textY + 1, Width: fillW - 2, Height: barH - 2,
					BgColor: fillColor, BorderColor: fillColor}
				fill.Draw(screen)
			}

			pwrLabel := fmt.Sprintf("%.0f/%.0fMW", planet.PowerGenerated, planet.PowerConsumed)
			views.DrawText(screen, pwrLabel, barX, textY+barH+3, utils.Theme.TextDim)
			textY += barH + 16
		} else {
			textY += 8
		}

		// Resource warnings (compact, one-line)
		var warnings []string
		waterStored := planet.GetStoredAmount("Water")
		if planet.Population > 0 && waterStored < 10 {
			warnings = append(warnings, "Water!")
		}
		fuelStored := planet.GetStoredAmount("Fuel")
		if (planet.PowerGenerated > 0 || planet.PowerConsumed > 0) && fuelStored < 5 {
			warnings = append(warnings, "Fuel!")
		}
		if len(warnings) > 0 {
			warnStr := ""
			for i, w := range warnings {
				if i > 0 {
					warnStr += " "
				}
				warnStr += w
			}
			views.DrawText(screen, warnStr, x+14, textY, utils.SystemRed)
			textY += 12
		}

		// Record clickable hit region for this planet
		a.empirePlanetHits = append(a.empirePlanetHits, empirePlanetHit{
			PlanetID: planet.GetID(),
			Y1: hitStart, Y2: textY,
			X1: x, X2: x + panelW,
		})
	}

	// Construction queue summary (below planets)
	if len(constructionItems) > 0 && textY < y+panelH-30 {
		// Separator
		views.DrawLine(screen, x+10, textY+2, x+panelW-10, textY+2, utils.Theme.PanelBorder)
		textY += 10

		views.DrawText(screen, fmt.Sprintf("Building (%d)", len(constructionItems)), x+10, textY, utils.Theme.Accent)
		textY += 14

		// Show up to 3 items
		shown := 0
		for _, item := range constructionItems {
			if shown >= 3 || textY > y+panelH-15 {
				break
			}
			label := fmt.Sprintf("%s %d%%", item.Name, item.Progress)
			views.DrawText(screen, label, x+14, textY, utils.Theme.TextDim)
			textY += 13
			shown++
		}
		if len(constructionItems) > shown {
			views.DrawText(screen, fmt.Sprintf("+%d more", len(constructionItems)-shown), x+14, textY, utils.Theme.TextDim)
			textY += 13
		}
	}
}

// handleEmpirePanelClick checks if the user clicked on a planet in the empire panel.
func (a *App) handleEmpirePanelClick() {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	mx, my := ebiten.CursorPosition()
	for _, hit := range a.empirePlanetHits {
		if mx >= hit.X1 && mx <= hit.X2 && my >= hit.Y1 && my <= hit.Y2 {
			// Navigate to this planet
			a.navigateToPlanet(hit.PlanetID)
			return
		}
	}
}

// navigateToPlanet switches to the planet view for the given planet ID.
func (a *App) navigateToPlanet(planetID int) {
	for _, sys := range a.Server.State.Systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.GetID() == planetID {
				// Switch to planet view
				if pv, ok := a.viewManager.GetView(views.ViewTypePlanet).(interface {
					SetPlanet(*entities.Planet)
				}); ok {
					pv.SetPlanet(planet)
					a.viewManager.SwitchTo(views.ViewTypePlanet)
				}
				return
			}
		}
	}
}

func formatCredits(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM cr", float64(n)/1000000.0)
	}
	if n >= 10000 {
		return fmt.Sprintf("%.0fk cr", float64(n)/1000.0)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk cr", float64(n)/1000.0)
	}
	return fmt.Sprintf("%d cr", n)
}
