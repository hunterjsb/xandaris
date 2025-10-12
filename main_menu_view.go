package main

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/tickable"
)

const (
	ViewTypeMainMenu ViewType = "mainmenu"
)

var (
	UIBackgroundDark = color.RGBA{15, 15, 25, 255}
)

// MainMenuView displays the main menu for starting or loading games
type MainMenuView struct {
	game           *Game
	playerName     string
	selectedOption int // 0 = New Game, 1 = Load Game
	saveFiles      []SaveFileInfo
	selectedSave   int
	showLoadMenu   bool
	inputActive    bool
	errorMessage   string
	errorTimer     int
}

// NewMainMenuView creates a new main menu view
func NewMainMenuView(game *Game) *MainMenuView {
	return &MainMenuView{
		game:           game,
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

	// Handle menu navigation
	if !mmv.showLoadMenu {
		mmv.handleMainMenuInput()
	} else {
		mmv.handleLoadMenuInput()
	}

	return nil
}

// handleTextInput processes keyboard input for player name
func (mmv *MainMenuView) handleTextInput() {
	// Handle character input
	for _, r := range ebiten.AppendInputChars(nil) {
		if len(mmv.playerName) < 20 { // Max 20 characters
			mmv.playerName += string(r)
		}
	}

	// Handle backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(mmv.playerName) > 0 {
			mmv.playerName = mmv.playerName[:len(mmv.playerName)-1]
		}
	}

	// Handle enter to confirm input
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		mmv.inputActive = false
	}

	// Handle escape to cancel input
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		mmv.inputActive = false
	}
}

// handleMainMenuInput handles keyboard input for main menu navigation
func (mmv *MainMenuView) handleMainMenuInput() {
	// Click on name field to activate input
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		// Check if clicking on name input box (roughly center of screen)
		if x >= 440 && x <= 840 && y >= 250 && y <= 290 {
			mmv.inputActive = true
			return
		}

		// Check if clicking on New Game button
		if x >= 490 && x <= 790 && y >= 350 && y <= 410 {
			mmv.startNewGame()
			return
		}

		// Check if clicking on Load Game button
		if x >= 490 && x <= 790 && y >= 430 && y <= 490 {
			mmv.showLoadGameMenu()
			return
		}

		// Check if clicking on Quit button
		if x >= 490 && x <= 790 && y >= 510 && y <= 570 {
			// Could add quit functionality here
			return
		}
	}

	// Keyboard navigation
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		mmv.selectedOption--
		if mmv.selectedOption < 0 {
			mmv.selectedOption = 2
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		mmv.selectedOption++
		if mmv.selectedOption > 2 {
			mmv.selectedOption = 0
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && !mmv.inputActive {
		switch mmv.selectedOption {
		case 0:
			mmv.startNewGame()
		case 1:
			mmv.showLoadGameMenu()
		case 2:
			// Quit
		}
	}
}

// handleLoadMenuInput handles keyboard input for load game menu
func (mmv *MainMenuView) handleLoadMenuInput() {
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
				mmv.selectedSave = i
				mmv.loadSelectedGame()
				return
			}
		}
	}

	// Keyboard navigation
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		mmv.selectedSave--
		if mmv.selectedSave < 0 {
			mmv.selectedSave = len(mmv.saveFiles) - 1
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		mmv.selectedSave++
		if mmv.selectedSave >= len(mmv.saveFiles) {
			mmv.selectedSave = 0
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(mmv.saveFiles) > 0 {
		mmv.loadSelectedGame()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		mmv.showLoadMenu = false
	}
}

// startNewGame starts a new game with the entered player name
func (mmv *MainMenuView) startNewGame() {
	if mmv.playerName == "" {
		mmv.showError("Please enter a player name")
		return
	}

	// Create a new game
	newGame := NewGame()
	newGame.humanPlayer.Name = mmv.playerName

	// Update all owned entities to use the new player name
	for _, planet := range newGame.humanPlayer.OwnedPlanets {
		planet.Owner = mmv.playerName
		// Update resources owned by the player
		for _, resourceEntity := range planet.Resources {
			if resource, ok := resourceEntity.(*entities.Resource); ok {
				if resource.Owner == "Player" {
					resource.Owner = mmv.playerName
				}
			}
		}
	}

	// Replace the current game with the new one
	*mmv.game = *newGame

	// Re-initialize tickable systems with the new game context
	context := &GameSystemContext{game: mmv.game}
	tickable.InitializeAllSystems(context)

	// Switch to galaxy view
	mmv.game.viewManager.SwitchTo(ViewTypeGalaxy)
}

// showLoadGameMenu displays the load game menu
func (mmv *MainMenuView) showLoadGameMenu() {
	saveFiles, err := ListSaveFiles()
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
	loadedGame, err := LoadGameFromFile(saveFile.Path)
	if err != nil {
		mmv.showError(fmt.Sprintf("Error loading game: %v", err))
		return
	}

	// Replace the current game with the loaded one
	*mmv.game = *loadedGame

	// Re-initialize tickable systems with the new game context
	// This is crucial because the systems were initialized with the old menu game
	context := &GameSystemContext{game: mmv.game}
	tickable.InitializeAllSystems(context)

	// Switch to galaxy view
	mmv.game.viewManager.SwitchTo(ViewTypeGalaxy)
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

	// Draw error message if present
	if mmv.errorMessage != "" {
		mmv.drawError(screen)
	}
}

// drawMainMenu draws the main menu screen
func (mmv *MainMenuView) drawMainMenu(screen *ebiten.Image) {
	centerX := screenWidth / 2

	// Title
	DrawTextCentered(screen, "XANDARIS II", centerX, 100, color.RGBA{100, 200, 255, 255}, 3.0)
	DrawTextCentered(screen, "A Space Trading Game", centerX, 150, UITextSecondary, 1.0)

	// Player name input
	DrawTextCentered(screen, "Player Name:", centerX, 230, UITextPrimary, 1.0)

	// Name input box
	nameBoxColor := UIBackgroundDark
	if mmv.inputActive {
		nameBoxColor = color.RGBA{40, 40, 60, 255}
	}
	panel := &UIPanel{
		X:           centerX - 200,
		Y:           250,
		Width:       400,
		Height:      40,
		BgColor:     nameBoxColor,
		BorderColor: UIPanelBorder,
	}
	panel.Draw(screen)

	nameText := mmv.playerName
	if mmv.inputActive && (mmv.game.tickManager.GetCurrentTick()/30)%2 == 0 {
		nameText += "_"
	}
	DrawTextCentered(screen, nameText, centerX, 265, UITextPrimary, 1.0)

	// Menu buttons
	mmv.drawButton(screen, "New Game", centerX, 380, mmv.selectedOption == 0)
	mmv.drawButton(screen, "Load Game", centerX, 460, mmv.selectedOption == 1)
	mmv.drawButton(screen, "Quit", centerX, 540, mmv.selectedOption == 2)

	// Controls hint
	DrawTextCentered(screen, "Arrow Keys to Navigate | Enter to Select | Click to Interact", centerX, screenHeight-30, UITextSecondary, 0.8)
}

// drawLoadMenu draws the load game menu
func (mmv *MainMenuView) drawLoadMenu(screen *ebiten.Image) {
	// Title
	DrawTextCentered(screen, "Load Game", screenWidth/2, 80, color.RGBA{100, 200, 255, 255}, 2.0)

	// Back button
	backPanel := &UIPanel{
		X:           50,
		Y:           50,
		Width:       150,
		Height:      40,
		BgColor:     UIPanelBg,
		BorderColor: UIPanelBorder,
	}
	backPanel.Draw(screen)
	DrawText(screen, "← Back", 70, 65, UITextPrimary)

	// Save files list
	if len(mmv.saveFiles) == 0 {
		DrawTextCentered(screen, "No save files found", screenWidth/2, screenHeight/2, UITextSecondary, 1.0)
		return
	}

	startY := 150
	for i, saveFile := range mmv.saveFiles {
		y := startY + i*80

		// Don't draw off screen
		if y > screenHeight-100 {
			break
		}

		// Save file panel
		panelColor := UIBackgroundDark
		if i == mmv.selectedSave {
			panelColor = color.RGBA{40, 40, 60, 255}
		}

		panel := &UIPanel{
			X:           200,
			Y:           y,
			Width:       880,
			Height:      70,
			BgColor:     panelColor,
			BorderColor: UIPanelBorder,
		}
		panel.Draw(screen)

		// Save file info
		DrawText(screen, saveFile.PlayerName, 220, y+15, UITextPrimary)
		DrawText(screen, fmt.Sprintf("Game Time: %s", saveFile.GameTime), 220, y+35, UITextSecondary)
		DrawText(screen, fmt.Sprintf("Saved: %s", saveFile.SavedAt.Format("2006-01-02 15:04:05")), 220, y+52, UITextSecondary)
	}

	// Controls hint
	DrawTextCentered(screen, "Arrow Keys to Navigate | Enter to Load | Escape to Go Back", screenWidth/2, screenHeight-30, UITextSecondary, 0.8)
}

// drawButton draws a menu button
func (mmv *MainMenuView) drawButton(screen *ebiten.Image, text string, centerX, centerY int, selected bool) {
	buttonColor := UIBackgroundDark
	textColor := UITextPrimary

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
		BorderColor: UIPanelBorder,
	}
	panel.Draw(screen)

	DrawTextCentered(screen, text, centerX, centerY-5, textColor, 1.2)
}

// drawError draws an error message
func (mmv *MainMenuView) drawError(screen *ebiten.Image) {
	centerX := screenWidth / 2
	centerY := screenHeight - 100

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
