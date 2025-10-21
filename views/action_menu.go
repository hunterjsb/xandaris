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
	padding := 10
	width := 250
	height := padding*2 + 20 + len(items)*20 + 10

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
		itemY := am.Panel.Y + am.Padding + 40 // Title + separator
		for _, item := range am.Items {
			if mx >= am.Panel.X+am.Padding && mx <= am.Panel.X+am.Panel.Width-am.Padding &&
				my >= itemY && my < itemY+20 {
				if item.Enabled && item.Action != nil {
					item.Action()
					am.visible = false
					return true
				}
				break
			}
			itemY += 20
		}
	}

	// Update hover state
	mx, my := ebiten.CursorPosition()
	itemY := am.Panel.Y + am.Padding + 40
	am.selectedItem = -1
	for i := range am.Items {
		if mx >= am.Panel.X+am.Padding && mx <= am.Panel.X+am.Panel.Width-am.Padding &&
			my >= itemY && my < itemY+20 {
			am.selectedItem = i
			break
		}
		itemY += 20
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

	// Draw title
	textY := am.Panel.Y + am.Padding
	DrawText(screen, am.Title, am.Panel.X+am.Padding, textY, utils.TextPrimary)

	// Draw separator
	textY += 20
	DrawText(screen, "──────────────────────", am.Panel.X+am.Padding, textY, utils.TextSecondary)

	// Draw items
	textY += 20
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
				Height:      18,
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
			shortcutX := am.Panel.X + am.Panel.Width - am.Padding - len(item.Shortcut)*6 - 5
			DrawText(screen, item.Shortcut, shortcutX, textY, utils.TextSecondary)
		}

		textY += 20
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
