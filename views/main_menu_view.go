package views

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/utils"
)

// MainMenuOption represents the different options in the main menu
type MainMenuOption int

const (
	MainMenuNewGame MainMenuOption = iota
	MainMenuLoadGame
	MainMenuSettings
	MainMenuQuit
)

// MainMenuView displays the main menu for starting or loading games
type MainMenuView struct {
	ctx              GameContext
	playerName       string
	selectedOption   MainMenuOption
	saveFiles        []SaveFileInfo
	selectedSave     int
	showLoadMenu     bool
	inputActive      bool
	errorMessage     string
	errorTimer       int
	renameMode       bool // true if renaming a save
	renameBuffer     string
	deleteConfirm    bool // true if confirming delete
	deleteConfirmMsg string
}

// NewMainMenuView creates a new main menu view
func NewMainMenuView(ctx GameContext) *MainMenuView {
	return &MainMenuView{
		ctx:            ctx,
		playerName:     "Player",
		selectedOption: 0,
		selectedSave:   0,
		showLoadMenu:   false,
		inputActive:    false,
	}
}

// Update updates the main menu view
func (mmv *MainMenuView) Update() error {
	// Decrement error timer
	if mmv.errorTimer > 0 {
		mmv.errorTimer--
		if mmv.errorTimer == 0 {
			mmv.errorMessage = ""
		}
	}

	// Handle text input for player name
	if mmv.inputActive && !mmv.showLoadMenu {
		mmv.handleTextInput()
	}

	// Handle rename mode
	if mmv.renameMode {
		mmv.handleRenameInput()
		return nil
	}

	// Handle delete confirmation
	if mmv.deleteConfirm {
		mmv.handleDeleteConfirm()
		return nil
	}

	// Handle menu navigation
	if !mmv.showLoadMenu {
		if err := mmv.handleMainMenuInput(); err != nil {
			return err
		}
	} else {
		mmv.handleLoadMenuInput()
	}

	return nil
}

// handleTextInput processes keyboard input for player name
func (mmv *MainMenuView) handleTextInput() {
	kb := mmv.ctx.GetKeyBindings()

	// Handle character input
	for _, r := range ebiten.AppendInputChars(nil) {
		if len(mmv.playerName) < 20 { // Max 20 characters
			mmv.playerName += string(r)
		}
	}

	// Handle backspace
	if kb.IsActionJustPressed(ActionMenuDelete) {
		if len(mmv.playerName) > 0 {
			mmv.playerName = mmv.playerName[:len(mmv.playerName)-1]
		}
	}

	// Handle enter to confirm input
	if kb.IsActionJustPressed(ActionMenuConfirm) {
		mmv.inputActive = false
	}

	// Handle escape to cancel input
	if kb.IsActionJustPressed(ActionMenuCancel) {
		mmv.inputActive = false
	}
}

// handleMainMenuInput handles keyboard input for main menu navigation
func (mmv *MainMenuView) handleMainMenuInput() error {
	kb := mmv.ctx.GetKeyBindings()

	// Click on name field to activate input
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		// Check if clicking on name input box (roughly center of screen)
		if x >= 440 && x <= 840 && y >= 250 && y <= 290 {
			mmv.inputActive = true
			return nil
		}

		// Check if clicking on New Game button
		if x >= 490 && x <= 790 && y >= 350 && y <= 410 {
			mmv.startNewGame()
			return nil
		}

		// Check if clicking on Load Game button
		if x >= 490 && x <= 790 && y >= 430 && y <= 490 {
			mmv.showLoadGameMenu()
			return nil
		}

		// Check if clicking on Settings button
		if x >= 490 && x <= 790 && y >= 510 && y <= 570 {
			mmv.showSettings()
			return nil
		}

		// Check if clicking on Quit button
		if x >= 490 && x <= 790 && y >= 590 && y <= 650 {
			return fmt.Errorf("user quit")
		}
	}

	// Keyboard navigation
	if kb.IsActionJustPressed(ActionMenuUp) {
		mmv.selectedOption--
		if mmv.selectedOption < 0 {
			mmv.selectedOption = 3
		}
	}

	if kb.IsActionJustPressed(ActionMenuDown) {
		mmv.selectedOption++
		if mmv.selectedOption > 3 {
			mmv.selectedOption = 0
		}
	}

	if kb.IsActionJustPressed(ActionMenuConfirm) && !mmv.inputActive {
		switch mmv.selectedOption {
		case 0:
			mmv.startNewGame()
		case 1:
			mmv.showLoadGameMenu()
		case 2:
			mmv.showSettings()
		case 3:
			// Quit - return error to stop the game
			return fmt.Errorf("user quit")
		}
	}

	return nil
}

// handleLoadMenuInput handles keyboard input for load game menu
func (mmv *MainMenuView) handleLoadMenuInput() {
	kb := mmv.ctx.GetKeyBindings()

	// Handle mouse clicks on save files
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		// Check back button
		if x >= 50 && x <= 200 && y >= 50 && y <= 90 {
			mmv.showLoadMenu = false
			return
		}

		// Check save file list
		startY := 150
		for i := range mmv.saveFiles {
			if x >= 200 && x <= 1080 && y >= startY+i*80 && y < startY+(i+1)*80 {
				// Check rename button (at x: 1000-1035)
				if x >= 1000 && x <= 1035 && y >= startY+i*80+5 && y < startY+i*80+35 {
					mmv.selectedSave = i
					mmv.startRename()
					return
				}

				// Check delete button (at x: 1050-1085)
				if x >= 1050 && x <= 1085 && y >= startY+i*80+5 && y < startY+i*80+35 {
					mmv.selectedSave = i
					mmv.startDelete()
					return
				}

				// Load game if clicking on the main panel area
				mmv.selectedSave = i
				mmv.loadSelectedGame()
				return
			}
		}
	}

	// Keyboard navigation
	if kb.IsActionJustPressed(ActionMenuUp) {
		mmv.selectedSave--
		if mmv.selectedSave < 0 {
			mmv.selectedSave = len(mmv.saveFiles) - 1
		}
	}

	if kb.IsActionJustPressed(ActionMenuDown) {
		mmv.selectedSave++
		if mmv.selectedSave >= len(mmv.saveFiles) {
			mmv.selectedSave = 0
		}
	}

	if kb.IsActionJustPressed(ActionMenuConfirm) && len(mmv.saveFiles) > 0 {
		mmv.loadSelectedGame()
	}

	if kb.IsActionJustPressed(ActionMenuCancel) {
		mmv.showLoadMenu = false
	}

	// Handle R key for rename
	if ebiten.IsKeyPressed(ebiten.KeyR) && len(mmv.saveFiles) > 0 {
		mmv.startRename()
	}

	// Handle D key for delete
	if ebiten.IsKeyPressed(ebiten.KeyD) && len(mmv.saveFiles) > 0 {
		mmv.startDelete()
	}

	// Handle right-click to show context menu (delete/rename)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		x, y := ebiten.CursorPosition()

		// Check if right-clicking on a save file
		startY := 150
		for i := range mmv.saveFiles {
			if x >= 200 && x <= 1080 && y >= startY+i*80 && y < startY+(i+1)*80 {
				mmv.selectedSave = i
				// Show context menu with delete and rename options
				mmv.showSaveContextMenu()
				return
			}
		}
	}
}

// handleRenameInput handles input for renaming a save file
func (mmv *MainMenuView) handleRenameInput() {
	kb := mmv.ctx.GetKeyBindings()

	// Handle character input
	for _, r := range ebiten.AppendInputChars(nil) {
		if len(mmv.renameBuffer) < 80 { // Max filename length
			mmv.renameBuffer += string(r)
		}
	}

	// Handle backspace
	if kb.IsActionJustPressed(ActionMenuDelete) {
		if len(mmv.renameBuffer) > 0 {
			mmv.renameBuffer = mmv.renameBuffer[:len(mmv.renameBuffer)-1]
		}
	}

	// Handle enter to confirm rename
	if kb.IsActionJustPressed(ActionMenuConfirm) {
		mmv.confirmRename()
	}

	// Handle escape to cancel
	if kb.IsActionJustPressed(ActionMenuCancel) {
		mmv.renameMode = false
		mmv.renameBuffer = ""
	}
}

// handleDeleteConfirm handles delete confirmation
func (mmv *MainMenuView) handleDeleteConfirm() {
	kb := mmv.ctx.GetKeyBindings()

	// Handle enter to confirm delete
	if kb.IsActionJustPressed(ActionMenuConfirm) {
		mmv.confirmDelete()
		return
	}

	// Handle escape to cancel
	if kb.IsActionJustPressed(ActionMenuCancel) {
		mmv.deleteConfirm = false
		mmv.deleteConfirmMsg = ""
	}

	// Handle mouse clicks for confirmation
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		// Yes button
		if x >= 300 && x <= 450 && y >= 400 && y <= 440 {
			mmv.confirmDelete()
			return
		}
		// No button
		if x >= 550 && x <= 700 && y >= 400 && y <= 440 {
			mmv.deleteConfirm = false
			mmv.deleteConfirmMsg = ""
		}
	}
}

// showSaveContextMenu displays delete and rename options for the selected save
func (mmv *MainMenuView) showSaveContextMenu() {
	// For simplicity, we'll prompt with R for rename and D for delete
	// This could be expanded to show a visual menu
	mmv.showError("Press R to Rename or D to Delete (Escape to Cancel)")
}

// startRename initiates rename mode for the selected save
func (mmv *MainMenuView) startRename() {
	if mmv.selectedSave >= len(mmv.saveFiles) {
		return
	}

	mmv.renameMode = true
	// Start with the current filename (without path)
	saveFile := mmv.saveFiles[mmv.selectedSave]
	mmv.renameBuffer = saveFile.Filename
}

// confirmRename confirms the rename operation
func (mmv *MainMenuView) confirmRename() {
	if mmv.selectedSave >= len(mmv.saveFiles) {
		mmv.renameMode = false
		mmv.renameBuffer = ""
		return
	}

	if mmv.renameBuffer == "" {
		mmv.showError("Filename cannot be empty")
		return
	}

	saveFile := mmv.saveFiles[mmv.selectedSave]
	saveLoad := mmv.ctx.GetSaveLoad()

	// Extract filename only (no path)
	filename := mmv.renameBuffer
	// Ensure it has the right extension
	if !strings.HasSuffix(filename, ".xsave") {
		filename += ".xsave"
	}

	if err := saveLoad.RenameSaveFile(saveFile.Path, filename); err != nil {
		mmv.showError(fmt.Sprintf("Error renaming save: %v", err))
	} else {
		mmv.showError("Save renamed successfully!")
		// Refresh the save file list
		mmv.showLoadGameMenu()
	}

	mmv.renameMode = false
	mmv.renameBuffer = ""
}

// startDelete initiates delete confirmation for the selected save
func (mmv *MainMenuView) startDelete() {
	if mmv.selectedSave >= len(mmv.saveFiles) {
		return
	}

	saveFile := mmv.saveFiles[mmv.selectedSave]
	mmv.deleteConfirm = true
	mmv.deleteConfirmMsg = fmt.Sprintf("Delete save '%s'?", saveFile.PlayerName)
}

// confirmDelete confirms the delete operation
func (mmv *MainMenuView) confirmDelete() {
	if mmv.selectedSave >= len(mmv.saveFiles) {
		mmv.deleteConfirm = false
		mmv.deleteConfirmMsg = ""
		return
	}

	saveFile := mmv.saveFiles[mmv.selectedSave]
	saveLoad := mmv.ctx.GetSaveLoad()

	if err := saveLoad.DeleteSaveFile(saveFile.Path); err != nil {
		mmv.showError(fmt.Sprintf("Error deleting save: %v", err))
	} else {
		mmv.showError("Save deleted successfully!")
		// Refresh the save file list
		mmv.showLoadGameMenu()
	}

	mmv.deleteConfirm = false
	mmv.deleteConfirmMsg = ""
}

// startNewGame starts a new game with the entered player name
func (mmv *MainMenuView) startNewGame() {
	if mmv.playerName == "" {
		mmv.showError("Please enter a player name")
		return
	}

	// Use the game context to initialize a new game
	if err := mmv.ctx.InitializeNewGame(mmv.playerName); err != nil {
		mmv.showError(fmt.Sprintf("Error creating game: %v", err))
		return
	}

	// Switch to galaxy view
	mmv.ctx.GetViewManager().SwitchTo(ViewTypeGalaxy)
}

// showSettings displays the settings menu
func (mmv *MainMenuView) showSettings() {
	vm := mmv.ctx.GetViewManager()
	vm.SwitchTo(ViewTypeSettings)

	// Set settings view to return to main menu
	if settingsView, ok := vm.GetView(ViewTypeSettings).(*SettingsView); ok {
		settingsView.SetReturnDestination(true) // true = return to main menu
	}
}

// showLoadGameMenu displays the load game menu
func (mmv *MainMenuView) showLoadGameMenu() {
	saveLoad := mmv.ctx.GetSaveLoad()
	saveFiles, err := saveLoad.ListSaveFiles()
	if err != nil {
		mmv.showError(fmt.Sprintf("Error loading saves: %v", err))
		return
	}

	if len(saveFiles) == 0 {
		mmv.showError("No save files found")
		return
	}

	// Sort by save time (newest first)
	sort.Slice(saveFiles, func(i, j int) bool {
		return saveFiles[i].SavedAt.After(saveFiles[j].SavedAt)
	})

	mmv.saveFiles = saveFiles
	mmv.selectedSave = 0
	mmv.showLoadMenu = true
}

// loadSelectedGame loads the selected save file
func (mmv *MainMenuView) loadSelectedGame() {
	if mmv.selectedSave >= len(mmv.saveFiles) {
		return
	}

	saveFile := mmv.saveFiles[mmv.selectedSave]
	if err := mmv.ctx.LoadGameFromPath(saveFile.Path); err != nil {
		mmv.showError(fmt.Sprintf("Error loading game: %v", err))
		return
	}

	// Switch to galaxy view
	mmv.ctx.GetViewManager().SwitchTo(ViewTypeGalaxy)

	// Clear error message if any
	mmv.errorMessage = ""
	mmv.errorTimer = 0
}

// showError displays an error message
func (mmv *MainMenuView) showError(message string) {
	mmv.errorMessage = message
	mmv.errorTimer = 180 // 3 seconds at 60 FPS
}

// Draw draws the main menu
func (mmv *MainMenuView) Draw(screen *ebiten.Image) {
	// Background
	screen.Fill(color.RGBA{10, 10, 20, 255})

	if mmv.showLoadMenu {
		mmv.drawLoadMenu(screen)
	} else {
		mmv.drawMainMenu(screen)
	}

	// Draw rename dialog if active
	if mmv.renameMode {
		mmv.drawRenameDialog(screen)
	}

	// Draw delete confirmation if active
	if mmv.deleteConfirm {
		mmv.drawDeleteConfirm(screen)
	}

	// Draw error message if present
	if mmv.errorMessage != "" {
		mmv.drawError(screen)
	}
}

// drawMainMenu draws the main menu screen
func (mmv *MainMenuView) drawMainMenu(screen *ebiten.Image) {
	tm := mmv.ctx.GetTickManager()
	centerX := ScreenWidth / 2

	// Title
	DrawTextCentered(screen, "XANDARIS II", centerX, 100, color.RGBA{100, 200, 255, 255}, 3.0)
	DrawTextCentered(screen, "A Space Trading Game", centerX, 150, utils.TextSecondary, 1.0)

	// Player name input
	DrawTextCentered(screen, "Player Name:", centerX, 230, utils.TextPrimary, 1.0)

	// Name input box
	nameBoxColor := utils.BackgroundDark
	if mmv.inputActive {
		nameBoxColor = color.RGBA{40, 40, 60, 255}
	}
	panel := &UIPanel{
		X:           centerX - 200,
		Y:           250,
		Width:       400,
		Height:      40,
		BgColor:     nameBoxColor,
		BorderColor: utils.PanelBorder,
	}
	panel.Draw(screen)

	nameText := mmv.playerName
	if mmv.inputActive && (tm.GetCurrentTick()/30)%2 == 0 {
		nameText += "_"
	}
	DrawTextCentered(screen, nameText, centerX, 265, utils.TextPrimary, 1.0)

	// Menu buttons
	mmv.drawButton(screen, "New Game", centerX, 380, mmv.selectedOption == 0)
	mmv.drawButton(screen, "Load Game", centerX, 460, mmv.selectedOption == 1)
	mmv.drawButton(screen, "Settings", centerX, 540, mmv.selectedOption == 2)
	mmv.drawButton(screen, "Quit", centerX, 620, mmv.selectedOption == 3)

	// Controls hint
	DrawTextCentered(screen, "Arrow Keys to Navigate | Enter to Select | Click to Interact", centerX, ScreenHeight-30, utils.TextSecondary, 0.8)
}

// drawLoadMenu draws the load game menu
func (mmv *MainMenuView) drawLoadMenu(screen *ebiten.Image) {
	// Title
	DrawTextCentered(screen, "Load Game", ScreenWidth/2, 80, color.RGBA{100, 200, 255, 255}, 2.0)

	// Back button
	backPanel := &UIPanel{
		X:           50,
		Y:           50,
		Width:       150,
		Height:      40,
		BgColor:     utils.PanelBg,
		BorderColor: utils.PanelBorder,
	}
	backPanel.Draw(screen)
	DrawText(screen, "← Back", 70, 65, utils.TextPrimary)

	// Save files list
	if len(mmv.saveFiles) == 0 {
		DrawTextCentered(screen, "No save files found", ScreenWidth/2, ScreenHeight/2, utils.TextSecondary, 1.0)
		return
	}

	startY := 150
	for i, saveFile := range mmv.saveFiles {
		y := startY + i*80

		// Don't draw off screen
		if y > ScreenHeight-100 {
			break
		}

		// Save file panel
		panelColor := utils.BackgroundDark
		if i == mmv.selectedSave {
			panelColor = color.RGBA{40, 40, 60, 255}
		}

		panel := &UIPanel{
			X:           200,
			Y:           y,
			Width:       880,
			Height:      70,
			BgColor:     panelColor,
			BorderColor: utils.PanelBorder,
		}
		panel.Draw(screen)

		// Save file info
		DrawText(screen, saveFile.PlayerName, 220, y+15, utils.TextPrimary)
		DrawText(screen, fmt.Sprintf("Game Time: %s", saveFile.GameTime), 220, y+35, utils.TextSecondary)
		DrawText(screen, fmt.Sprintf("Saved: %s", saveFile.SavedAt.Format("2006-01-02 15:04:05")), 220, y+52, utils.TextSecondary)

		// Rename button
		renameButtonX := 1000
		renameButtonY := y + 5
		renameButtonPanel := &UIPanel{
			X:           renameButtonX,
			Y:           renameButtonY,
			Width:       35,
			Height:      30,
			BgColor:     color.RGBA{50, 50, 100, 255},
			BorderColor: utils.PanelBorder,
		}
		renameButtonPanel.Draw(screen)
		DrawTextCentered(screen, "✎", renameButtonX+17, renameButtonY+15, color.RGBA{150, 200, 255, 255}, 1.0)

		// Delete button
		deleteButtonX := 1050
		deleteButtonY := y + 5
		deleteButtonPanel := &UIPanel{
			X:           deleteButtonX,
			Y:           deleteButtonY,
			Width:       35,
			Height:      30,
			BgColor:     color.RGBA{100, 30, 30, 255},
			BorderColor: color.RGBA{200, 80, 80, 255},
		}
		deleteButtonPanel.Draw(screen)
		DrawTextCentered(screen, "✕", deleteButtonX+17, deleteButtonY+15, color.RGBA{255, 150, 150, 255}, 1.0)
	}

	// Controls hint
	DrawTextCentered(screen, "Arrow Keys to Navigate | Enter to Load | R to Rename | D to Delete | Escape to Go Back", ScreenWidth/2, ScreenHeight-30, utils.TextSecondary, 0.8)
}

// drawButton draws a menu button
func (mmv *MainMenuView) drawButton(screen *ebiten.Image, text string, centerX, centerY int, selected bool) {
	buttonColor := utils.BackgroundDark
	textColor := utils.TextPrimary

	if selected {
		buttonColor = color.RGBA{50, 50, 80, 255}
		textColor = color.RGBA{150, 220, 255, 255}
	}

	panel := &UIPanel{
		X:           centerX - 150,
		Y:           centerY - 30,
		Width:       300,
		Height:      60,
		BgColor:     buttonColor,
		BorderColor: utils.PanelBorder,
	}
	panel.Draw(screen)

	DrawTextCentered(screen, text, centerX, centerY-5, textColor, 1.2)
}

// drawError draws an error message
func (mmv *MainMenuView) drawError(screen *ebiten.Image) {
	centerX := ScreenWidth / 2
	centerY := ScreenHeight - 100

	panel := &UIPanel{
		X:           centerX - 250,
		Y:           centerY - 20,
		Width:       500,
		Height:      40,
		BgColor:     color.RGBA{80, 20, 20, 255},
		BorderColor: color.RGBA{200, 50, 50, 255},
	}
	panel.Draw(screen)

	DrawTextCentered(screen, mmv.errorMessage, centerX, centerY-5, color.RGBA{255, 150, 150, 255}, 1.0)
}

// drawRenameDialog draws the rename dialog
func (mmv *MainMenuView) drawRenameDialog(screen *ebiten.Image) {
	centerX := ScreenWidth / 2
	centerY := ScreenHeight / 2

	// Semi-transparent overlay
	overlay := ebiten.NewImage(ScreenWidth, ScreenHeight)
	overlay.Fill(color.RGBA{0, 0, 0, 150})
	screen.DrawImage(overlay, nil)

	// Dialog panel
	panel := &UIPanel{
		X:           centerX - 250,
		Y:           centerY - 100,
		Width:       500,
		Height:      200,
		BgColor:     color.RGBA{20, 20, 40, 255},
		BorderColor: utils.PanelBorder,
	}
	panel.Draw(screen)

	// Title
	DrawTextCentered(screen, "Rename Save", centerX, centerY-80, utils.TextPrimary, 1.2)

	// Input box
	inputPanel := &UIPanel{
		X:           centerX - 200,
		Y:           centerY - 30,
		Width:       400,
		Height:      40,
		BgColor:     utils.BackgroundDark,
		BorderColor: utils.PanelBorder,
	}
	inputPanel.Draw(screen)

	// Display filename with cursor
	displayText := mmv.renameBuffer
	tm := mmv.ctx.GetTickManager()
	if (tm.GetCurrentTick()/30)%2 == 0 {
		displayText += "_"
	}
	DrawTextCentered(screen, displayText, centerX, centerY-15, utils.TextPrimary, 1.0)

	// Instructions
	DrawTextCentered(screen, "Enter to confirm | Escape to cancel", centerX, centerY+40, utils.TextSecondary, 0.9)
}

// drawDeleteConfirm draws the delete confirmation dialog
func (mmv *MainMenuView) drawDeleteConfirm(screen *ebiten.Image) {
	centerX := ScreenWidth / 2
	centerY := ScreenHeight / 2

	// Semi-transparent overlay
	overlay := ebiten.NewImage(ScreenWidth, ScreenHeight)
	overlay.Fill(color.RGBA{0, 0, 0, 150})
	screen.DrawImage(overlay, nil)

	// Dialog panel
	panel := &UIPanel{
		X:           centerX - 250,
		Y:           centerY - 100,
		Width:       500,
		Height:      200,
		BgColor:     color.RGBA{40, 20, 20, 255},
		BorderColor: color.RGBA{200, 50, 50, 255},
	}
	panel.Draw(screen)

	// Title
	DrawTextCentered(screen, "Confirm Delete", centerX, centerY-80, color.RGBA{255, 100, 100, 255}, 1.2)

	// Message
	DrawTextCentered(screen, mmv.deleteConfirmMsg, centerX, centerY-20, utils.TextPrimary, 1.0)

	// Yes button
	yesPanel := &UIPanel{
		X:           centerX - 180,
		Y:           centerY + 40,
		Width:       120,
		Height:      40,
		BgColor:     color.RGBA{100, 30, 30, 255},
		BorderColor: color.RGBA{200, 80, 80, 255},
	}
	yesPanel.Draw(screen)
	DrawTextCentered(screen, "Yes (Enter)", centerX-60, centerY+55, color.RGBA{255, 150, 150, 255}, 1.0)

	// No button
	noPanel := &UIPanel{
		X:           centerX + 60,
		Y:           centerY + 40,
		Width:       120,
		Height:      40,
		BgColor:     utils.PanelBg,
		BorderColor: utils.PanelBorder,
	}
	noPanel.Draw(screen)
	DrawTextCentered(screen, "No (Esc)", centerX+120, centerY+55, utils.TextPrimary, 1.0)
}

// OnEnter is called when entering this view
func (mmv *MainMenuView) OnEnter() {
	mmv.showLoadMenu = false
	mmv.inputActive = false
	mmv.errorMessage = ""
}

// OnExit is called when leaving this view
func (mmv *MainMenuView) OnExit() {
}

// GetType returns the view type
func (mmv *MainMenuView) GetType() ViewType {
	return ViewTypeMainMenu
}
