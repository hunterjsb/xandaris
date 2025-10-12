package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 1280
	screenHeight = 720
	gridWidth    = 10
	gridHeight   = 6
	circleRadius = 12
)

var (
	cellSize int
)

func init() {
	// Calculate cell size to fit the grid perfectly in the window
	// Leave some margin for UI text and padding
	availableWidth := screenWidth - 40   // 20px margin on each side
	availableHeight := screenHeight - 80 // 40px top margin for text, 40px bottom

	cellSizeByWidth := availableWidth / gridWidth
	cellSizeByHeight := availableHeight / gridHeight

	// Use the smaller dimension to ensure everything fits
	if cellSizeByWidth < cellSizeByHeight {
		cellSize = cellSizeByWidth
	} else {
		cellSize = cellSizeByHeight
	}
}

// System represents a star system
type System struct {
	X     int
	Y     int
	Name  string
	Color color.RGBA
}

// Game implements ebiten.Game interface
type Game struct {
	systems [][]*System
}

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		systems: make([][]*System, gridHeight),
	}

	// Initialize the system matrix
	for y := 0; y < gridHeight; y++ {
		g.systems[y] = make([]*System, gridWidth)
		for x := 0; x < gridWidth; x++ {
			g.systems[y][x] = g.generateSystem(x, y)
		}
	}

	return g
}

// generateSystem creates a new system
func (g *Game) generateSystem(x, y int) *System {
	colors := []color.RGBA{
		{100, 100, 200, 255}, // Blue
		{200, 100, 150, 255}, // Purple
		{150, 200, 100, 255}, // Green
		{200, 150, 100, 255}, // Orange
		{200, 200, 100, 255}, // Yellow
		{200, 100, 100, 255}, // Red
	}

	return &System{
		X:     x,
		Y:     y,
		Name:  fmt.Sprintf("%d,%d", x, y),
		Color: colors[rand.Intn(len(colors))],
	}
}

// Update updates the game state
func (g *Game) Update() error {
	return nil
}

// Draw draws the game screen
func (g *Game) Draw(screen *ebiten.Image) {
	// Fill background
	screen.Fill(color.RGBA{10, 10, 20, 255})

	// Calculate starting position to center the grid
	gridPixelWidth := gridWidth * cellSize
	gridPixelHeight := gridHeight * cellSize
	startX := (screenWidth - gridPixelWidth) / 2
	startY := (screenHeight - gridPixelHeight) / 2

	// Draw all systems
	for y := 0; y < gridHeight; y++ {
		for x := 0; x < gridWidth; x++ {
			system := g.systems[y][x]
			g.drawSystem(screen, system, startX, startY)
		}
	}

	// Draw UI info
	ebitenutil.DebugPrint(screen, "Xandaris II - Galaxy Map\nPress ESC to quit")
}

// drawCenteredText draws text centered at the given position
func drawCenteredText(screen *ebiten.Image, text string, x, y int) {
	// Approximate text width (each character is about 6 pixels wide)
	textWidth := len(text) * 6
	ebitenutil.DebugPrintAt(screen, text, x-textWidth/2, y)
}

// drawSystem renders a single system
func (g *Game) drawSystem(screen *ebiten.Image, system *System, offsetX, offsetY int) {
	centerX := offsetX + system.X*cellSize + cellSize/2
	centerY := offsetY + system.Y*cellSize + cellSize/2

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

	// Draw coordinate in top-right of cell
	ebitenutil.DebugPrintAt(screen, system.Name, centerX-cellSize/2+5, centerY-cellSize/2+5)

	// Draw centered label at bottom of circle
	labelY := centerY + circleRadius + 10
	drawCenteredText(screen, fmt.Sprintf("SYS-%s", system.Name), centerX, labelY)
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
