package core

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hunterjsb/xandaris/views"
)

// drawTickInfo draws tick information overlay
func (a *App) drawTickInfo(screen *ebiten.Image) {
	// Don't draw in main menu
	if a.viewManager.GetCurrentView().GetType() == views.ViewTypeMainMenu {
		return
	}

	// Draw in bottom-left corner
	x := 10
	y := a.screenHeight - 60

	// Create small panel
	panel := views.NewUIPanel(x, y, 200, 50)
	panel.Draw(screen)

	// Draw tick info
	textX := x + 10
	textY := y + 15

	speedStr := a.tickManager.GetSpeedString()
	views.DrawText(screen, "Speed: "+speedStr, textX, textY, utils.TextPrimary)
	views.DrawText(screen, a.tickManager.GetGameTimeFormatted(), textX, textY+15, utils.TextSecondary)
	views.DrawText(screen, "[Space] Pause  [F5] Save", textX, textY+30, utils.TextSecondary)
}
