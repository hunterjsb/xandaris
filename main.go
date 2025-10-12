package main

import (
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	_ "github.com/hunterjsb/xandaris/entities/planet"
	_ "github.com/hunterjsb/xandaris/entities/station"
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

// Game implements ebiten.Game interface
type Game struct {
	systems     []*System
	hyperlanes  []Hyperlane
	viewManager *ViewManager
	seed        int64
}

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		systems:    make([]*System, 0),
		hyperlanes: make([]Hyperlane, 0),
		seed:       time.Now().UnixNano(),
	}

	// Generate galaxy data
	g.generateSystems()
	g.generateHyperlanes()

	// Initialize view system
	g.viewManager = NewViewManager(g)

	// Create and register views
	galaxyView := NewGalaxyView(g)
	systemView := NewSystemView(g)

	g.viewManager.RegisterView(galaxyView)
	g.viewManager.RegisterView(systemView)

	// Start with galaxy view
	g.viewManager.SwitchTo(ViewTypeGalaxy)

	return g
}

// Update updates the game state
func (g *Game) Update() error {
	return g.viewManager.Update()
}

// Draw draws the game screen
func (g *Game) Draw(screen *ebiten.Image) {
	g.viewManager.Draw(screen)
}

// Layout returns the game's screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Xandaris II - Space Trading Game")

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
