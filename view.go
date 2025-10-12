package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// ViewType represents the type of view
type ViewType string

const (
	ViewTypeGalaxy ViewType = "Galaxy"
	ViewTypeSystem ViewType = "System"
)

// View interface that all game views must implement
type View interface {
	Update() error
	Draw(screen *ebiten.Image)
	OnEnter()
	OnExit()
	GetType() ViewType
}

// ViewManager manages switching between different views
type ViewManager struct {
	currentView View
	views       map[ViewType]View
	game        *Game
}

// NewViewManager creates a new view manager
func NewViewManager(game *Game) *ViewManager {
	vm := &ViewManager{
		views: make(map[ViewType]View),
		game:  game,
	}
	return vm
}

// RegisterView adds a view to the manager
func (vm *ViewManager) RegisterView(view View) {
	vm.views[view.GetType()] = view
}

// SwitchTo switches to a different view
func (vm *ViewManager) SwitchTo(viewType ViewType) {
	if vm.currentView != nil {
		vm.currentView.OnExit()
	}

	newView, exists := vm.views[viewType]
	if !exists {
		return
	}

	vm.currentView = newView
	vm.currentView.OnEnter()
}

// GetCurrentView returns the active view
func (vm *ViewManager) GetCurrentView() View {
	return vm.currentView
}

// Update updates the current view
func (vm *ViewManager) Update() error {
	if vm.currentView != nil {
		return vm.currentView.Update()
	}
	return nil
}

// Draw draws the current view
func (vm *ViewManager) Draw(screen *ebiten.Image) {
	if vm.currentView != nil {
		vm.currentView.Draw(screen)
	}
}
