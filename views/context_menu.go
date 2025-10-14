package views

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/utils"
)

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
	if c.Panel.X+c.Panel.Width > ScreenWidth-10 {
		c.Panel.X = targetX - offset - c.Panel.Width
	}

	// Adjust if too close to top edge
	if c.Panel.Y < 10 {
		c.Panel.Y = 10
	}

	// Adjust if too close to bottom edge
	if c.Panel.Y+c.Panel.Height > ScreenHeight-10 {
		c.Panel.Y = ScreenHeight - c.Panel.Height - 10
	}
}

// Draw renders the context menu
func (c *ContextMenu) Draw(screen *ebiten.Image) {
	// Draw panel background
	c.Panel.Draw(screen)

	// Draw title
	textY := c.Panel.Y + c.Padding
	DrawText(screen, c.Title, c.Panel.X+c.Padding, textY, utils.TextPrimary)

	// Draw separator
	textY += 20
	DrawText(screen, "─────────────────", c.Panel.X+c.Padding, textY, utils.TextSecondary)

	// Draw items
	textY += 20
	for _, item := range c.Items {
		DrawColoredMenuItem(screen, item, c.Panel.X+c.Padding, textY)
		textY += 15
	}
}

// DrawColoredMenuItem draws a menu item with special handling for planet/station types
func DrawColoredMenuItem(screen *ebiten.Image, textStr string, x, y int) {
	// Check if this is a type line for a planet
	if len(textStr) > 6 && textStr[:5] == "Type:" {
		// Extract the type (everything after "Type: ")
		typeText := textStr[6:]

		// Get color for the type
		var typeColor = utils.TextPrimary
		if planetColor := utils.GetPlanetTypeColor(typeText); planetColor.A > 0 {
			typeColor = planetColor
		} else if stationColor := utils.GetStationTypeColor(typeText); stationColor.A > 0 {
			typeColor = stationColor
		}

		// Draw "Type: " in secondary color
		DrawText(screen, "Type: ", x, y, utils.TextSecondary)

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
			var typeColor = utils.TextSecondary
			if planetColor := utils.GetPlanetTypeColor(typeText); planetColor.A > 0 {
				typeColor = planetColor
			} else if stationColor := utils.GetStationTypeColor(typeText); stationColor.A > 0 {
				typeColor = stationColor
			}

			// Draw everything before the type
			DrawText(screen, beforeType+"(", x, y, utils.TextSecondary)
			// Calculate offset for the type text
			beforeWidth := len(beforeType+"(") * 6
			// Draw the type in color
			DrawText(screen, typeText, x+beforeWidth, y, typeColor)
			// Draw closing paren and anything after
			typeWidth := len(typeText) * 6
			DrawText(screen, afterType, x+beforeWidth+typeWidth, y, utils.TextSecondary)
		} else {
			// No parentheses, just draw normally
			DrawText(screen, textStr, x, y, utils.TextSecondary)
		}
	} else {
		// Normal text in secondary color
		DrawText(screen, textStr, x, y, utils.TextSecondary)
	}
}
