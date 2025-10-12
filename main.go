package main

import (
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
			} else {
				g.selectedSystem = system
			}
			return
		}
	}
	// If we didn't click on any system, deselect
	g.selectedSystem = nil
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
		g.drawSystemHighlight(screen, g.selectedSystem)
	}

	// Draw context menu if a system is selected
	if g.selectedSystem != nil {
		g.drawContextMenu(screen, g.selectedSystem)
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

// drawSystemHighlight draws a highlight ring around the selected system
func (g *Game) drawSystemHighlight(screen *ebiten.Image, system *System) {
	centerX := int(system.X)
	centerY := int(system.Y)
	highlightRadius := circleRadius + 4

	// Draw a ring around the system
	highlightColor := color.RGBA{255, 255, 100, 255}

	for angle := 0.0; angle < 6.28; angle += 0.1 {
		x := centerX + int(float64(highlightRadius)*math.Cos(angle))
		y := centerY + int(float64(highlightRadius)*math.Sin(angle))
		if x >= 0 && x < screenWidth && y >= 0 && y < screenHeight {
			screen.Set(x, y, highlightColor)
			screen.Set(x+1, y, highlightColor)
			screen.Set(x, y+1, highlightColor)
		}
	}
}

// drawContextMenu draws a context menu near the selected system
func (g *Game) drawContextMenu(screen *ebiten.Image, system *System) {
	menuWidth := 200
	menuHeight := 120
	padding := 10

	// Position menu to the right of the system, or left if too close to edge
	menuX := int(system.X) + circleRadius + 20
	menuY := int(system.Y) - menuHeight/2

	// Keep menu on screen
	if menuX+menuWidth > screenWidth-10 {
		menuX = int(system.X) - circleRadius - menuWidth - 20
	}
	if menuY < 10 {
		menuY = 10
	}
	if menuY+menuHeight > screenHeight-10 {
		menuY = screenHeight - menuHeight - 10
	}

	// Draw menu background
	menuImg := ebiten.NewImage(menuWidth, menuHeight)
	menuImg.Fill(color.RGBA{20, 20, 40, 230})

	// Draw border
	borderColor := color.RGBA{100, 100, 150, 255}
	for i := 0; i < menuWidth; i++ {
		menuImg.Set(i, 0, borderColor)
		menuImg.Set(i, menuHeight-1, borderColor)
	}
	for i := 0; i < menuHeight; i++ {
		menuImg.Set(0, i, borderColor)
		menuImg.Set(menuWidth-1, i, borderColor)
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(menuX), float64(menuY))
	screen.DrawImage(menuImg, opts)

	// Draw menu content
	textY := menuY + padding
	ebitenutil.DebugPrintAt(screen, system.Name, menuX+padding, textY)
	textY += 20
	ebitenutil.DebugPrintAt(screen, "─────────────────", menuX+padding, textY)
	textY += 20
	ebitenutil.DebugPrintAt(screen, "Planets: Coming soon", menuX+padding, textY)
	textY += 15
	ebitenutil.DebugPrintAt(screen, "Resources: TBD", menuX+padding, textY)
	textY += 15
	ebitenutil.DebugPrintAt(screen, "Population: TBD", menuX+padding, textY)
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
