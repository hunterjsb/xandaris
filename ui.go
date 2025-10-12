package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// UIPanel represents a rectangular panel with border
type UIPanel struct {
	X           int
	Y           int
	Width       int
	Height      int
	BgColor     color.RGBA
	BorderColor color.RGBA
}

// NewUIPanel creates a new UI panel
func NewUIPanel(x, y, width, height int) *UIPanel {
	return &UIPanel{
		X:           x,
		Y:           y,
		Width:       width,
		Height:      height,
		BgColor:     color.RGBA{20, 20, 40, 230},
		BorderColor: color.RGBA{100, 100, 150, 255},
	}
}

// Draw renders the panel to the screen
func (p *UIPanel) Draw(screen *ebiten.Image) {
	// Create panel image
	panelImg := ebiten.NewImage(p.Width, p.Height)
	panelImg.Fill(p.BgColor)

	// Draw border
	for i := 0; i < p.Width; i++ {
		panelImg.Set(i, 0, p.BorderColor)
		panelImg.Set(i, p.Height-1, p.BorderColor)
	}
	for i := 0; i < p.Height; i++ {
		panelImg.Set(0, i, p.BorderColor)
		panelImg.Set(p.Width-1, i, p.BorderColor)
	}

	// Draw to screen
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(p.X), float64(p.Y))
	screen.DrawImage(panelImg, opts)
}

// Contains checks if a point is within the panel bounds
func (p *UIPanel) Contains(x, y int) bool {
	return x >= p.X && x < p.X+p.Width && y >= p.Y && y < p.Y+p.Height
}

// ContextMenu represents a popup menu that appears near an entity
type ContextMenu struct {
	Panel   *UIPanel
	Title   string
	Items   []string
	Padding int
}

// NewContextMenu creates a new context menu
func NewContextMenu(title string, items []string) *ContextMenu {
	padding := 10
	width := 200
	height := padding*2 + 20 + len(items)*15 + 20 // title + separator + items + extra bottom padding

	return &ContextMenu{
		Panel:   NewUIPanel(0, 0, width, height),
		Title:   title,
		Items:   items,
		Padding: padding,
	}
}

// PositionNear positions the context menu near a target point
// Automatically adjusts to stay within screen bounds
func (c *ContextMenu) PositionNear(targetX, targetY, offset int) {
	// Try to position to the right of the target
	c.Panel.X = targetX + offset
	c.Panel.Y = targetY - c.Panel.Height/2

	// Adjust if too close to right edge
	if c.Panel.X+c.Panel.Width > screenWidth-10 {
		c.Panel.X = targetX - offset - c.Panel.Width
	}

	// Adjust if too close to top edge
	if c.Panel.Y < 10 {
		c.Panel.Y = 10
	}

	// Adjust if too close to bottom edge
	if c.Panel.Y+c.Panel.Height > screenHeight-10 {
		c.Panel.Y = screenHeight - c.Panel.Height - 10
	}
}

// Draw renders the context menu
func (c *ContextMenu) Draw(screen *ebiten.Image) {
	// Draw panel background
	c.Panel.Draw(screen)

	// Draw title
	textY := c.Panel.Y + c.Padding
	ebitenutil.DebugPrintAt(screen, c.Title, c.Panel.X+c.Padding, textY)

	// Draw separator
	textY += 20
	ebitenutil.DebugPrintAt(screen, "─────────────────", c.Panel.X+c.Padding, textY)

	// Draw items
	textY += 20
	for _, item := range c.Items {
		ebitenutil.DebugPrintAt(screen, item, c.Panel.X+c.Padding, textY)
		textY += 15
	}
}

// DrawCenteredText draws text centered at the given position
func DrawCenteredText(screen *ebiten.Image, text string, x, y int) {
	// Approximate text width (each character is about 6 pixels wide)
	textWidth := len(text) * 6
	ebitenutil.DebugPrintAt(screen, text, x-textWidth/2, y)
}

// DrawHighlightCircle draws a highlight ring around a circular object
func DrawHighlightCircle(screen *ebiten.Image, centerX, centerY, radius int, highlightColor color.RGBA) {
	highlightRadius := radius + 4

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

// DrawLine draws a simple line between two points
func DrawLine(screen *ebiten.Image, x1, y1, x2, y2 int, c color.RGBA) {
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
