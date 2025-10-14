package views

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	DefaultFontFace *text.GoXFace
)

func init() {
	// Initialize default font face using bitmap font
	DefaultFontFace = text.NewGoXFace(bitmapfont.Face)
}

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
		BgColor:     utils.PanelBg,
		BorderColor: utils.PanelBorder,
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

// DrawText draws text in a specific color using text/v2
func DrawText(screen *ebiten.Image, textStr string, x, y int, textColor color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, textStr, DefaultFontFace, op)
}

// DrawCenteredText draws text centered at the given position
func DrawCenteredText(screen *ebiten.Image, textStr string, x, y int) {
	// Approximate text width (each character is about 6 pixels wide)
	textWidth := len(textStr) * 6
	DrawText(screen, textStr, x-textWidth/2, y, utils.TextPrimary)
}

// DrawTextCentered draws text centered at the given position with color and scale
func DrawTextCentered(screen *ebiten.Image, textStr string, x, y int, textColor color.RGBA, scale float64) {
	op := &text.DrawOptions{}

	// Calculate text bounds for centering
	bounds, _ := text.Measure(textStr, DefaultFontFace, 0)
	width := bounds * scale

	// Apply scale and center
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(x)-width/2, float64(y))
	op.ColorScale.ScaleWithColor(textColor)

	text.Draw(screen, textStr, DefaultFontFace, op)
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

		screen.Set(x, y, c)
	}
}

// DrawHighlightCircle draws a highlight ring around a circular object
func DrawHighlightCircle(screen *ebiten.Image, centerX, centerY, radius int, highlightColor color.RGBA) {
	highlightRadius := radius + 4

	for angle := 0.0; angle < 6.28; angle += 0.1 {
		x := centerX + int(float64(highlightRadius)*math.Cos(angle))
		y := centerY + int(float64(highlightRadius)*math.Sin(angle))
		if x >= 0 && x < ScreenWidth && y >= 0 && y < ScreenHeight {
			screen.Set(x, y, highlightColor)
			screen.Set(x+1, y, highlightColor)
			screen.Set(x, y+1, highlightColor)
		}
	}
}

// DrawOwnershipRing draws a colored ring around an object to indicate ownership
func DrawOwnershipRing(screen *ebiten.Image, centerX, centerY int, radius float64, ownerColor color.RGBA) {
	// Make the ring semi-transparent
	ringColor := ownerColor
	ringColor.A = 180

	// Draw the ring
	segments := 32
	for i := 0; i < segments; i++ {
		angle1 := float64(i) * 2 * 3.14159 / float64(segments)
		angle2 := float64(i+1) * 2 * 3.14159 / float64(segments)

		x1 := centerX + int(radius*math.Cos(angle1))
		y1 := centerY + int(radius*math.Sin(angle1))
		x2 := centerX + int(radius*math.Cos(angle2))
		y2 := centerY + int(radius*math.Sin(angle2))

		DrawLine(screen, x1, y1, x2, y2, ringColor)
	}
}
