package views

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/utils"
)

const (
	keyBindingsConfigPath = "~/.xandaris/keybindings.json"
)

// SettingsView displays game settings including key bindings
type SettingsView struct {
	ctx              GameContext
	selectedIndex    int
	scrollOffset     int
	editingAction    KeyAction
	waitingForKey    bool
	actions          []KeyAction
	errorMessage     string
	errorTimer       int
	returnToMainMenu bool // Whether to return to main menu (vs galaxy view)
}

// NewSettingsView creates a new settings view
func NewSettingsView(ctx GameContext) *SettingsView {
	sv := &SettingsView{
		ctx:              ctx,
		selectedIndex:    0,
		scrollOffset:     0,
		editingAction:    "",
		waitingForKey:    false,
		returnToMainMenu: true,
	}
	sv.actions = ctx.GetKeyBindings().GetAllActions()
	return sv
}

// SetReturnDestination sets where escape should go
func (sv *SettingsView) SetReturnDestination(toMainMenu bool) {
	sv.returnToMainMenu = toMainMenu
}

// Update updates the settings view
func (sv *SettingsView) Update() error {
	kb := sv.ctx.GetKeyBindings()
	vm := sv.ctx.GetViewManager()

	// Decrement error timer
	if sv.errorTimer > 0 {
		sv.errorTimer--
	}

	// If waiting for key input
	if sv.waitingForKey {
		sv.handleKeyInput()
		return nil
	}

	// Handle escape to go back
	if kb.IsActionJustPressed(ActionEscape) {
		if sv.returnToMainMenu {
			vm.SwitchTo(ViewTypeMainMenu)
		} else {
			vm.SwitchTo(ViewTypeGalaxy)
		}
		return nil
	}

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		sv.handleMouseClick(mx, my)
	}

	// Keyboard navigation
	if kb.IsActionJustPressed(ActionMenuUp) {
		sv.selectedIndex--
		if sv.selectedIndex < 0 {
			sv.selectedIndex = len(sv.actions) + 1 // +1 for save button, +1 for reset button
		}
	}

	if kb.IsActionJustPressed(ActionMenuDown) {
		sv.selectedIndex++
		if sv.selectedIndex > len(sv.actions)+1 {
			sv.selectedIndex = 0
		}
	}

	if kb.IsActionJustPressed(ActionMenuConfirm) {
		sv.handleSelection()
	}

	// Handle scrolling
	_, dy := ebiten.Wheel()
	if dy != 0 {
		sv.scrollOffset -= int(dy * 20)
		maxScroll := len(sv.actions)*50 - 400
		if maxScroll < 0 {
			maxScroll = 0
		}
		if sv.scrollOffset < 0 {
			sv.scrollOffset = 0
		}
		if sv.scrollOffset > maxScroll {
			sv.scrollOffset = maxScroll
		}
	}

	return nil
}

// handleMouseClick handles mouse clicks on settings items
func (sv *SettingsView) handleMouseClick(mx, my int) {
	vm := sv.ctx.GetViewManager()

	// Back button
	if mx >= 50 && mx <= 200 && my >= 50 && my <= 90 {
		if sv.returnToMainMenu {
			vm.SwitchTo(ViewTypeMainMenu)
		} else {
			vm.SwitchTo(ViewTypeGalaxy)
		}
		return
	}

	// Save button
	saveButtonY := ScreenHeight - 120
	if mx >= ScreenWidth/2-100 && mx <= ScreenWidth/2+100 && my >= saveButtonY && my <= saveButtonY+40 {
		sv.saveSettings()
		return
	}

	// Reset to defaults button
	resetButtonY := ScreenHeight - 70
	if mx >= ScreenWidth/2-100 && mx <= ScreenWidth/2+100 && my >= resetButtonY && my <= resetButtonY+40 {
		sv.resetToDefaults()
		return
	}

	// Key binding items
	startY := 200
	for i, action := range sv.actions {
		itemY := startY + i*50 - sv.scrollOffset
		if itemY < 150 || itemY > ScreenHeight-180 {
			continue
		}

		// Check if clicked on this item
		if mx >= 200 && mx <= 1080 && my >= itemY && my < itemY+45 {
			sv.selectedIndex = i
			sv.startEditingKey(action)
			return
		}
	}
}

// handleSelection handles enter key on selected item
func (sv *SettingsView) handleSelection() {
	if sv.selectedIndex < len(sv.actions) {
		// Edit key binding
		sv.startEditingKey(sv.actions[sv.selectedIndex])
	} else if sv.selectedIndex == len(sv.actions) {
		// Save button
		sv.saveSettings()
	} else if sv.selectedIndex == len(sv.actions)+1 {
		// Reset button
		sv.resetToDefaults()
	}
}

// startEditingKey starts editing a key binding
func (sv *SettingsView) startEditingKey(action KeyAction) {
	sv.editingAction = action
	sv.waitingForKey = true
}

// handleKeyInput waits for a key press to rebind
func (sv *SettingsView) handleKeyInput() {
	kb := sv.ctx.GetKeyBindings()

	// Check for escape to cancel
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sv.waitingForKey = false
		sv.editingAction = ""
		return
	}

	// Check all keys
	for key := ebiten.Key(0); key < ebiten.KeyMax; key++ {
		if inpututil.IsKeyJustPressed(key) && key != ebiten.KeyEscape {
			// Check if this key is already bound to another action
			existingAction := sv.findActionForKey(key)
			if existingAction != "" && existingAction != sv.editingAction {
				sv.errorMessage = fmt.Sprintf("Key already bound to: %s", GetActionName(existingAction))
				sv.errorTimer = 180 // 3 seconds
				sv.waitingForKey = false
				sv.editingAction = ""
				return
			}

			// Bind the key
			kb.SetKey(sv.editingAction, key)
			sv.waitingForKey = false
			sv.editingAction = ""
			return
		}
	}
}

// findActionForKey finds which action is bound to a key
func (sv *SettingsView) findActionForKey(key ebiten.Key) KeyAction {
	kb := sv.ctx.GetKeyBindings()
	for _, action := range sv.actions {
		if boundKey, ok := kb.GetKey(action); ok && boundKey == key {
			return action
		}
	}
	return ""
}

// saveSettings saves the current key bindings to config file
func (sv *SettingsView) saveSettings() {
	kb := sv.ctx.GetKeyBindings()
	if err := kb.SaveToFile(keyBindingsConfigPath); err != nil {
		sv.errorMessage = fmt.Sprintf("Failed to save: %v", err)
		sv.errorTimer = 180
	} else {
		sv.errorMessage = "Settings saved!"
		sv.errorTimer = 120
	}
}

// resetToDefaults resets all key bindings to defaults
func (sv *SettingsView) resetToDefaults() {
	kb := sv.ctx.GetKeyBindings()
	kb.LoadDefaults()
	sv.errorMessage = "Reset to defaults"
	sv.errorTimer = 120
}

// Draw renders the settings view
func (sv *SettingsView) Draw(screen *ebiten.Image) {
	kb := sv.ctx.GetKeyBindings()

	// Background
	screen.Fill(utils.BackgroundDark)

	// Title
	DrawTextCentered(screen, "Settings", ScreenWidth/2, 80, utils.SystemLightBlue, 2.0)

	// Back button
	backPanel := &UIPanel{
		X:           50,
		Y:           50,
		Width:       150,
		Height:      40,
		BgColor:     utils.ButtonActive,
		BorderColor: utils.Highlight,
	}
	backPanel.Draw(screen)
	DrawText(screen, "< Back", 70, 60, utils.TextPrimary)

	// Subtitle
	DrawTextCentered(screen, "Key Bindings", ScreenWidth/2, 130, utils.TextPrimary, 1.2)

	// Key binding list
	startY := 200
	for i, action := range sv.actions {
		itemY := startY + i*50 - sv.scrollOffset

		// Skip if off screen
		if itemY < 150 || itemY > ScreenHeight-180 {
			continue
		}

		// Highlight selected
		bgColor := utils.PanelBg
		if i == sv.selectedIndex {
			bgColor = utils.ButtonActive
		}

		// Draw item background
		itemPanel := &UIPanel{
			X:           200,
			Y:           itemY,
			Width:       880,
			Height:      45,
			BgColor:     bgColor,
			BorderColor: utils.PanelBorder,
		}
		itemPanel.Draw(screen)

		// Action name
		actionName := GetActionName(action)
		DrawText(screen, actionName, 220, itemY+15, utils.TextPrimary)

		// Current key binding
		keyName := "Not bound"
		if currentKey, ok := kb.GetKey(action); ok {
			keyName = kb.GetKeyName(currentKey)
		}

		// If editing this action, show "Press key..."
		if sv.waitingForKey && sv.editingAction == action {
			keyName = "Press key..."
		}

		keyColor := utils.TextSecondary
		if sv.waitingForKey && sv.editingAction == action {
			keyColor = utils.SystemYellow
		}

		DrawText(screen, keyName, 800, itemY+15, keyColor)
		DrawText(screen, "[Click to change]", 920, itemY+15, color.RGBA{100, 100, 100, 255})
	}

	// Save button
	saveButtonY := ScreenHeight - 120
	saveSelected := sv.selectedIndex == len(sv.actions)
	saveBgColor := utils.ButtonActive
	if saveSelected {
		saveBgColor = utils.Highlight
	}

	savePanel := &UIPanel{
		X:           ScreenWidth/2 - 100,
		Y:           saveButtonY,
		Width:       200,
		Height:      40,
		BgColor:     saveBgColor,
		BorderColor: utils.Highlight,
	}
	savePanel.Draw(screen)
	DrawTextCentered(screen, "Save Settings", ScreenWidth/2, saveButtonY+12, utils.TextPrimary, 1.0)

	// Reset to defaults button
	resetButtonY := ScreenHeight - 70
	resetSelected := sv.selectedIndex == len(sv.actions)+1
	resetBgColor := utils.ButtonDisabled
	if resetSelected {
		resetBgColor = color.RGBA{100, 60, 60, 230}
	}

	resetPanel := &UIPanel{
		X:           ScreenWidth/2 - 100,
		Y:           resetButtonY,
		Width:       200,
		Height:      40,
		BgColor:     resetBgColor,
		BorderColor: utils.PanelBorder,
	}
	resetPanel.Draw(screen)
	DrawTextCentered(screen, "Reset to Defaults", ScreenWidth/2, resetButtonY+12, utils.TextPrimary, 0.9)

	// Error/success message
	if sv.errorTimer > 0 {
		msgColor := utils.SystemYellow
		if sv.errorMessage == "Settings saved!" || sv.errorMessage == "Reset to defaults" {
			msgColor = utils.StationResearch // Green
		} else {
			msgColor = utils.SystemRed
		}
		DrawTextCentered(screen, sv.errorMessage, ScreenWidth/2, 160, msgColor, 1.0)
	}

	// Scroll hint
	if len(sv.actions) > 10 {
		DrawTextCentered(screen, "Scroll for more", ScreenWidth/2, ScreenHeight-20, utils.TextSecondary, 0.8)
	}
}

// GetType returns the view type
func (sv *SettingsView) GetType() ViewType {
	return ViewTypeSettings
}

// OnEnter is called when entering this view
func (sv *SettingsView) OnEnter() {
	// Refresh actions list
	sv.actions = sv.ctx.GetKeyBindings().GetAllActions()
	sv.selectedIndex = 0
	sv.scrollOffset = 0
	sv.waitingForKey = false
	sv.editingAction = ""
	sv.errorMessage = ""
	sv.errorTimer = 0
}

// OnExit is called when leaving this view
func (sv *SettingsView) OnExit() {
	// Nothing to clean up
}

// GetActionName returns a human-readable name for an action
func GetActionName(action KeyAction) string {
	switch action {
	case ActionPauseToggle:
		return "Pause/Unpause"
	case ActionSpeedSlow:
		return "Speed: 1x"
	case ActionSpeedNormal:
		return "Speed: 2x"
	case ActionSpeedFast:
		return "Speed: 4x"
	case ActionSpeedVeryFast:
		return "Speed: 8x"
	case ActionQuickSave:
		return "Quick Save"
	case ActionEscape:
		return "Escape/Back"
	case ActionOpenBuildMenu:
		return "Open Build Menu"
	case ActionOpenMarket:
		return "Open Market"
	case ActionOpenPlayerDir:
		return "Open Player Directory"
	case ActionFocusHome:
		return "Focus Home System"
	case ActionToggleWorkforceView:
		return "Toggle Workforce View"
	case ActionMenuUp:
		return "Menu: Up"
	case ActionMenuDown:
		return "Menu: Down"
	case ActionMenuConfirm:
		return "Menu: Confirm"
	case ActionMenuCancel:
		return "Menu: Cancel"
	case ActionMenuDelete:
		return "Menu: Delete"
	default:
		return string(action)
	}
}
