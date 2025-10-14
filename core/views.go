package core

import (
	"github.com/hunterjsb/xandaris/ui"
	"github.com/hunterjsb/xandaris/views"
)

// initializeViews creates and registers all game views
// This must be called after UI components are created
func (a *App) initializeViews() {

	// Create and register views (pass App as GameContext)
	mainMenuView := views.NewMainMenuView(a)
	settingsView := views.NewSettingsView(a)

	a.viewManager.RegisterView(mainMenuView)
	a.viewManager.RegisterView(settingsView)
}

// initializeGameViews creates and registers all game views
// This must be called after UI components are created
func (a *App) initializeGameViews(buildMenu *ui.BuildMenu, constructionQueue *ui.ConstructionQueueUI,
	resourceStorage *ui.ResourceStorageUI, shipyardUI *ui.ShipyardUI, fleetInfoUI *ui.FleetInfoUI) {

	// Create and register views (pass App as GameContext)
	galaxyView := views.NewGalaxyView(a)
	systemView := views.NewSystemView(a, fleetInfoUI)
	planetView := views.NewPlanetView(a, buildMenu, constructionQueue, resourceStorage, shipyardUI, fleetInfoUI)

	a.viewManager.RegisterView(galaxyView)
	a.viewManager.RegisterView(systemView)
	a.viewManager.RegisterView(planetView)
}
