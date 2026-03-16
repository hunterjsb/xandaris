package views

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/utils"
)

// ActionMenuItem represents a clickable menu item with a callback
type ActionMenuItem struct {
	Label    string
	Action   func() // Callback when clicked
	Enabled  bool
	Shortcut string // Optional keyboard shortcut hint
}

// ActionMenu represents an interactive popup menu with clickable actions
type ActionMenu struct {
	Panel        *UIPanel
	Title        string
	Items        []ActionMenuItem
	Padding      int
	selectedItem int
	visible      bool
}

// NewActionMenu creates a new action menu
func NewActionMenu(title string, items []ActionMenuItem) *ActionMenu {
	lh := int(15.0 * utils.UIScale)
	cw := utils.CharWidth()
	padding := cw
	width := 28 * cw
	height := padding*2 + lh + len(items)*lh + lh

	return &ActionMenu{
		Panel:        NewUIPanel(0, 0, width, height),
		Title:        title,
		Items:        items,
		Padding:      padding,
		selectedItem: -1,
		visible:      true,
	}
}

// PositionNear positions the action menu near a target point
func (am *ActionMenu) PositionNear(targetX, targetY, offset int) {
	// Try to position to the right of the target
	am.Panel.X = targetX + offset
	am.Panel.Y = targetY - am.Panel.Height/2

	// Adjust if too close to right edge
	if am.Panel.X+am.Panel.Width > ScreenWidth-10 {
		am.Panel.X = targetX - offset - am.Panel.Width
	}

	// Adjust if too close to top edge
	if am.Panel.Y < 10 {
		am.Panel.Y = 10
	}

	// Adjust if too close to bottom edge
	if am.Panel.Y+am.Panel.Height > ScreenHeight-10 {
		am.Panel.Y = ScreenHeight - am.Panel.Height - 10
	}
}

// Update handles input for the action menu
func (am *ActionMenu) Update() bool {
	if !am.visible {
		return false
	}

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()

		// Check if clicking outside menu to close
		if mx < am.Panel.X || mx > am.Panel.X+am.Panel.Width ||
			my < am.Panel.Y || my > am.Panel.Y+am.Panel.Height {
			am.visible = false
			return true
		}

		// Check clicks on menu items
		lh := int(15.0 * utils.UIScale)
		itemY := am.Panel.Y + am.Padding + lh*2 // Title + separator
		for _, item := range am.Items {
			if mx >= am.Panel.X+am.Padding && mx <= am.Panel.X+am.Panel.Width-am.Padding &&
				my >= itemY && my < itemY+lh {
				if item.Enabled && item.Action != nil {
					item.Action()
					am.visible = false
					return true
				}
				break
			}
			itemY += lh
		}
	}

	// Update hover state
	lh := int(15.0 * utils.UIScale)
	mx, my := ebiten.CursorPosition()
	itemY := am.Panel.Y + am.Padding + lh*2
	am.selectedItem = -1
	for i := range am.Items {
		if mx >= am.Panel.X+am.Padding && mx <= am.Panel.X+am.Panel.Width-am.Padding &&
			my >= itemY && my < itemY+lh {
			am.selectedItem = i
			break
		}
		itemY += lh
	}

	return false
}

// Draw renders the action menu
func (am *ActionMenu) Draw(screen *ebiten.Image) {
	if !am.visible {
		return
	}

	// Draw panel background
	am.Panel.Draw(screen)

	lh := int(15.0 * utils.UIScale)

	// Draw title
	textY := am.Panel.Y + am.Padding
	DrawText(screen, am.Title, am.Panel.X+am.Padding, textY, utils.TextPrimary)

	// Draw separator
	textY += lh
	sepY := textY + lh/4
	DrawLine(screen, am.Panel.X+am.Padding, sepY, am.Panel.X+am.Panel.Width-am.Padding, sepY, utils.Theme.PanelBorder)
	textY += lh/2 + lh/4

	// Draw items
	for i, item := range am.Items {
		itemColor := utils.TextPrimary
		if !item.Enabled {
			itemColor = utils.TextSecondary
		} else if i == am.selectedItem {
			// Highlight hovered item
			highlightPanel := &UIPanel{
				X:           am.Panel.X + am.Padding,
				Y:           textY - 2,
				Width:       am.Panel.Width - am.Padding*2,
				Height:      lh,
				BgColor:     utils.ButtonActive,
				BorderColor: utils.PanelBorder,
			}
			highlightPanel.Draw(screen)
			itemColor = utils.Highlight
		}

		// Draw item label
		DrawText(screen, item.Label, am.Panel.X+am.Padding+5, textY, itemColor)

		// Draw shortcut hint if available
		if item.Shortcut != "" {
			shortcutX := am.Panel.X + am.Panel.Width - am.Padding - len(item.Shortcut)*utils.CharWidth() - 5
			DrawText(screen, item.Shortcut, shortcutX, textY, utils.TextSecondary)
		}

		textY += lh
	}
}

// IsVisible returns whether the menu is visible
func (am *ActionMenu) IsVisible() bool {
	return am.visible
}

// Hide hides the action menu
func (am *ActionMenu) Hide() {
	am.visible = false
}
