// Package widgets provides reusable UI components with automatic scaling.
// All sizes are specified in character/line units, not pixels.
// The widgets handle conversion to pixels using utils.UIScale and CharWidth.
package widgets

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hunterjsb/xandaris/views"
)

// Anchor determines where a panel is pinned on screen.
type Anchor int

const (
	AnchorTopLeft Anchor = iota
	AnchorTopRight
	AnchorBottomLeft
	AnchorBottomRight
	AnchorCenter
	AnchorManual // use X/Y directly (in pixels)
)

// Align controls horizontal text alignment within a line.
type Align int

const (
	AlignLeft Align = iota
	AlignRight
	AlignCenter
)

// lineEntry represents one renderable line in a panel.
type lineEntry struct {
	text      string
	color     color.RGBA
	align     Align
	separator bool // draw a horizontal line instead of text
	bar       *barEntry
}

// barEntry represents a progress/capacity bar.
type barEntry struct {
	value    float64
	max      float64
	color    color.RGBA
	label    string
}

// Panel is a self-laying-out UI panel.
// Sizes are in character units (width) and line units (padding/spacing).
type Panel struct {
	// Configuration (set before Draw)
	Anchor    Anchor
	WidthCh   int     // width in character units (0 = auto-fit to content)
	X, Y      int     // pixel position (used when Anchor is AnchorManual)
	PaddingCh int     // padding in character units (default 1)
	MarginPx  int     // margin from screen edge in pixels (default 10)
	BgColor   color.RGBA
	Border    color.RGBA

	lines []lineEntry
}

// NewPanel creates a panel with default theme colors.
func NewPanel(anchor Anchor, widthCh int) *Panel {
	return &Panel{
		Anchor:    anchor,
		WidthCh:   widthCh,
		PaddingCh: 1,
		MarginPx:  10,
		BgColor:   utils.Theme.PanelBg,
		Border:    utils.Theme.PanelBorder,
	}
}

// Clear removes all lines (reuse panel each frame without reallocating).
func (p *Panel) Clear() {
	p.lines = p.lines[:0]
}

// Line adds a text line.
func (p *Panel) Line(text string, c color.RGBA) {
	p.lines = append(p.lines, lineEntry{text: text, color: c})
}

// LineRight adds a right-aligned text line.
func (p *Panel) LineRight(text string, c color.RGBA) {
	p.lines = append(p.lines, lineEntry{text: text, color: c, align: AlignRight})
}

// LineCenter adds a center-aligned text line.
func (p *Panel) LineCenter(text string, c color.RGBA) {
	p.lines = append(p.lines, lineEntry{text: text, color: c, align: AlignCenter})
}

// LinePair adds a line with left text and right-aligned text on the same line.
func (p *Panel) LinePair(left string, leftColor color.RGBA, right string, rightColor color.RGBA) {
	// We encode both parts into one entry; Draw handles the split
	p.lines = append(p.lines, lineEntry{
		text:  left + "\x00" + right, // null byte separator
		color: leftColor,
		align: AlignLeft, // special: contains pair
	})
	// Store right color in a second hidden entry
	p.lines = append(p.lines, lineEntry{
		text:  "",
		color: rightColor,
		align: Align(-1), // marker: this is the right part's color
	})
}

// Sep adds a horizontal separator line.
func (p *Panel) Sep() {
	p.lines = append(p.lines, lineEntry{separator: true})
}

// Bar adds a progress bar with a label.
func (p *Panel) Bar(value, max float64, barColor color.RGBA, label string) {
	p.lines = append(p.lines, lineEntry{
		bar: &barEntry{value: value, max: max, color: barColor, label: label},
	})
}

// LineH returns the current line height in pixels.
func LineH() int {
	return int(15.0 * utils.UIScale)
}

// Draw renders the panel to the screen.
func (p *Panel) Draw(screen *ebiten.Image) {
	cw := utils.CharWidth()
	lh := LineH()
	pad := p.PaddingCh * cw

	// Calculate dimensions
	widthPx := p.WidthCh * cw
	if widthPx == 0 {
		// Auto-fit: find widest line
		for _, line := range p.lines {
			if line.align == Align(-1) || line.separator {
				continue
			}
			w := len(line.text) * cw
			if w > widthPx {
				widthPx = w
			}
		}
		widthPx += pad * 2
	}

	// Count visible lines for height (bars with labels take extra space)
	heightContent := 0
	for _, line := range p.lines {
		if line.align == Align(-1) {
			continue // skip pair color markers
		}
		if line.bar != nil && line.bar.label != "" {
			heightContent += lh + 10 // bar + label
		} else if line.separator {
			heightContent += lh / 2
		} else {
			heightContent += lh
		}
	}
	heightPx := pad + heightContent + pad/2

	// Calculate position based on anchor
	px, py := p.X, p.Y
	switch p.Anchor {
	case AnchorTopLeft:
		px, py = p.MarginPx, p.MarginPx
	case AnchorTopRight:
		px = views.ScreenWidth - widthPx - p.MarginPx
		py = p.MarginPx
	case AnchorBottomLeft:
		px = p.MarginPx
		py = views.ScreenHeight - heightPx - p.MarginPx
	case AnchorBottomRight:
		px = views.ScreenWidth - widthPx - p.MarginPx
		py = views.ScreenHeight - heightPx - p.MarginPx
	case AnchorCenter:
		px = (views.ScreenWidth - widthPx) / 2
		py = (views.ScreenHeight - heightPx) / 2
	}

	// Draw background
	bg := &views.UIPanel{
		X: px, Y: py, Width: widthPx, Height: heightPx,
		BgColor: p.BgColor, BorderColor: p.Border,
	}
	bg.Draw(screen)

	// Draw lines
	textX := px + pad
	textY := py + pad
	contentW := widthPx - pad*2

	for i := 0; i < len(p.lines); i++ {
		line := p.lines[i]

		// Skip pair color markers
		if line.align == Align(-1) {
			continue
		}

		if line.separator {
			sepY := textY + lh/2
			views.DrawLine(screen, textX, sepY, textX+contentW, sepY, p.Border)
			textY += lh / 2
			continue
		}

		if line.bar != nil {
			// Draw progress bar
			barH := 6
			barBg := &views.UIPanel{X: textX, Y: textY + 2, Width: contentW, Height: barH,
				BgColor: utils.Theme.BarBg, BorderColor: utils.Theme.PanelBorder}
			barBg.Draw(screen)
			if line.bar.max > 0 {
				ratio := line.bar.value / line.bar.max
				if ratio > 1 {
					ratio = 1
				}
				fillW := int(float64(contentW) * ratio)
				if fillW > 0 {
					fill := &views.UIPanel{X: textX + 1, Y: textY + 3, Width: fillW - 2, Height: barH - 2,
						BgColor: line.bar.color, BorderColor: line.bar.color}
					fill.Draw(screen)
				}
			}
			if line.bar.label != "" {
				views.DrawText(screen, line.bar.label, textX, textY+barH+4, utils.Theme.TextDim)
				textY += lh + barH + 4 // bar + label needs two lines of space
			} else {
				textY += barH + 4
			}
			continue
		}

		// Check for pair (null-byte separated)
		if idx := findNull(line.text); idx >= 0 {
			leftText := line.text[:idx]
			rightText := line.text[idx+1:]
			views.DrawText(screen, leftText, textX, textY, line.color)

			rightColor := line.color
			if i+1 < len(p.lines) && p.lines[i+1].align == Align(-1) {
				rightColor = p.lines[i+1].color
			}
			rightW := len(rightText) * cw
			views.DrawText(screen, rightText, textX+contentW-rightW, textY, rightColor)
			textY += lh
			continue
		}

		// Regular line
		switch line.align {
		case AlignRight:
			w := len(line.text) * cw
			views.DrawText(screen, line.text, textX+contentW-w, textY, line.color)
		case AlignCenter:
			w := len(line.text) * cw
			views.DrawText(screen, line.text, textX+(contentW-w)/2, textY, line.color)
		default:
			views.DrawText(screen, line.text, textX, textY, line.color)
		}
		textY += lh
	}
}

// GetBounds returns the pixel bounds of the panel after drawing.
// Must be called after Draw.
func (p *Panel) GetBounds() (x, y, w, h int) {
	cw := utils.CharWidth()
	lh := LineH()
	pad := p.PaddingCh * cw
	widthPx := p.WidthCh * cw
	if widthPx == 0 {
		for _, line := range p.lines {
			if line.align == Align(-1) || line.separator {
				continue
			}
			w := len(line.text) * cw
			if w > widthPx {
				widthPx = w
			}
		}
		widthPx += pad * 2
	}
	heightContent := 0
	for _, line := range p.lines {
		if line.align == Align(-1) {
			continue
		}
		if line.bar != nil && line.bar.label != "" {
			heightContent += lh + 10
		} else if line.separator {
			heightContent += lh / 2
		} else {
			heightContent += lh
		}
	}
	heightPx := pad + heightContent + pad/2

	px, py := p.X, p.Y
	switch p.Anchor {
	case AnchorTopRight:
		px = views.ScreenWidth - widthPx - p.MarginPx
		py = p.MarginPx
	case AnchorBottomLeft:
		px = p.MarginPx
		py = views.ScreenHeight - heightPx - p.MarginPx
	case AnchorBottomRight:
		px = views.ScreenWidth - widthPx - p.MarginPx
		py = views.ScreenHeight - heightPx - p.MarginPx
	case AnchorCenter:
		px = (views.ScreenWidth - widthPx) / 2
		py = (views.ScreenHeight - heightPx) / 2
	case AnchorTopLeft:
		px, py = p.MarginPx, p.MarginPx
	}
	return px, py, widthPx, heightPx
}

func findNull(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			return i
		}
	}
	return -1
}
