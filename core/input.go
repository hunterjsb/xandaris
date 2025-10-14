package core

import (
	"fmt"

	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/views"
)

// handleGlobalInput handles keyboard input for game-wide controls
func (a *App) handleGlobalInput() {
	// Don't handle game controls in main menu
	if a.viewManager.GetCurrentView().GetType() == views.ViewTypeMainMenu {
		return
	}

	// Toggle pause
	if a.keyBindings.IsActionJustPressed(views.ActionPauseToggle) {
		a.tickManager.TogglePause()
	}

	// Speed control
	if a.keyBindings.IsActionJustPressed(views.ActionSpeedSlow) {
		a.tickManager.SetSpeed(systems.TickSpeed1x)
	}
	if a.keyBindings.IsActionJustPressed(views.ActionSpeedNormal) {
		a.tickManager.SetSpeed(systems.TickSpeed2x)
	}
	if a.keyBindings.IsActionJustPressed(views.ActionSpeedFast) {
		a.tickManager.SetSpeed(systems.TickSpeed4x)
	}
	if a.keyBindings.IsActionJustPressed(views.ActionSpeedVeryFast) {
		a.tickManager.SetSpeed(systems.TickSpeed8x)
	}

	// Cycle speed
	if a.keyBindings.IsActionJustPressed(views.ActionSpeedIncrease) {
		a.tickManager.CycleSpeed()
	}

	// Quick save
	if a.keyBindings.IsActionJustPressed(views.ActionQuickSave) {
		if a.state.HumanPlayer != nil {
			err := a.SaveGameToFile(a.state.HumanPlayer.Name)
			if err != nil {
				fmt.Printf("Failed to save game: %v\n", err)
			} else {
				fmt.Println("Game saved successfully!")
			}
		}
	}
}
