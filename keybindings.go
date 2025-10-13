package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/views"
)

// Import key action constants from views package
const (
	ActionPauseToggle   = views.ActionPauseToggle
	ActionSpeedSlow     = views.ActionSpeedSlow
	ActionSpeedNormal   = views.ActionSpeedNormal
	ActionSpeedFast     = views.ActionSpeedFast
	ActionSpeedVeryFast = views.ActionSpeedVeryFast
	ActionSpeedIncrease = views.ActionSpeedIncrease
	ActionQuickSave     = views.ActionQuickSave
	ActionEscape        = views.ActionEscape
	ActionOpenBuildMenu = views.ActionOpenBuildMenu
	ActionMenuUp        = views.ActionMenuUp
	ActionMenuDown      = views.ActionMenuDown
	ActionMenuConfirm   = views.ActionMenuConfirm
	ActionMenuCancel    = views.ActionMenuCancel
	ActionMenuDelete    = views.ActionMenuDelete
)

// KeyBindings manages all key bindings for the game
type KeyBindings struct {
	bindings map[views.KeyAction]ebiten.Key
}

// NewKeyBindings creates a new key bindings manager with default bindings
func NewKeyBindings() *KeyBindings {
	kb := &KeyBindings{
		bindings: make(map[views.KeyAction]ebiten.Key),
	}
	kb.LoadDefaults()
	return kb
}

// LoadDefaults loads the default key bindings
func (kb *KeyBindings) LoadDefaults() {
	// Global actions
	kb.bindings[ActionPauseToggle] = ebiten.KeySpace
	kb.bindings[ActionSpeedSlow] = ebiten.Key1
	kb.bindings[ActionSpeedNormal] = ebiten.Key2
	kb.bindings[ActionSpeedFast] = ebiten.Key3
	kb.bindings[ActionSpeedVeryFast] = ebiten.Key4
	kb.bindings[ActionSpeedIncrease] = ebiten.KeyEqual
	kb.bindings[ActionQuickSave] = ebiten.KeyF5

	// View navigation
	kb.bindings[ActionEscape] = ebiten.KeyEscape
	kb.bindings[ActionOpenBuildMenu] = ebiten.KeyB

	// Menu navigation
	kb.bindings[ActionMenuUp] = ebiten.KeyUp
	kb.bindings[ActionMenuDown] = ebiten.KeyDown
	kb.bindings[ActionMenuConfirm] = ebiten.KeyEnter
	kb.bindings[ActionMenuCancel] = ebiten.KeyEscape
	kb.bindings[ActionMenuDelete] = ebiten.KeyBackspace
}

// IsActionJustPressed checks if the key bound to an action was just pressed
func (kb *KeyBindings) IsActionJustPressed(action views.KeyAction) bool {
	key, exists := kb.bindings[action]
	if !exists {
		return false
	}
	return inpututil.IsKeyJustPressed(key)
}

// GetKey returns the key bound to an action (implements KeyBindingsInterface)
func (kb *KeyBindings) GetKey(action views.KeyAction) (ebiten.Key, bool) {
	key, exists := kb.bindings[action]
	return key, exists
}

// SetKey sets the key binding for an action
func (kb *KeyBindings) SetKey(action views.KeyAction, key ebiten.Key) {
	kb.bindings[action] = key
}

// GetActionName returns a human-readable name for an action
func (kb *KeyBindings) GetActionName(action views.KeyAction) string {
	names := map[views.KeyAction]string{
		ActionPauseToggle:   "Pause/Resume",
		ActionSpeedSlow:     "Speed: Slow",
		ActionSpeedNormal:   "Speed: Normal",
		ActionSpeedFast:     "Speed: Fast",
		ActionSpeedVeryFast: "Speed: Very Fast",
		ActionSpeedIncrease: "Speed: Increase",
		ActionQuickSave:     "Quick Save",
		ActionEscape:        "Escape/Back",
		ActionOpenBuildMenu: "Open Build Menu",
		ActionMenuUp:        "Menu: Up",
		ActionMenuDown:      "Menu: Down",
		ActionMenuConfirm:   "Menu: Confirm",
		ActionMenuCancel:    "Menu: Cancel",
		ActionMenuDelete:    "Menu: Delete",
	}

	name, exists := names[action]
	if !exists {
		return string(action)
	}
	return name
}

// GetKeyName returns a human-readable name for a key
func (kb *KeyBindings) GetKeyName(key ebiten.Key) string {
	// Special cases
	switch key {
	case ebiten.KeySpace:
		return "Space"
	case ebiten.KeyEnter:
		return "Enter"
	case ebiten.KeyEscape:
		return "Escape"
	case ebiten.KeyBackspace:
		return "Backspace"
	case ebiten.KeyTab:
		return "Tab"
	case ebiten.KeyUp:
		return "Up"
	case ebiten.KeyDown:
		return "Down"
	case ebiten.KeyLeft:
		return "Left"
	case ebiten.KeyRight:
		return "Right"
	case ebiten.KeyEqual:
		return "="
	case ebiten.KeyMinus:
		return "-"
	case ebiten.KeyKPAdd:
		return "Numpad +"
	case ebiten.KeyKPSubtract:
		return "Numpad -"
	}

	// F-keys
	if key >= ebiten.KeyF1 && key <= ebiten.KeyF12 {
		return fmt.Sprintf("F%d", int(key-ebiten.KeyF1+1))
	}

	// Number keys
	if key >= ebiten.Key0 && key <= ebiten.Key9 {
		return fmt.Sprintf("%d", int(key-ebiten.Key0))
	}

	// Letter keys
	if key >= ebiten.KeyA && key <= ebiten.KeyZ {
		return fmt.Sprintf("%c", 'A'+int(key-ebiten.KeyA))
	}

	return fmt.Sprintf("Key%d", int(key))
}

// GetAllActions returns all available actions
func (kb *KeyBindings) GetAllActions() []views.KeyAction {
	return []views.KeyAction{
		ActionPauseToggle,
		ActionSpeedSlow,
		ActionSpeedNormal,
		ActionSpeedFast,
		ActionSpeedVeryFast,
		ActionSpeedIncrease,
		ActionQuickSave,
		ActionEscape,
		ActionOpenBuildMenu,
		ActionMenuUp,
		ActionMenuDown,
		ActionMenuConfirm,
		ActionMenuCancel,
		ActionMenuDelete,
	}
}

// SaveToFile saves key bindings to a JSON file
func (kb *KeyBindings) SaveToFile(filename string) error {
	// Convert to a serializable format
	data := make(map[string]int)
	for action, key := range kb.bindings {
		data[string(action)] = int(key)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal key bindings: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write key bindings file: %w", err)
	}

	return nil
}

// LoadFromFile loads key bindings from a JSON file
func (kb *KeyBindings) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, use defaults
			return nil
		}
		return fmt.Errorf("failed to read key bindings file: %w", err)
	}

	// Parse JSON
	rawData := make(map[string]int)
	if err := json.Unmarshal(data, &rawData); err != nil {
		return fmt.Errorf("failed to unmarshal key bindings: %w", err)
	}

	// Convert back to views.KeyAction -> ebiten.Key
	for actionStr, keyInt := range rawData {
		action := views.KeyAction(actionStr)
		key := ebiten.Key(keyInt)
		kb.bindings[action] = key
	}

	return nil
}

// GetConfigPath returns the default config file path
func GetKeyBindingsConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "keybindings.json"
	}
	return filepath.Join(homeDir, ".xandaris", "keybindings.json")
}
