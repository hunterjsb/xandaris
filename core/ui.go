package core

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
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

	panel := views.NewUIPanel(x, y, 240, 45)
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

	// Credits + controls
	credStr := ""
	if a.Server.State.HumanPlayer != nil {
		credStr = fmt.Sprintf("Credits: %d  ", a.Server.State.HumanPlayer.Credits)
	}
	views.DrawText(screen, credStr+"[Space] Pause  [F5] Save", textX, textY+15, utils.TextSecondary)
}
