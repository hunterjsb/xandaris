package views

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hunterjsb/xandaris/utils"
)

var (
	DefaultFontFace *text.GoXFace
	rectCache       = utils.NewRectImageCache()
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

// DrawRectOutline draws a rectangle outline with the specified color
func DrawRectOutline(screen *ebiten.Image, x, y, width, height int, c color.RGBA) {
	if width <= 0 || height <= 0 {
		return
	}
	DrawLine(screen, x, y, x+width-1, y, c)
	DrawLine(screen, x, y+height-1, x+width-1, y+height-1, c)
	DrawLine(screen, x, y, x, y+height-1, c)
	DrawLine(screen, x+width-1, y, x+width-1, y+height-1, c)
}

// DrawTextCenteredInRect draws text centered within the given rectangle
func DrawTextCenteredInRect(screen *ebiten.Image, textStr string, rect image.Rectangle, textColor color.RGBA) {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}

	const (
		charWidth      = 6
		charHeight     = 12
		baselineAdjust = 2
	)

	textWidth := len(textStr) * charWidth
	textX := rect.Min.X + (rect.Dx()-textWidth)/2
	textY := rect.Min.Y + (rect.Dy()+charHeight)/2 - baselineAdjust - 10

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(textX), float64(textY))
	op.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, textStr, DefaultFontFace, op)
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

// DrawGlow draws a glow effect around a point
func DrawGlow(screen *ebiten.Image, centerX, centerY int, radius float64, glowColor color.RGBA) {
	segments := 32
	for i := 0; i < segments; i++ {
		angle1 := float64(i) * 2 * 3.14159 / float64(segments)
		angle2 := float64(i+1) * 2 * 3.14159 / float64(segments)

		x1 := centerX + int(radius*math.Cos(angle1))
		y1 := centerY + int(radius*math.Sin(angle1))
		x2 := centerX + int(radius*math.Cos(angle2))
		y2 := centerY + int(radius*math.Sin(angle2))

		DrawLine(screen, x1, y1, x2, y2, glowColor)
	}
}

// UIProgressBar renders a horizontal progress indicator with border
type UIProgressBar struct {
	X, Y          int
	Width, Height int
	Value, Max    float64
	FillColor     color.RGBA
	BgColor       color.RGBA
	BorderColor   color.RGBA
	cachedBg      *ebiten.Image
	cachedFill    *ebiten.Image
}

// NewUIProgressBar constructs a progress bar with sensible defaults
func NewUIProgressBar(x, y, width, height int) *UIProgressBar {
	return &UIProgressBar{
		X:           x,
		Y:           y,
		Width:       width,
		Height:      height,
		Value:       0,
		Max:         1,
		FillColor:   utils.PlayerGreen,
		BgColor:     color.RGBA{20, 20, 40, 255},
		BorderColor: utils.PanelBorder,
	}
}

// DrawLabeledButton renders a rectangular button with centered text.
func DrawLabeledButton(screen *ebiten.Image, rect image.Rectangle, label string, active bool) {
	panel := &UIPanel{
		X:           rect.Min.X,
		Y:           rect.Min.Y,
		Width:       rect.Dx(),
		Height:      rect.Dy(),
		BgColor:     utils.PanelBg,
		BorderColor: utils.PanelBorder,
	}
	if active {
		panel.BgColor = utils.ButtonActive
	}
	panel.Draw(screen)

	textColor := utils.TextSecondary
	if active {
		textColor = utils.TextPrimary
	}
	DrawTextCenteredInRect(screen, label, rect, textColor)
}

// ChartSegment describes a slice in a stacked bar or legend.
type ChartSegment struct {
	Label string
	Value float64
	Color color.RGBA
}

// DrawStackedBar draws a horizontal stacked bar chart inside the given rectangle.
func DrawStackedBar(screen *ebiten.Image, rect image.Rectangle, segments []ChartSegment, background, border color.RGBA) {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}

	bar := rectCache.GetOrCreate(rect.Dx(), rect.Dy(), background)

	total := 0.0
	for _, seg := range segments {
		if seg.Value > 0 {
			total += seg.Value
		}
	}

	if total > 0 {
		offset := 0
		for idx, seg := range segments {
			if seg.Value <= 0 {
				continue
			}
			ratio := seg.Value / total
			segWidth := int(ratio * float64(rect.Dx()))
			if segWidth <= 0 {
				// ensure at least 1px for visible segments, but don't overflow the bar width
				if idx == len(segments)-1 {
					segWidth = rect.Dx() - offset
				} else {
					segWidth = 1
				}
			}
			if offset+segWidth > rect.Dx() {
				segWidth = rect.Dx() - offset
			}
			if segWidth <= 0 {
				continue
			}

			segImg := rectCache.GetOrCreate(segWidth, rect.Dy(), seg.Color)
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(offset), 0)
			bar.DrawImage(segImg, opts)
			offset += segWidth
			if offset >= rect.Dx() {
				break
			}
		}
	}

	barOpts := &ebiten.DrawImageOptions{}
	barOpts.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	screen.DrawImage(bar, barOpts)
	DrawRectOutline(screen, rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy(), border)
}

// DrawLegend renders a vertical legend for the provided segments.
// It returns the y-coordinate immediately following the drawn legend.
func DrawLegend(screen *ebiten.Image, start image.Point, segments []ChartSegment) int {
	currentY := start.Y
	for _, seg := range segments {
		if seg.Value <= 0 {
			continue
		}
		drawColorSwatch(screen, start.X, currentY, seg.Color)
		label := fmt.Sprintf("%s (%s)", seg.Label, utils.FormatInt64WithCommas(int64(seg.Value+0.5)))
		DrawText(screen, label, start.X+18, currentY+12, utils.TextSecondary)
		currentY += 20
	}
	return currentY
}

// drawColorSwatch draws a small filled square with a border for use in legends.
func drawColorSwatch(screen *ebiten.Image, x, y int, c color.RGBA) {
	swatch := rectCache.GetOrCreate(12, 12, c)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(swatch, opts)
	DrawRectOutline(screen, x, y, 12, 12, utils.PanelBorder)
}

// SetValue updates the current value and maximum for the progress bar
func (pb *UIProgressBar) SetValue(value, max float64) {
	pb.Value = value
	pb.Max = max
}

// Draw renders the progress bar
func (pb *UIProgressBar) Draw(screen *ebiten.Image) {
	if pb.Width <= 0 || pb.Height <= 0 {
		return
	}

	if pb.Max == 0 {
		pb.Max = 1
	}

	// Cache background image
	if pb.cachedBg == nil {
		pb.cachedBg = ebiten.NewImage(pb.Width, pb.Height)
		pb.cachedBg.Fill(pb.BgColor)
	}

	bgOpts := &ebiten.DrawImageOptions{}
	bgOpts.GeoM.Translate(float64(pb.X), float64(pb.Y))
	screen.DrawImage(pb.cachedBg, bgOpts)

	if pb.Value > 0 && pb.Max > 0 {
		ratio := pb.Value / pb.Max
		if ratio > 1 {
			ratio = 1
		}
		fillWidth := int(ratio * float64(pb.Width))
		if fillWidth > 0 {
			// Reuse or create fill image
			if pb.cachedFill == nil || pb.cachedFill.Bounds().Dx() != fillWidth {
				if pb.cachedFill != nil {
					pb.cachedFill.Deallocate()
				}
				pb.cachedFill = ebiten.NewImage(fillWidth, pb.Height)
			}
			pb.cachedFill.Clear()
			pb.cachedFill.Fill(pb.FillColor)

			fillOpts := &ebiten.DrawImageOptions{}
			fillOpts.GeoM.Translate(float64(pb.X), float64(pb.Y))
			screen.DrawImage(pb.cachedFill, fillOpts)
		}
	}

	DrawRectOutline(screen, pb.X, pb.Y, pb.Width, pb.Height, pb.BorderColor)
}
