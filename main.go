package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

// Game implements ebiten.Game interface
type Game struct {
	systems        []*System
	hyperlanes     []Hyperlane
	selectedSystem *System
	contextMenu    *ContextMenu
}

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		systems:    make([]*System, 0),
		hyperlanes: make([]Hyperlane, 0),
	}

	g.generateSystems()
	g.generateHyperlanes()

	return g
}

// Update updates the game state
func (g *Game) Update() error {
	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		g.handleClick(x, y)
	}
	return nil
}

// handleClick checks if a system was clicked
func (g *Game) handleClick(x, y int) {
	// Check each system to see if the click was within its radius
	for _, system := range g.systems {
		dx := float64(x) - system.X
		dy := float64(y) - system.Y
		distance := math.Sqrt(dx*dx + dy*dy)

		if distance <= float64(circleRadius) {
			// Toggle selection - if already selected, deselect
			if g.selectedSystem == system {
				g.selectedSystem = nil
				g.contextMenu = nil
			} else {
				g.selectedSystem = system
				g.createContextMenuForSystem(system)
			}
			return
		}
	}
	// If we didn't click on any system, deselect
	g.selectedSystem = nil
	g.contextMenu = nil
}

// createContextMenuForSystem creates and positions a context menu for the given system
func (g *Game) createContextMenuForSystem(system *System) {
	items := []string{}

	// Add entity counts summary
	planetCount := len(system.GetEntitiesByType(EntityTypePlanet))
	stationCount := len(system.GetEntitiesByType(EntityTypeStation))

	items = append(items, fmt.Sprintf("Planets: %d", planetCount))
	if stationCount > 0 {
		items = append(items, fmt.Sprintf("Stations: %d", stationCount))
	}
	items = append(items, "") // Empty line for spacing

	// List planets
	for _, entity := range system.GetEntitiesByType(EntityTypePlanet) {
		items = append(items, fmt.Sprintf("  - %s", entity.GetDescription()))
	}

	// List stations
	for _, entity := range system.GetEntitiesByType(EntityTypeStation) {
		items = append(items, fmt.Sprintf("  - %s", entity.GetDescription()))
	}

	g.contextMenu = NewContextMenu(system.Name, items)
	g.contextMenu.PositionNear(
		int(system.X),
		int(system.Y),
		circleRadius+20,
	)
}

// Draw draws the game screen
func (g *Game) Draw(screen *ebiten.Image) {
	// Fill background
	screen.Fill(color.RGBA{5, 5, 15, 255})

	// Draw hyperlanes first (so they appear behind systems)
	g.drawHyperlanes(screen)

	// Draw all systems
	for _, system := range g.systems {
		g.drawSystem(screen, system)
	}

	// Highlight selected system
	if g.selectedSystem != nil {
		DrawHighlightCircle(screen,
			int(g.selectedSystem.X),
			int(g.selectedSystem.Y),
			circleRadius,
			color.RGBA{255, 255, 100, 255})
	}

	// Draw context menu if a system is selected
	if g.contextMenu != nil {
		g.contextMenu.Draw(screen)
	}

	// Draw UI info
	ebitenutil.DebugPrint(screen, "Xandaris II - Galaxy Map with Hyperlanes\nPress ESC to quit")
}

// drawHyperlanes draws connections between systems
func (g *Game) drawHyperlanes(screen *ebiten.Image) {
	hyperlaneColor := color.RGBA{40, 40, 80, 255}

	for _, hyperlane := range g.hyperlanes {
		fromSystem := g.systems[hyperlane.From]
		toSystem := g.systems[hyperlane.To]

		// Draw line between systems
		DrawLine(screen,
			int(fromSystem.X), int(fromSystem.Y),
			int(toSystem.X), int(toSystem.Y),
			hyperlaneColor)
	}
}

// drawSystem renders a single system
func (g *Game) drawSystem(screen *ebiten.Image, system *System) {
	centerX := int(system.X)
	centerY := int(system.Y)

	// Create a circular image for the system
	circleImg := ebiten.NewImage(circleRadius*2, circleRadius*2)

	// Draw a circle by filling pixels within the radius
	for py := 0; py < circleRadius*2; py++ {
		for px := 0; px < circleRadius*2; px++ {
			// Calculate distance from center
			dx := float64(px - circleRadius)
			dy := float64(py - circleRadius)
			dist := dx*dx + dy*dy

			// If within radius, set pixel to system color
			if dist <= float64(circleRadius*circleRadius) {
				circleImg.Set(px, py, system.Color)
			}
		}
	}

	// Draw the circle centered
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(centerX-circleRadius), float64(centerY-circleRadius))
	screen.DrawImage(circleImg, opts)

	// Draw centered label below the circle
	labelY := centerY + circleRadius + 15
	DrawCenteredText(screen, system.Name, centerX, labelY)
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
