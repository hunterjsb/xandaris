package ui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hunterjsb/xandaris/utils"
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
	cachedImage *ebiten.Image
	cachedKey   string
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
	// Generate cache key based on panel properties
	key := fmt.Sprintf("panel_%dx%d_%d_%d_%d_%d_%d_%d_%d_%d",
		p.Width, p.Height,
		p.BgColor.R, p.BgColor.G, p.BgColor.B, p.BgColor.A,
		p.BorderColor.R, p.BorderColor.G, p.BorderColor.B, p.BorderColor.A)

	// Check if we need to regenerate the cached image
	if p.cachedImage == nil || p.cachedKey != key {
		// Create panel image
		if p.cachedImage != nil {
			p.cachedImage.Clear()
		} else {
			p.cachedImage = ebiten.NewImage(p.Width, p.Height)
		}
		p.cachedImage.Fill(p.BgColor)

		// Draw border
		for i := 0; i < p.Width; i++ {
			p.cachedImage.Set(i, 0, p.BorderColor)
			p.cachedImage.Set(i, p.Height-1, p.BorderColor)
		}
		for i := 0; i < p.Height; i++ {
			p.cachedImage.Set(0, i, p.BorderColor)
			p.cachedImage.Set(p.Width-1, i, p.BorderColor)
		}
		p.cachedKey = key
	}

	// Draw to screen
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(p.X), float64(p.Y))
	screen.DrawImage(p.cachedImage, opts)
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
