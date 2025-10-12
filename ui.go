package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	defaultFontFace *text.GoXFace
)

func init() {
	// Initialize default font face using bitmap font
	defaultFontFace = text.NewGoXFace(bitmapfont.Face)
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
		BgColor:     UIPanelBg,
		BorderColor: UIPanelBorder,
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

// ContextMenuProvider interface for objects that can provide context menu data
type ContextMenuProvider interface {
	GetContextMenuTitle() string
	GetContextMenuItems() []string
}

// Clickable interface for objects that can be clicked and have a position
type Clickable interface {
	ContextMenuProvider
	GetPosition() (float64, float64) // Returns X, Y coordinates
	GetClickRadius() float64         // Returns click detection radius
}

// ClickHandler manages clickable objects and context menus
type ClickHandler struct {
	clickables     []Clickable
	activeMenu     *ContextMenu
	selectedObject Clickable
}

// NewClickHandler creates a new click handler
func NewClickHandler() *ClickHandler {
	return &ClickHandler{
		clickables: make([]Clickable, 0),
	}
}

// AddClickable adds an object to the click handler
func (c *ClickHandler) AddClickable(clickable Clickable) {
	c.clickables = append(c.clickables, clickable)
}

// ClearClickables removes all clickable objects
func (c *ClickHandler) ClearClickables() {
	c.clickables = make([]Clickable, 0)
	c.activeMenu = nil
	c.selectedObject = nil
}

// HandleClick processes a click at the given coordinates
func (c *ClickHandler) HandleClick(x, y int) bool {
	// Check clickables in reverse order (last added = first checked)
	// This gives priority to smaller objects added later
	for i := len(c.clickables) - 1; i >= 0; i-- {
		clickable := c.clickables[i]
		objX, objY := clickable.GetPosition()
		dx := float64(x) - objX
		dy := float64(y) - objY
		distance := math.Sqrt(dx*dx + dy*dy)

		if distance <= clickable.GetClickRadius() {
			// Toggle selection - if already selected, deselect
			if c.selectedObject == clickable {
				c.selectedObject = nil
				c.activeMenu = nil
			} else {
				c.selectedObject = clickable
				c.createContextMenu(clickable, int(objX), int(objY))
			}
			return true // Click was handled
		}
	}

	// No object was clicked, deselect
	c.selectedObject = nil
	c.activeMenu = nil
	return false
}

// createContextMenu creates and positions a context menu for the given object
func (c *ClickHandler) createContextMenu(clickable Clickable, x, y int) {
	c.activeMenu = NewContextMenuFromProvider(clickable)
	c.activeMenu.PositionNear(x, y, int(clickable.GetClickRadius())+20)
}

// GetSelectedObject returns the currently selected object
func (c *ClickHandler) GetSelectedObject() Clickable {
	return c.selectedObject
}

// GetActiveMenu returns the active context menu
func (c *ClickHandler) GetActiveMenu() *ContextMenu {
	return c.activeMenu
}

// HasActiveMenu returns whether there is an active menu
func (c *ClickHandler) HasActiveMenu() bool {
	return c.activeMenu != nil
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

// NewContextMenuFromProvider creates a context menu from any object that implements ContextMenuProvider
func NewContextMenuFromProvider(provider ContextMenuProvider) *ContextMenu {
	return NewContextMenu(provider.GetContextMenuTitle(), provider.GetContextMenuItems())
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
	DrawText(screen, c.Title, c.Panel.X+c.Padding, textY, UITextPrimary)

	// Draw separator
	textY += 20
	DrawText(screen, "─────────────────", c.Panel.X+c.Padding, textY, UITextSecondary)

	// Draw items
	textY += 20
	for _, item := range c.Items {
		DrawColoredMenuItem(screen, item, c.Panel.X+c.Padding, textY)
		textY += 15
	}
}

// DrawText draws text in a specific color using text/v2
func DrawText(screen *ebiten.Image, textStr string, x, y int, textColor color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, textStr, defaultFontFace, op)
}

// DrawCenteredText draws text centered at the given position
func DrawCenteredText(screen *ebiten.Image, textStr string, x, y int) {
	// Approximate text width (each character is about 6 pixels wide)
	textWidth := len(textStr) * 6
	DrawText(screen, textStr, x-textWidth/2, y, UITextPrimary)
}

// DrawTextCentered draws text centered at the given position with color and scale
func DrawTextCentered(screen *ebiten.Image, textStr string, x, y int, textColor color.RGBA, scale float64) {
	op := &text.DrawOptions{}

	// Calculate text bounds for centering
	bounds, _ := text.Measure(textStr, defaultFontFace, 0)
	width := bounds * scale

	// Apply scale and center
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(x)-width/2, float64(y))
	op.ColorScale.ScaleWithColor(textColor)

	text.Draw(screen, textStr, defaultFontFace, op)
}

// DrawColoredMenuItem draws a menu item with special handling for planet/station types
func DrawColoredMenuItem(screen *ebiten.Image, textStr string, x, y int) {
	// Check if this is a type line for a planet
	if len(textStr) > 6 && textStr[:5] == "Type:" {
		// Extract the type (everything after "Type: ")
		typeText := textStr[6:]

		// Get color for the type
		var typeColor color.RGBA
		if planetColor := getPlanetTypeColor(typeText); planetColor != (color.RGBA{}) {
			typeColor = planetColor
		} else if stationColor := getStationTypeColor(typeText); stationColor != (color.RGBA{}) {
			typeColor = stationColor
		} else {
			// Default to primary text color
			typeColor = UITextPrimary
		}

		// Draw "Type: " in secondary color
		DrawText(screen, "Type: ", x, y, UITextSecondary)

		// Draw the type in its color
		DrawText(screen, typeText, x+36, y, typeColor)
	} else if len(textStr) > 4 && textStr[:3] == "  -" {
		// This is a list item like "  - Planet 1 (Terrestrial)"
		// Extract planet/station type from parentheses
		openParen := -1
		closeParen := -1
		for i, c := range textStr {
			if c == '(' {
				openParen = i
			} else if c == ')' {
				closeParen = i
				break
			}
		}

		if openParen > 0 && closeParen > openParen {
			// Extract the type from parentheses
			typeText := textStr[openParen+1 : closeParen]
			beforeType := textStr[:openParen]
			afterType := textStr[closeParen:]

			// Get color for the type
			var typeColor color.RGBA
			if planetColor := getPlanetTypeColor(typeText); planetColor != (color.RGBA{}) {
				typeColor = planetColor
			} else if stationColor := getStationTypeColor(typeText); stationColor != (color.RGBA{}) {
				typeColor = stationColor
			} else {
				typeColor = UITextSecondary
			}

			// Draw everything before the type
			DrawText(screen, beforeType+"(", x, y, UITextSecondary)
			// Calculate offset for the type text
			beforeWidth := len(beforeType+"(") * 6
			// Draw the type in color
			DrawText(screen, typeText, x+beforeWidth, y, typeColor)
			// Draw closing paren and anything after
			typeWidth := len(typeText) * 6
			DrawText(screen, afterType, x+beforeWidth+typeWidth, y, UITextSecondary)
		} else {
			// No parentheses, just draw normally
			DrawText(screen, textStr, x, y, UITextSecondary)
		}
	} else {
		// Normal text in secondary color
		DrawText(screen, textStr, x, y, UITextSecondary)
	}
}

// getPlanetTypeColor returns the color for a planet type
func getPlanetTypeColor(planetType string) color.RGBA {
	switch planetType {
	case "Terrestrial":
		return PlanetTerrestrial
	case "Gas Giant":
		return PlanetGasGiant
	case "Ice World":
		return PlanetIce
	case "Desert":
		return PlanetDesert
	case "Ocean":
		return PlanetOcean
	case "Lava":
		return PlanetLava
	default:
		return color.RGBA{}
	}
}

// getStationTypeColor returns the color for a station type
func getStationTypeColor(stationType string) color.RGBA {
	// Remove " Station" suffix if present
	if len(stationType) > 8 && stationType[len(stationType)-8:] == " Station" {
		stationType = stationType[:len(stationType)-8]
	}

	switch stationType {
	case "Trading":
		return ColorStationTrading
	case "Military":
		return ColorStationMilitary
	case "Research":
		return ColorStationResearch
	case "Mining":
		return ColorStationMining
	case "Refinery":
		return ColorStationRefinery
	case "Shipyard":
		return ColorStationShipyard
	default:
		return color.RGBA{}
	}
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
