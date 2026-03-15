package views

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// MainMenuOption represents the different options in the main menu
type MainMenuOption int

const (
	// Signed-out options
	MenuSignIn MainMenuOption = iota
	MenuPlayOffline
	MenuSettings

	// Signed-in options
	MenuPlayOnline MainMenuOption = 10
	MenuAccount    MainMenuOption = 11
	MenuQuit       MainMenuOption = 12
)

// MainMenuView displays the main menu
type MainMenuView struct {
	ctx            GameContext
	playerName     string
	selectedOption int
	options        []menuEntry
	saveFiles      []SaveFileInfo
	selectedSave   int
	showLoadMenu   bool
	showAccount    bool
	inputActive    bool
	errorMessage   string
	errorTimer     int
	renameMode     bool
	renameBuffer   string
	deleteConfirm  bool
	deleteConfirmMsg string

	// State
	quitRequested bool
	loggedIn      bool
	authName   string
	authKey    string
	authPID    int
	signingIn  bool
}

type menuEntry struct {
	label  string
	option MainMenuOption
}

// NewMainMenuView creates a new main menu view
func NewMainMenuView(ctx GameContext) *MainMenuView {
	mmv := &MainMenuView{
		ctx:        ctx,
		playerName: "Player",
	}
	mmv.loadSession()
	mmv.buildMenu()
	return mmv
}

func (mmv *MainMenuView) loadSession() {
	name, key, pid := platformLoadSession()
	if key != "" && name != "" {
		mmv.loggedIn = true
		mmv.authName = name
		mmv.authKey = key
		mmv.authPID = pid
		mmv.playerName = name
	}
}

func (mmv *MainMenuView) buildMenu() {
	mmv.options = nil
	if mmv.loggedIn {
		mmv.options = []menuEntry{
			{"Play Online", MenuPlayOnline},
			{"Play Offline", MenuPlayOffline},
			{"Account", MenuAccount},
			{"Settings", MenuSettings},
			{"Quit", MenuQuit},
		}
	} else {
		mmv.options = []menuEntry{
			{"Sign in with Discord", MenuSignIn},
			{"Play Offline", MenuPlayOffline},
			{"Settings", MenuSettings},
			{"Quit", MenuQuit},
		}
	}
	mmv.selectedOption = 0
}

// Update updates the main menu view
func (mmv *MainMenuView) Update() error {
	if mmv.quitRequested {
		return fmt.Errorf("user quit")
	}

	if mmv.errorTimer > 0 {
		mmv.errorTimer--
		if mmv.errorTimer == 0 {
			mmv.errorMessage = ""
		}
	}

	// Check if sign-in completed (desktop: session file written by callback)
	if mmv.signingIn {
		name, key, pid := platformLoadSession()
		if key != "" && name != "" {
			mmv.loggedIn = true
			mmv.authName = name
			mmv.authKey = key
			mmv.authPID = pid
			mmv.playerName = name
			mmv.signingIn = false
			mmv.showError(fmt.Sprintf("Welcome, %s!", name))
			mmv.buildMenu()
		}
	}

	if mmv.inputActive && !mmv.showLoadMenu {
		mmv.handleTextInput()
	}

	if mmv.renameMode {
		mmv.handleRenameInput()
		return nil
	}

	if mmv.deleteConfirm {
		mmv.handleDeleteConfirm()
		return nil
	}

	if mmv.showAccount {
		mmv.handleAccountInput()
		return nil
	}

	if !mmv.showLoadMenu {
		if err := mmv.handleMainMenuInput(); err != nil {
			return err
		}
	} else {
		mmv.handleLoadMenuInput()
	}

	return nil
}

func (mmv *MainMenuView) handleTextInput() {
	kb := mmv.ctx.GetKeyBindings()

	for _, r := range ebiten.AppendInputChars(nil) {
		if len(mmv.playerName) < 20 {
			mmv.playerName += string(r)
		}
	}

	if kb.IsActionJustPressed(ActionMenuDelete) {
		if len(mmv.playerName) > 0 {
			mmv.playerName = mmv.playerName[:len(mmv.playerName)-1]
		}
	}

	if kb.IsActionJustPressed(ActionMenuConfirm) {
		mmv.inputActive = false
		mmv.startNewGame()
	}

	if kb.IsActionJustPressed(ActionMenuCancel) {
		mmv.inputActive = false
	}
}

func (mmv *MainMenuView) handleMainMenuInput() error {
	kb := mmv.ctx.GetKeyBindings()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		centerX := ScreenWidth / 2

		// Check name input box (only in offline flow when inputActive or about to be)
		if !mmv.loggedIn {
			if x >= centerX-200 && x <= centerX+200 && y >= 280 && y <= 320 {
				mmv.inputActive = true
				return nil
			}
		}

		// Check menu buttons
		for i, entry := range mmv.options {
			btnY := mmv.menuStartY() + i*70
			if x >= centerX-160 && x <= centerX+160 && y >= btnY-25 && y <= btnY+35 {
				mmv.selectedOption = i
				mmv.activateOption(entry.option)
				return nil
			}
		}
	}

	if kb.IsActionJustPressed(ActionMenuUp) {
		mmv.selectedOption--
		if mmv.selectedOption < 0 {
			mmv.selectedOption = len(mmv.options) - 1
		}
	}

	if kb.IsActionJustPressed(ActionMenuDown) {
		mmv.selectedOption++
		if mmv.selectedOption >= len(mmv.options) {
			mmv.selectedOption = 0
		}
	}

	if kb.IsActionJustPressed(ActionMenuConfirm) && !mmv.inputActive {
		if mmv.selectedOption < len(mmv.options) {
			mmv.activateOption(mmv.options[mmv.selectedOption].option)
		}
	}

	return nil
}

func (mmv *MainMenuView) menuStartY() int {
	if mmv.loggedIn {
		return 340
	}
	return 380
}

func (mmv *MainMenuView) activateOption(opt MainMenuOption) {
	switch opt {
	case MenuSignIn:
		mmv.signingIn = true
		// Desktop: starts local listener + opens browser
		// WASM: redirects the page to Discord OAuth
		platformStartOAuthListener(func(name, apiKey string, playerID int) {
			mmv.loggedIn = true
			mmv.authName = name
			mmv.authKey = apiKey
			mmv.authPID = playerID
			mmv.playerName = name
			mmv.signingIn = false
			mmv.buildMenu()
		})
	case MenuPlayOnline:
		mmv.connectToServer()
	case MenuPlayOffline:
		if mmv.playerName == "" {
			mmv.inputActive = true
		} else {
			mmv.startNewGame()
		}
	case MenuAccount:
		mmv.showAccount = true
	case MenuSettings:
		mmv.showSettings()
	case MenuQuit:
		mmv.quitRequested = true
	}
}

func (mmv *MainMenuView) connectToServer() {
	if mmv.authKey == "" {
		mmv.showError("Not signed in")
		return
	}

	mmv.showError("Connecting to server...")
	if err := mmv.ctx.ConnectToRemote(platformServerURL(), mmv.authName, mmv.authKey); err != nil {
		mmv.showError(fmt.Sprintf("Connection failed: %v", err))
	}
}

func (mmv *MainMenuView) handleAccountInput() {
	kb := mmv.ctx.GetKeyBindings()

	if kb.IsActionJustPressed(ActionMenuCancel) || kb.IsActionJustPressed(ActionEscape) {
		mmv.showAccount = false
		return
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		centerX := ScreenWidth / 2

		// Sign out button
		if x >= centerX-80 && x <= centerX+80 && y >= 520 && y <= 556 {
			platformClearSession()
			mmv.loggedIn = false
			mmv.authName = ""
			mmv.authKey = ""
			mmv.authPID = 0
			mmv.showAccount = false
			mmv.buildMenu()
		}

		// Back button
		if x >= centerX-80 && x <= centerX+80 && y >= 570 && y <= 606 {
			mmv.showAccount = false
		}
	}
}

func (mmv *MainMenuView) startNewGame() {
	if mmv.playerName == "" {
		mmv.showError("Please enter a player name")
		return
	}

	if err := mmv.ctx.InitializeNewGame(mmv.playerName); err != nil {
		mmv.showError(fmt.Sprintf("Error: %v", err))
		return
	}

	mmv.ctx.GetViewManager().SwitchTo(ViewTypeGalaxy)
}

func (mmv *MainMenuView) showSettings() {
	vm := mmv.ctx.GetViewManager()
	vm.SwitchTo(ViewTypeSettings)

	if settingsView, ok := vm.GetView(ViewTypeSettings).(*SettingsView); ok {
		settingsView.SetReturnDestination(true)
	}
}

func (mmv *MainMenuView) showLoadGameMenu() {
	saveLoad := mmv.ctx.GetSaveLoad()
	saveFiles, err := saveLoad.ListSaveFiles()
	if err != nil {
		mmv.showError(fmt.Sprintf("Error: %v", err))
		return
	}

	if len(saveFiles) == 0 {
		mmv.showError("No save files found")
		return
	}

	sort.Slice(saveFiles, func(i, j int) bool {
		return saveFiles[i].SavedAt.After(saveFiles[j].SavedAt)
	})

	mmv.saveFiles = saveFiles
	mmv.selectedSave = 0
	mmv.showLoadMenu = true
}

func (mmv *MainMenuView) handleLoadMenuInput() {
	kb := mmv.ctx.GetKeyBindings()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		if x >= 50 && x <= 200 && y >= 50 && y <= 90 {
			mmv.showLoadMenu = false
			return
		}

		startY := 150
		for i := range mmv.saveFiles {
			if x >= 200 && x <= 1080 && y >= startY+i*80 && y < startY+(i+1)*80 {
				if x >= 1000 && x <= 1035 && y >= startY+i*80+5 && y < startY+i*80+35 {
					mmv.selectedSave = i
					mmv.startRename()
					return
				}

				if x >= 1050 && x <= 1085 && y >= startY+i*80+5 && y < startY+i*80+35 {
					mmv.selectedSave = i
					mmv.startDelete()
					return
				}

				mmv.selectedSave = i
				mmv.loadSelectedGame()
				return
			}
		}
	}

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

	if ebiten.IsKeyPressed(ebiten.KeyR) && len(mmv.saveFiles) > 0 {
		mmv.startRename()
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) && len(mmv.saveFiles) > 0 {
		mmv.startDelete()
	}
}

func (mmv *MainMenuView) loadSelectedGame() {
	if mmv.selectedSave >= len(mmv.saveFiles) {
		return
	}

	saveFile := mmv.saveFiles[mmv.selectedSave]
	if err := mmv.ctx.LoadGameFromPath(saveFile.Path); err != nil {
		mmv.showError(fmt.Sprintf("Error: %v", err))
		return
	}

	mmv.ctx.GetViewManager().SwitchTo(ViewTypeGalaxy)
}

// Rename/delete helpers (unchanged logic)
func (mmv *MainMenuView) handleRenameInput() {
	kb := mmv.ctx.GetKeyBindings()
	for _, r := range ebiten.AppendInputChars(nil) {
		if len(mmv.renameBuffer) < 80 {
			mmv.renameBuffer += string(r)
		}
	}
	if kb.IsActionJustPressed(ActionMenuDelete) && len(mmv.renameBuffer) > 0 {
		mmv.renameBuffer = mmv.renameBuffer[:len(mmv.renameBuffer)-1]
	}
	if kb.IsActionJustPressed(ActionMenuConfirm) {
		mmv.confirmRename()
	}
	if kb.IsActionJustPressed(ActionMenuCancel) {
		mmv.renameMode = false
		mmv.renameBuffer = ""
	}
}

func (mmv *MainMenuView) handleDeleteConfirm() {
	kb := mmv.ctx.GetKeyBindings()
	if kb.IsActionJustPressed(ActionMenuConfirm) {
		mmv.confirmDelete()
		return
	}
	if kb.IsActionJustPressed(ActionMenuCancel) {
		mmv.deleteConfirm = false
		mmv.deleteConfirmMsg = ""
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if x >= 300 && x <= 450 && y >= 400 && y <= 440 {
			mmv.confirmDelete()
			return
		}
		if x >= 550 && x <= 700 && y >= 400 && y <= 440 {
			mmv.deleteConfirm = false
			mmv.deleteConfirmMsg = ""
		}
	}
}

func (mmv *MainMenuView) showSaveContextMenu() {
	mmv.showError("Press R to Rename or D to Delete")
}

func (mmv *MainMenuView) startRename() {
	if mmv.selectedSave >= len(mmv.saveFiles) {
		return
	}
	mmv.renameMode = true
	mmv.renameBuffer = mmv.saveFiles[mmv.selectedSave].Filename
}

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
	filename := mmv.renameBuffer
	if !strings.HasSuffix(filename, ".xsave") {
		filename += ".xsave"
	}
	if err := mmv.ctx.GetSaveLoad().RenameSaveFile(saveFile.Path, filename); err != nil {
		mmv.showError(fmt.Sprintf("Error: %v", err))
	} else {
		mmv.showLoadGameMenu()
	}
	mmv.renameMode = false
	mmv.renameBuffer = ""
}

func (mmv *MainMenuView) startDelete() {
	if mmv.selectedSave >= len(mmv.saveFiles) {
		return
	}
	mmv.deleteConfirm = true
	mmv.deleteConfirmMsg = fmt.Sprintf("Delete '%s'?", mmv.saveFiles[mmv.selectedSave].PlayerName)
}

func (mmv *MainMenuView) confirmDelete() {
	if mmv.selectedSave >= len(mmv.saveFiles) {
		mmv.deleteConfirm = false
		mmv.deleteConfirmMsg = ""
		return
	}
	saveFile := mmv.saveFiles[mmv.selectedSave]
	if err := mmv.ctx.GetSaveLoad().DeleteSaveFile(saveFile.Path); err != nil {
		mmv.showError(fmt.Sprintf("Error: %v", err))
	} else {
		mmv.showLoadGameMenu()
	}
	mmv.deleteConfirm = false
	mmv.deleteConfirmMsg = ""
}

func (mmv *MainMenuView) showError(message string) {
	mmv.errorMessage = message
	mmv.errorTimer = 180
}

// --- Drawing ---

var (
	bgColor      = color.RGBA{10, 12, 20, 255}
	accentColor  = color.RGBA{127, 219, 202, 255}
	cardBg       = color.RGBA{18, 22, 42, 255}
	cardBorder   = color.RGBA{30, 40, 68, 255}
	discordColor = color.RGBA{88, 101, 242, 255}
	textDim      = color.RGBA{102, 119, 136, 255}
	textLight    = color.RGBA{192, 200, 216, 255}
)

func (mmv *MainMenuView) Draw(screen *ebiten.Image) {
	screen.Fill(bgColor)

	if mmv.showLoadMenu {
		mmv.drawLoadMenu(screen)
	} else if mmv.showAccount {
		mmv.drawAccountPanel(screen)
	} else {
		mmv.drawMainMenu(screen)
	}

	if mmv.renameMode {
		mmv.drawRenameDialog(screen)
	}
	if mmv.deleteConfirm {
		mmv.drawDeleteConfirm(screen)
	}
	if mmv.errorMessage != "" {
		mmv.drawError(screen)
	}
}

func (mmv *MainMenuView) drawMainMenu(screen *ebiten.Image) {
	centerX := ScreenWidth / 2

	// Title
	DrawTextCentered(screen, "XANDARIS II", centerX, 100, accentColor, 3.0)
	DrawTextCentered(screen, "A fully-simulated space economy", centerX, 155, textDim, 1.0)

	// Logged-in status
	if mmv.loggedIn {
		DrawTextCentered(screen, fmt.Sprintf("Signed in as %s", mmv.authName), centerX, 200, accentColor, 1.0)
	} else if mmv.signingIn {
		DrawTextCentered(screen, "Waiting for Discord sign-in...", centerX, 200, textDim, 1.0)
	}

	// Player name input (only for offline play when not logged in)
	if !mmv.loggedIn {
		DrawTextCentered(screen, "Player Name", centerX, 260, textDim, 0.9)
		nameBoxColor := cardBg
		if mmv.inputActive {
			nameBoxColor = color.RGBA{30, 35, 55, 255}
		}
		panel := &UIPanel{
			X: centerX - 200, Y: 275, Width: 400, Height: 36,
			BgColor: nameBoxColor, BorderColor: cardBorder,
		}
		panel.Draw(screen)

		nameText := mmv.playerName
		if mmv.inputActive {
			tm := mmv.ctx.GetTickManager()
			if (tm.GetCurrentTick()/30)%2 == 0 {
				nameText += "_"
			}
		}
		DrawTextCentered(screen, nameText, centerX, 288, textLight, 1.0)
	}

	// Menu buttons
	startY := mmv.menuStartY()
	for i, entry := range mmv.options {
		btnY := startY + i*70
		selected := i == mmv.selectedOption
		mmv.drawMenuButton(screen, entry, centerX, btnY, selected)
	}

	// Controls hint
	DrawTextCentered(screen, "Arrow Keys / Click to Navigate", centerX, ScreenHeight-30, textDim, 0.8)
}

func (mmv *MainMenuView) drawMenuButton(screen *ebiten.Image, entry menuEntry, centerX, centerY int, selected bool) {
	btnBg := cardBg
	btnBorder := cardBorder
	textColor := textLight

	if selected {
		btnBg = color.RGBA{25, 30, 50, 255}
		btnBorder = accentColor
		textColor = accentColor
	}

	// Discord button gets special styling
	if entry.option == MenuSignIn {
		btnBg = discordColor
		if selected {
			btnBg = color.RGBA{71, 82, 196, 255}
			btnBorder = color.RGBA{120, 130, 255, 255}
		}
		textColor = color.RGBA{255, 255, 255, 255}
	}

	panel := &UIPanel{
		X: centerX - 160, Y: centerY - 25, Width: 320, Height: 50,
		BgColor: btnBg, BorderColor: btnBorder,
	}
	panel.Draw(screen)

	DrawTextCentered(screen, entry.label, centerX, centerY-5, textColor, 1.1)
}

func (mmv *MainMenuView) drawAccountPanel(screen *ebiten.Image) {
	centerX := ScreenWidth / 2

	DrawTextCentered(screen, "ACCOUNT", centerX, 100, accentColor, 2.5)

	// Info card
	panel := &UIPanel{
		X: centerX - 250, Y: 180, Width: 500, Height: 300,
		BgColor: cardBg, BorderColor: cardBorder,
	}
	panel.Draw(screen)

	// Player name
	DrawTextCentered(screen, "Player", centerX, 210, textDim, 0.9)
	DrawTextCentered(screen, mmv.authName, centerX, 235, accentColor, 1.2)

	// Player ID
	DrawTextCentered(screen, "Player ID", centerX, 275, textDim, 0.9)
	DrawTextCentered(screen, fmt.Sprintf("%d", mmv.authPID), centerX, 300, textLight, 1.0)

	// API Key
	DrawTextCentered(screen, "API Key", centerX, 340, textDim, 0.9)
	keyDisplay := mmv.authKey
	if len(keyDisplay) > 30 {
		keyDisplay = keyDisplay[:30] + "..."
	}
	DrawTextCentered(screen, keyDisplay, centerX, 365, accentColor, 0.9)

	// Full key on next line
	if len(mmv.authKey) > 30 {
		DrawTextCentered(screen, "..."+mmv.authKey[30:], centerX, 385, accentColor, 0.9)
	}

	// Sign out button
	signOutPanel := &UIPanel{
		X: centerX - 80, Y: 520, Width: 160, Height: 36,
		BgColor: color.RGBA{80, 30, 30, 255}, BorderColor: color.RGBA{150, 60, 60, 255},
	}
	signOutPanel.Draw(screen)
	DrawTextCentered(screen, "Sign Out", centerX, 533, color.RGBA{255, 150, 150, 255}, 1.0)

	// Back button
	backPanel := &UIPanel{
		X: centerX - 80, Y: 570, Width: 160, Height: 36,
		BgColor: cardBg, BorderColor: cardBorder,
	}
	backPanel.Draw(screen)
	DrawTextCentered(screen, "Back", centerX, 583, textLight, 1.0)
}

func (mmv *MainMenuView) drawLoadMenu(screen *ebiten.Image) {
	DrawTextCentered(screen, "LOAD GAME", ScreenWidth/2, 80, accentColor, 2.0)

	backPanel := &UIPanel{
		X: 50, Y: 50, Width: 150, Height: 40,
		BgColor: cardBg, BorderColor: cardBorder,
	}
	backPanel.Draw(screen)
	DrawText(screen, "<- Back", 70, 65, textLight)

	if len(mmv.saveFiles) == 0 {
		DrawTextCentered(screen, "No save files found", ScreenWidth/2, ScreenHeight/2, textDim, 1.0)
		return
	}

	startY := 150
	for i, saveFile := range mmv.saveFiles {
		y := startY + i*80
		if y > ScreenHeight-100 {
			break
		}

		panelColor := cardBg
		border := cardBorder
		if i == mmv.selectedSave {
			panelColor = color.RGBA{25, 30, 50, 255}
			border = accentColor
		}

		panel := &UIPanel{
			X: 200, Y: y, Width: 880, Height: 70,
			BgColor: panelColor, BorderColor: border,
		}
		panel.Draw(screen)

		DrawText(screen, saveFile.PlayerName, 220, y+15, textLight)
		DrawText(screen, fmt.Sprintf("Game Time: %s", saveFile.GameTime), 220, y+35, textDim)
		DrawText(screen, fmt.Sprintf("Saved: %s", saveFile.SavedAt.Format("2006-01-02 15:04")), 220, y+52, textDim)

		// Rename button
		renamePanel := &UIPanel{
			X: 1000, Y: y + 5, Width: 35, Height: 30,
			BgColor: color.RGBA{35, 35, 70, 255}, BorderColor: cardBorder,
		}
		renamePanel.Draw(screen)
		DrawTextCentered(screen, "R", 1017, y+15, accentColor, 1.0)

		// Delete button
		deletePanel := &UIPanel{
			X: 1050, Y: y + 5, Width: 35, Height: 30,
			BgColor: color.RGBA{80, 30, 30, 255}, BorderColor: color.RGBA{150, 60, 60, 255},
		}
		deletePanel.Draw(screen)
		DrawTextCentered(screen, "X", 1067, y+15, color.RGBA{255, 150, 150, 255}, 1.0)
	}

	DrawTextCentered(screen, "Enter to Load | R to Rename | D to Delete | Escape to Back", ScreenWidth/2, ScreenHeight-30, textDim, 0.8)
}

func (mmv *MainMenuView) drawError(screen *ebiten.Image) {
	centerX := ScreenWidth / 2
	centerY := ScreenHeight - 80

	panel := &UIPanel{
		X: centerX - 250, Y: centerY - 18, Width: 500, Height: 36,
		BgColor: color.RGBA{18, 22, 42, 230}, BorderColor: accentColor,
	}
	panel.Draw(screen)

	DrawTextCentered(screen, mmv.errorMessage, centerX, centerY-5, accentColor, 1.0)
}

func (mmv *MainMenuView) drawRenameDialog(screen *ebiten.Image) {
	centerX := ScreenWidth / 2
	centerY := ScreenHeight / 2

	overlay := ebiten.NewImage(ScreenWidth, ScreenHeight)
	overlay.Fill(color.RGBA{0, 0, 0, 150})
	screen.DrawImage(overlay, nil)

	panel := &UIPanel{
		X: centerX - 250, Y: centerY - 100, Width: 500, Height: 200,
		BgColor: color.RGBA{15, 18, 35, 255}, BorderColor: cardBorder,
	}
	panel.Draw(screen)

	DrawTextCentered(screen, "Rename Save", centerX, centerY-80, accentColor, 1.2)

	inputPanel := &UIPanel{
		X: centerX - 200, Y: centerY - 30, Width: 400, Height: 40,
		BgColor: bgColor, BorderColor: cardBorder,
	}
	inputPanel.Draw(screen)

	displayText := mmv.renameBuffer
	tm := mmv.ctx.GetTickManager()
	if (tm.GetCurrentTick()/30)%2 == 0 {
		displayText += "_"
	}
	DrawTextCentered(screen, displayText, centerX, centerY-15, textLight, 1.0)
	DrawTextCentered(screen, "Enter to confirm | Escape to cancel", centerX, centerY+40, textDim, 0.9)
}

func (mmv *MainMenuView) drawDeleteConfirm(screen *ebiten.Image) {
	centerX := ScreenWidth / 2
	centerY := ScreenHeight / 2

	overlay := ebiten.NewImage(ScreenWidth, ScreenHeight)
	overlay.Fill(color.RGBA{0, 0, 0, 150})
	screen.DrawImage(overlay, nil)

	panel := &UIPanel{
		X: centerX - 250, Y: centerY - 100, Width: 500, Height: 200,
		BgColor: color.RGBA{30, 15, 15, 255}, BorderColor: color.RGBA{150, 50, 50, 255},
	}
	panel.Draw(screen)

	DrawTextCentered(screen, "Confirm Delete", centerX, centerY-80, color.RGBA{255, 100, 100, 255}, 1.2)
	DrawTextCentered(screen, mmv.deleteConfirmMsg, centerX, centerY-20, textLight, 1.0)

	yesPanel := &UIPanel{
		X: centerX - 180, Y: centerY + 40, Width: 120, Height: 40,
		BgColor: color.RGBA{100, 30, 30, 255}, BorderColor: color.RGBA{200, 80, 80, 255},
	}
	yesPanel.Draw(screen)
	DrawTextCentered(screen, "Yes (Enter)", centerX-120, centerY+55, color.RGBA{255, 150, 150, 255}, 1.0)

	noPanel := &UIPanel{
		X: centerX + 60, Y: centerY + 40, Width: 120, Height: 40,
		BgColor: cardBg, BorderColor: cardBorder,
	}
	noPanel.Draw(screen)
	DrawTextCentered(screen, "No (Esc)", centerX+120, centerY+55, textLight, 1.0)
}

// OnEnter is called when entering this view
func (mmv *MainMenuView) OnEnter() {
	mmv.showLoadMenu = false
	mmv.showAccount = false
	mmv.inputActive = false
	mmv.errorMessage = ""
	mmv.loadSession()
	mmv.buildMenu()
}

// OnExit is called when leaving this view
func (mmv *MainMenuView) OnExit() {}

// GetType returns the view type
func (mmv *MainMenuView) GetType() ViewType {
	return ViewTypeMainMenu
}
