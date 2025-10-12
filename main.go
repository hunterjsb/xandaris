package main

import (
	"log"

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
	systems      []*System
	hyperlanes   []Hyperlane
	clickHandler *ClickHandler
}

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		systems:      make([]*System, 0),
		hyperlanes:   make([]Hyperlane, 0),
		clickHandler: NewClickHandler(),
	}

	g.generateSystems()
	g.generateHyperlanes()

	// Add all systems as clickable objects
	for _, system := range g.systems {
		g.clickHandler.AddClickable(system)
	}

	return g
}

// Update updates the game state
func (g *Game) Update() error {
	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		g.clickHandler.HandleClick(x, y)
	}
	return nil
}

// Draw draws the game screen
func (g *Game) Draw(screen *ebiten.Image) {
	// Fill background
	screen.Fill(UIBackground)

	// Draw hyperlanes first (so they appear behind systems)
	g.drawHyperlanes(screen)

	// Draw all systems
	for _, system := range g.systems {
		g.drawSystem(screen, system)
	}

	// Highlight selected object
	if selectedObj := g.clickHandler.GetSelectedObject(); selectedObj != nil {
		x, y := selectedObj.GetPosition()
		DrawHighlightCircle(screen,
			int(x), int(y),
			int(selectedObj.GetClickRadius()),
			UIHighlight)
	}

	// Draw context menu if active
	if g.clickHandler.HasActiveMenu() {
		g.clickHandler.GetActiveMenu().Draw(screen)
	}

	// Draw UI info
	ebitenutil.DebugPrint(screen, "Xandaris II - Galaxy Map with Hyperlanes\nPress ESC to quit")
}

// drawHyperlanes draws connections between systems
func (g *Game) drawHyperlanes(screen *ebiten.Image) {
	hyperlaneColor := HyperlaneNormal

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
