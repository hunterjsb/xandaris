package main

import (
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	_ "github.com/hunterjsb/xandaris/entities/planet"
	_ "github.com/hunterjsb/xandaris/entities/resource"
	_ "github.com/hunterjsb/xandaris/entities/star"
	_ "github.com/hunterjsb/xandaris/entities/station"
	"github.com/hunterjsb/xandaris/tickable"
	_ "github.com/hunterjsb/xandaris/tickable" // Import tickable systems for auto-registration
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

// GameSystemContext implements tickable.SystemContext
type GameSystemContext struct {
	game *Game
}

func (gsc *GameSystemContext) GetGame() interface{} {
	return gsc.game
}

func (gsc *GameSystemContext) GetPlayers() interface{} {
	return gsc.game.players
}

func (gsc *GameSystemContext) GetTick() int64 {
	return gsc.game.tickManager.GetCurrentTick()
}

// Game implements ebiten.Game interface
type Game struct {
	systems     []*System
	hyperlanes  []Hyperlane
	viewManager *ViewManager
	seed        int64
	players     []*Player
	humanPlayer *Player
	tickManager *TickManager
}

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		systems:    make([]*System, 0),
		hyperlanes: make([]Hyperlane, 0),
		seed:       time.Now().UnixNano(),
		players:    make([]*Player, 0),
	}

	// Initialize tick system (10 ticks per second at 1x speed)
	g.tickManager = NewTickManager(10.0)

	// Generate galaxy data
	g.generateSystems()
	g.generateHyperlanes()

	// Create human player
	playerColor := color.RGBA{100, 200, 100, 255} // Green for player
	g.humanPlayer = NewPlayer(0, "Player", playerColor, PlayerTypeHuman)
	g.players = append(g.players, g.humanPlayer)

	// Initialize player with starting planet
	g.InitializePlayer(g.humanPlayer)

	// Register player as tick listener for resource production
	g.tickManager.AddListener(g.humanPlayer)

	// Initialize tickable systems
	context := &GameSystemContext{game: g}
	tickable.InitializeAllSystems(context)

	// Initialize view system
	g.viewManager = NewViewManager(g)

	// Create and register views
	galaxyView := NewGalaxyView(g)
	systemView := NewSystemView(g)
	planetView := NewPlanetView(g)

	g.viewManager.RegisterView(galaxyView)
	g.viewManager.RegisterView(systemView)
	g.viewManager.RegisterView(planetView)

	// Start with galaxy view
	g.viewManager.SwitchTo(ViewTypeGalaxy)

	return g
}

// Update updates the game state
func (g *Game) Update() error {
	// Handle global keyboard shortcuts
	g.handleGlobalInput()

	// Update tick system (this will also update tickable systems)
	g.tickManager.Update()

	// Update current view
	return g.viewManager.Update()
}

// handleGlobalInput handles keyboard input for game-wide controls
func (g *Game) handleGlobalInput() {
	// Space to toggle pause
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.tickManager.TogglePause()
	}

	// Number keys for speed control
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.tickManager.SetSpeed(TickSpeed1x)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.tickManager.SetSpeed(TickSpeed2x)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		g.tickManager.SetSpeed(TickSpeed4x)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		g.tickManager.SetSpeed(TickSpeed8x)
	}

	// Plus/Minus to cycle speed
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
		g.tickManager.CycleSpeed()
	}
}

// Draw draws the game screen
func (g *Game) Draw(screen *ebiten.Image) {
	g.viewManager.Draw(screen)

	// Draw tick info overlay
	g.drawTickInfo(screen)
}

// drawTickInfo draws tick information overlay
func (g *Game) drawTickInfo(screen *ebiten.Image) {
	// Draw in bottom-left corner
	x := 10
	y := screenHeight - 60

	// Create small panel
	panel := NewUIPanel(x, y, 200, 50)
	panel.Draw(screen)

	// Draw tick info
	textX := x + 10
	textY := y + 15

	speedStr := g.tickManager.GetSpeedString()
	DrawText(screen, "Speed: "+speedStr, textX, textY, UITextPrimary)
	DrawText(screen, g.tickManager.GetGameTimeFormatted(), textX, textY+15, UITextSecondary)
	DrawText(screen, "[Space] Pause  [1-4] Speed", textX, textY+30, UITextSecondary)
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
