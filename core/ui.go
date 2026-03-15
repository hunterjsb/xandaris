package core

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
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

	a.drawStatusBar(screen)
	a.drawEmpirePanel(screen)
}

// drawStatusBar renders speed/credits/controls in the bottom-left.
func (a *App) drawStatusBar(screen *ebiten.Image) {
	x := 10
	y := a.screenHeight - 55

	panel := views.NewUIPanel(x, y, 340, 45)
	panel.Draw(screen)

	textX := x + 10
	textY := y + 12

	speedStr := a.Server.TickManager.GetSpeedString()
	pauseStr := ""
	if a.Server.TickManager.IsPaused() {
		pauseStr = " [PAUSED]"
	}
	views.DrawText(screen, fmt.Sprintf("Speed: %s  %s%s", speedStr, a.Server.TickManager.GetGameTimeFormatted(), pauseStr), textX, textY, utils.TextPrimary)

	credStr := ""
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
		net := income - upkeep
		netColor := "+"
		if net < 0 {
			netColor = ""
		}
		credStr = fmt.Sprintf("Credits: %d (%s%d/s)  ", human.Credits, netColor, net)
	}
	views.DrawText(screen, credStr+"[`] Chat  [Space] Pause", textX, textY+15, utils.TextSecondary)
}

// drawEmpirePanel renders a persistent top-right panel with empire vitals.
func (a *App) drawEmpirePanel(screen *ebiten.Image) {
	human := a.Server.State.HumanPlayer
	if human == nil || len(human.OwnedPlanets) == 0 {
		return
	}

	panelW := 220
	panelH := 14 + len(human.OwnedPlanets)*42 + 4
	if panelH > 200 {
		panelH = 200
	}
	x := a.screenWidth - panelW - 10
	y := 10

	panel := &views.UIPanel{
		X: x, Y: y, Width: panelW, Height: panelH,
		BgColor:     color.RGBA{12, 16, 28, 200},
		BorderColor: color.RGBA{30, 40, 68, 255},
	}
	panel.Draw(screen)

	textY := y + 10
	views.DrawText(screen, fmt.Sprintf("%s  %d planets  %d ships",
		human.Name, len(human.OwnedPlanets), len(human.OwnedShips)), x+8, textY, utils.TextSecondary)
	textY += 14

	for _, planet := range human.OwnedPlanets {
		if planet == nil || textY > y+panelH-10 {
			continue
		}

		// Planet name
		views.DrawText(screen, planet.Name, x+8, textY, color.RGBA{127, 219, 202, 255})
		textY += 12

		// Population
		popCap := planet.GetTotalPopulationCapacity()
		popStr := fmt.Sprintf("Pop: %d", planet.Population)
		if popCap > 0 {
			popStr = fmt.Sprintf("Pop: %d/%d", planet.Population, popCap)
		}
		views.DrawText(screen, popStr, x+12, textY, utils.TextSecondary)

		// Happiness indicator (right side)
		happyStr := fmt.Sprintf("%.0f%%", planet.Happiness*100)
		happyColor := utils.SystemGreen
		if planet.Happiness < 0.4 {
			happyColor = utils.SystemRed
		} else if planet.Happiness < 0.7 {
			happyColor = utils.SystemOrange
		}
		views.DrawText(screen, happyStr, x+panelW-40, textY, happyColor)
		textY += 12

		// Power bar
		if planet.PowerConsumed > 0 {
			powerRatio := planet.GetPowerRatio()
			barX := x + 12
			barW := panelW - 24
			barH := 6

			// Background
			bg := &views.UIPanel{X: barX, Y: textY, Width: barW, Height: barH,
				BgColor: color.RGBA{20, 20, 30, 255}, BorderColor: color.RGBA{40, 40, 60, 255}}
			bg.Draw(screen)

			// Fill
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

			// Label
			pwrLabel := fmt.Sprintf("%.0f/%.0fMW", planet.PowerGenerated, planet.PowerConsumed)
			views.DrawText(screen, pwrLabel, barX, textY+barH+1, utils.TextSecondary)
			textY += barH + 12
		} else {
			textY += 6
		}
	}
}
