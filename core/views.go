package core

import (
	"github.com/hunterjsb/xandaris/views"
)

// initializeViews creates and registers only the menu-related views
func (a *App) initializeViews() {
	mainMenuView := views.NewMainMenuView(a)
	settingsView := views.NewSettingsView(a)

	a.viewManager.RegisterView(mainMenuView)
	a.viewManager.RegisterView(settingsView)
}
