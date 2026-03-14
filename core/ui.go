package core

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hunterjsb/xandaris/views"
)

// drawTickInfo draws game status overlay in bottom-left corner
func (a *App) drawTickInfo(screen *ebiten.Image) {
	if a.viewManager.GetCurrentView().GetType() == views.ViewTypeMainMenu || a.Server.TickManager == nil {
		return
	}

	x := 10
	y := a.screenHeight - 55

	panel := views.NewUIPanel(x, y, 290, 45)
	panel.Draw(screen)

	textX := x + 10
	textY := y + 12

	// Speed and game time
	speedStr := a.Server.TickManager.GetSpeedString()
	pauseStr := ""
	if a.Server.TickManager.IsPaused() {
		pauseStr = " [PAUSED]"
	}
	views.DrawText(screen, fmt.Sprintf("Speed: %s  %s%s", speedStr, a.Server.TickManager.GetGameTimeFormatted(), pauseStr), textX, textY, utils.TextPrimary)

	// Credits with net income indicator + controls
	credStr := ""
	if human := a.Server.State.HumanPlayer; human != nil {
		// Calculate net credit flow per interval
		income := 0
		upkeep := 0
		for _, planet := range human.OwnedPlanets {
			if planet == nil {
				continue
			}
			income += int(planet.Population / 100) // base income
			upkeep += int(planet.Population / 1000) // admin
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
		netStr := fmt.Sprintf(" (%+d)", net)
		credStr = fmt.Sprintf("Credits: %d%s  ", human.Credits, netStr)
	}
	views.DrawText(screen, credStr+"[Space] Pause  [F5] Save", textX, textY+15, utils.TextSecondary)
}
