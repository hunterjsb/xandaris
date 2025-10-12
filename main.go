package main

import (
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

// Game implements ebiten.Game interface
type Game struct {
	systems    []*System
	hyperlanes []Hyperlane
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
	return nil
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
		g.drawLine(screen,
			int(fromSystem.X), int(fromSystem.Y),
			int(toSystem.X), int(toSystem.Y),
			hyperlaneColor)
	}
}

// drawLine draws a simple line between two points
func (g *Game) drawLine(screen *ebiten.Image, x1, y1, x2, y2 int, c color.RGBA) {
	dx := x2 - x1
	dy := y2 - y1

	steps := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy))))

	if steps == 0 {
		return
	}

	xStep := float64(dx) / float64(steps)
	yStep := float64(dy) / float64(steps)

	for i := 0; i <= steps; i++ {
		x := x1 + int(float64(i)*xStep)
		y := y1 + int(float64(i)*yStep)

		if x >= 0 && x < screenWidth && y >= 0 && y < screenHeight {
			screen.Set(x, y, c)
		}
	}
}

// drawCenteredText draws text centered at the given position
func drawCenteredText(screen *ebiten.Image, text string, x, y int) {
	// Approximate text width (each character is about 6 pixels wide)
	textWidth := len(text) * 6
	ebitenutil.DebugPrintAt(screen, text, x-textWidth/2, y)
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
	drawCenteredText(screen, system.Name, centerX, labelY)
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
