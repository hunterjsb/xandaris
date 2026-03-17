package core

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/ui/widgets"
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
	p := widgets.NewPanel(widgets.AnchorBottomLeft, 38)

	// Line 1: Speed + game time
	speedStr := a.Server.TickManager.GetSpeedString()
	timeStr := a.Server.TickManager.GetGameTimeFormatted()
	speedLine := fmt.Sprintf("Speed: %s  %s", speedStr, timeStr)
	speedColor := utils.Theme.TextLight
	if a.Server.TickManager.IsPaused() {
		speedLine += "  PAUSED"
		speedColor = utils.SystemYellow
	}

	// Add construction indicator to speed line
	if human := a.Server.State.HumanPlayer; human != nil {
		queueCount := len(a.getConstructionItems(human.Name))
		if queueCount > 0 {
			p.LinePair(speedLine, speedColor, fmt.Sprintf("Building %d", queueCount), utils.SystemGreen)
		} else {
			p.Line(speedLine, speedColor)
		}
	} else {
		p.Line(speedLine, speedColor)
	}

	// Line 2: Credits with net flow + hints
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
		flowStr := fmt.Sprintf("(+%d/s)", net)
		flowColor := utils.SystemGreen
		if net < 0 {
			flowStr = fmt.Sprintf("(%d/s)", net)
			flowColor = utils.SystemRed
		} else if net == 0 {
			flowStr = "(0/s)"
			flowColor = utils.Theme.TextDim
		}
		credLine := fmt.Sprintf("Credits: %d %s", human.Credits, flowStr)

		hints := "[`] Chat"
		if !a.IsRemote() {
			hints += "  [Space] Pause"
		}
		p.LinePair(credLine, flowColor, hints, utils.Theme.TextDim)
	}

	p.Draw(screen)
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

	p := widgets.NewPanel(widgets.AnchorTopRight, 24)

	// Header: name + credits
	var totalPop int64
	for _, planet := range human.OwnedPlanets {
		if planet != nil {
			totalPop += planet.Population
		}
	}
	p.LinePair(human.Name, utils.Theme.Accent, formatCredits(human.Credits), utils.Theme.TextLight)
	p.Line(fmt.Sprintf("Pop: %s  %d planets", utils.FormatInt64WithCommas(totalPop), len(human.OwnedPlanets)), utils.Theme.TextDim)
	p.Sep()

	// Per-planet info
	for _, planet := range human.OwnedPlanets {
		if planet == nil {
			continue
		}

		// Planet name
		p.Line(planet.Name, utils.Theme.Accent)

		// Population + happiness
		popStr := fmt.Sprintf("Pop: %d", planet.Population)
		if cap := planet.GetTotalPopulationCapacity(); cap > 0 {
			popStr = fmt.Sprintf("Pop: %d/%d", planet.Population, cap)
		}
		happyStr := fmt.Sprintf("%.0f%%", planet.Happiness*100)
		happyColor := utils.SystemGreen
		if planet.Happiness < 0.4 {
			happyColor = utils.SystemRed
		} else if planet.Happiness < 0.7 {
			happyColor = utils.SystemOrange
		}
		p.LinePair(popStr, utils.Theme.TextDim, happyStr, happyColor)

		// Tech level (compact)
		if planet.TechLevel > 0.01 {
			era := entities.TechEraName(planet.TechLevel)
			techColor := utils.Theme.TextDim
			techStr := fmt.Sprintf("Tech %.1f %s", planet.TechLevel, era)
			if planet.GetStoredAmount(entities.ResElectronics) == 0 && planet.TechLevel > 0.1 && planet.Population > 500 {
				techColor = utils.SystemOrange
				techStr += " !"
			}
			p.Line(techStr, techColor)
		}

		// Power bar
		if planet.PowerConsumed > 0 || planet.PowerGenerated > 0 {
			powerRatio := planet.GetPowerRatio()
			pwrColor := utils.SystemGreen
			if powerRatio < 0.5 {
				pwrColor = utils.SystemRed
			} else if powerRatio < 0.8 {
				pwrColor = utils.SystemOrange
			}
			p.Bar(planet.PowerGenerated, planet.PowerConsumed,
				pwrColor, fmt.Sprintf("%.0f/%.0fMW", planet.PowerGenerated, planet.PowerConsumed))
		}

		// Resource warnings
		var warns []string
		if planet.Population > 0 && planet.GetStoredAmount("Water") < 10 {
			warns = append(warns, "Water!")
		}
		if (planet.PowerGenerated > 0 || planet.PowerConsumed > 0) && planet.GetStoredAmount("Fuel") < 5 {
			warns = append(warns, "Fuel!")
		}
		if len(warns) > 0 {
			warnStr := ""
			for i, w := range warns {
				if i > 0 {
					warnStr += " "
				}
				warnStr += w
			}
			p.Line(warnStr, utils.SystemRed)
		}
	}

	// Construction queue
	constructionItems := a.getConstructionItems(human.Name)
	if len(constructionItems) > 0 {
		p.Sep()
		p.Line(fmt.Sprintf("Building (%d)", len(constructionItems)), utils.Theme.Accent)
		shown := 0
		for _, item := range constructionItems {
			if shown >= 3 {
				break
			}
			p.Line(fmt.Sprintf("%s %d%%", item.Name, item.Progress), utils.Theme.TextDim)
			shown++
		}
		if len(constructionItems) > shown {
			p.Line(fmt.Sprintf("+%d more", len(constructionItems)-shown), utils.Theme.TextDim)
		}
	}

	// Ships summary
	if len(human.OwnedShips) > 0 {
		p.Sep()
		p.Line(fmt.Sprintf("Ships (%d)", len(human.OwnedShips)), utils.Theme.Accent)
		shown := 0
		for _, ship := range human.OwnedShips {
			if ship == nil || shown >= 4 {
				continue
			}
			status := string(ship.Status)
			statusColor := utils.Theme.TextDim
			label := fmt.Sprintf("%s", ship.Name)
			switch ship.Status {
			case entities.ShipStatusMoving:
				status = "in transit"
				statusColor = utils.SystemGreen
			case entities.ShipStatusOrbiting:
				status = "orbiting"
			case entities.ShipStatusDocked:
				status = "docked"
			case entities.ShipStatusIdle:
				status = "idle"
			}
			if ship.GetTotalCargo() > 0 {
				status += fmt.Sprintf(" [%d]", ship.GetTotalCargo())
			}
			p.LinePair(label, utils.Theme.TextLight, status, statusColor)
			shown++
		}
		if len(human.OwnedShips) > shown {
			p.Line(fmt.Sprintf("+%d more", len(human.OwnedShips)-shown), utils.Theme.TextDim)
		}
	}

	p.Draw(screen)

	// Update click hit regions from panel bounds
	px, py, pw, _ := p.GetBounds()
	lh := widgets.LineH()
	a.empirePlanetHits = a.empirePlanetHits[:0]
	hitY := py + lh*3 // skip header + pop + separator
	for _, planet := range human.OwnedPlanets {
		if planet == nil {
			continue
		}
		hitStart := hitY
		hitY += lh // name
		hitY += lh // pop+happy
		if planet.PowerConsumed > 0 || planet.PowerGenerated > 0 {
			hitY += lh + 4 // power bar
		}
		if planet.Population > 0 && planet.GetStoredAmount("Water") < 10 {
			hitY += lh // warning
		} else if (planet.PowerGenerated > 0 || planet.PowerConsumed > 0) && planet.GetStoredAmount("Fuel") < 5 {
			hitY += lh // warning
		}
		a.empirePlanetHits = append(a.empirePlanetHits, empirePlanetHit{
			PlanetID: planet.GetID(),
			Y1: hitStart, Y2: hitY,
			X1: px, X2: px + pw,
		})
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
