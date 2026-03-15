package ui

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hunterjsb/xandaris/views"
)

// CommandBar is a game console / command bar that slides up from the bottom.
// Triggered by backtick (`). Shows event feed + text input for commands.
type CommandBar struct {
	ctx          UIContext
	isOpen       bool
	input        string
	history      []string // command history
	historyIdx   int
	feedMessages []feedMessage
	maxFeed      int
	screenWidth  int
	screenHeight int
}

type feedMessage struct {
	Text  string
	Color color.RGBA
}

// NewCommandBar creates a new command bar.
func NewCommandBar(ctx UIContext, screenWidth, screenHeight int) *CommandBar {
	return &CommandBar{
		ctx:          ctx,
		maxFeed:      12,
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
	}
}

// IsOpen returns whether the command bar is active.
func (cb *CommandBar) IsOpen() bool {
	return cb.isOpen
}

// Toggle opens/closes the command bar.
func (cb *CommandBar) Toggle() {
	cb.isOpen = !cb.isOpen
	if cb.isOpen {
		cb.input = ""
		cb.refreshFeed()
	}
}

// Close closes the command bar.
func (cb *CommandBar) Close() {
	cb.isOpen = false
	cb.input = ""
}

// Update handles text input when the bar is open.
func (cb *CommandBar) Update() {
	if !cb.isOpen {
		return
	}

	// Text input
	for _, r := range ebiten.AppendInputChars(nil) {
		if r == '`' {
			continue // Don't type the toggle key
		}
		if len(cb.input) < 120 {
			cb.input += string(r)
		}
	}

	// Backspace
	kb := cb.ctx.GetKeyBindings()
	if kb.IsActionJustPressed(views.ActionMenuDelete) {
		if len(cb.input) > 0 {
			cb.input = cb.input[:len(cb.input)-1]
		}
	}

	// Enter — execute command
	if kb.IsActionJustPressed(views.ActionMenuConfirm) && cb.input != "" {
		cb.executeCommand(cb.input)
		cb.history = append(cb.history, cb.input)
		cb.historyIdx = len(cb.history)
		cb.input = ""
	}

	// Escape — close
	if kb.IsActionJustPressed(views.ActionEscape) {
		cb.Close()
	}

	// Refresh feed periodically (every 60 frames = ~1 second)
	tick := cb.ctx.GetTickManager().GetCurrentTick()
	if tick%10 == 0 {
		cb.refreshFeed()
	}
}

// Draw renders the command bar at the bottom of the screen.
func (cb *CommandBar) Draw(screen *ebiten.Image) {
	if !cb.isOpen {
		return
	}

	barHeight := 220
	barY := cb.screenHeight - barHeight
	barX := 0
	barWidth := cb.screenWidth

	// Semi-transparent background
	bgPanel := &views.UIPanel{
		X: barX, Y: barY, Width: barWidth, Height: barHeight,
		BgColor:     color.RGBA{10, 10, 25, 220},
		BorderColor: color.RGBA{80, 80, 140, 255},
	}
	bgPanel.Draw(screen)

	// Title
	views.DrawText(screen, "Command Bar  [` to close]", barX+10, barY+12, utils.TextSecondary)

	// Event feed (above the input)
	feedY := barY + 28
	for i, msg := range cb.feedMessages {
		if i >= cb.maxFeed {
			break
		}
		views.DrawText(screen, msg.Text, barX+10, feedY, msg.Color)
		feedY += 13
	}

	// Input line at the bottom
	inputY := barY + barHeight - 22

	// Input background
	inputBg := &views.UIPanel{
		X: barX + 5, Y: inputY - 4, Width: barWidth - 10, Height: 20,
		BgColor:     color.RGBA{20, 20, 40, 255},
		BorderColor: color.RGBA{100, 100, 180, 255},
	}
	inputBg.Draw(screen)

	// Prompt + text
	cursor := "_"
	tick := cb.ctx.GetTickManager().GetCurrentTick()
	if tick%20 < 10 {
		cursor = " "
	}
	inputText := fmt.Sprintf("> %s%s", cb.input, cursor)
	views.DrawText(screen, inputText, barX+12, inputY, utils.Highlight)
}

// refreshFeed loads recent events into the feed display.
func (cb *CommandBar) refreshFeed() {
	el := cb.ctx.GetEventLog()
	if el == nil {
		return
	}

	events := el.Recent(cb.maxFeed)
	cb.feedMessages = make([]feedMessage, 0, len(events))

	for _, ev := range events {
		c := utils.TextSecondary
		switch game.EventType(ev.Type) {
		case game.EventTrade:
			c = utils.SystemGreen
		case game.EventBuild:
			c = utils.SystemBlue
		case game.EventColonize:
			c = utils.SystemPurple
		}
		// Check for "event" type (economic events)
		if string(ev.Type) == "event" {
			c = utils.SystemOrange
		}

		text := fmt.Sprintf("[%s] %s", ev.Time, ev.Message)
		if len(text) > 100 {
			text = text[:97] + "..."
		}
		cb.feedMessages = append(cb.feedMessages, feedMessage{Text: text, Color: c})
	}
}

// executeCommand parses and executes a user command.
func (cb *CommandBar) executeCommand(input string) {
	input = strings.TrimSpace(input)
	lower := strings.ToLower(input)

	// Add the command itself to feed
	cb.addFeedMessage(fmt.Sprintf("> %s", input), utils.Highlight)

	// Navigation commands
	switch {
	case strings.Contains(lower, "home") || strings.Contains(lower, "take me home"):
		cb.navigateHome()
	case strings.Contains(lower, "galaxy") || strings.Contains(lower, "galaxy view"):
		cb.ctx.GetViewManager().SwitchTo(views.ViewTypeGalaxy)
		cb.addFeedMessage("Switched to galaxy view", utils.SystemGreen)
	case strings.Contains(lower, "market") || strings.Contains(lower, "show market"):
		cb.ctx.GetViewManager().SwitchTo(views.ViewTypeMarket)
		cb.addFeedMessage("Opened market view", utils.SystemGreen)
	case strings.Contains(lower, "players") || strings.Contains(lower, "directory"):
		cb.ctx.GetViewManager().SwitchTo(views.ViewTypePlayers)
		cb.addFeedMessage("Opened player directory", utils.SystemGreen)

	// Query commands
	case strings.Contains(lower, "credits") || strings.Contains(lower, "balance"):
		cb.showCredits()
	case strings.Contains(lower, "recent trades") || strings.Contains(lower, "show trades"):
		cb.showRecentTrades()
	case strings.Contains(lower, "events"):
		cb.refreshFeed()
		cb.addFeedMessage("Refreshed event feed", utils.SystemGreen)
	case strings.Contains(lower, "status") || lower == "info":
		cb.showStatus()
	case strings.Contains(lower, "help"):
		cb.showHelp()

	// Speed commands — TickSpeed is float64 under the hood
	case strings.Contains(lower, "pause"):
		cb.ctx.GetTickManager().TogglePause()
		cb.addFeedMessage("Toggled pause", utils.SystemGreen)
	case lower == "1x" || lower == "slow":
		cb.ctx.GetTickManager().SetSpeed(float64(1.0))
		cb.addFeedMessage("Speed: 1x", utils.SystemGreen)
	case lower == "2x" || lower == "normal":
		cb.ctx.GetTickManager().SetSpeed(float64(2.0))
		cb.addFeedMessage("Speed: 2x", utils.SystemGreen)
	case lower == "4x" || lower == "fast":
		cb.ctx.GetTickManager().SetSpeed(float64(4.0))
		cb.addFeedMessage("Speed: 4x", utils.SystemGreen)
	case lower == "8x" || strings.Contains(lower, "very fast"):
		cb.ctx.GetTickManager().SetSpeed(float64(8.0))
		cb.addFeedMessage("Speed: 8x", utils.SystemGreen)

	default:
		cb.addFeedMessage(fmt.Sprintf("Unknown command: %s (type 'help' for commands)", input), utils.SystemRed)
	}
}

func (cb *CommandBar) addFeedMessage(text string, c color.RGBA) {
	cb.feedMessages = append([]feedMessage{{Text: text, Color: c}}, cb.feedMessages...)
	if len(cb.feedMessages) > cb.maxFeed {
		cb.feedMessages = cb.feedMessages[:cb.maxFeed]
	}
}

func (cb *CommandBar) navigateHome() {
	player := cb.ctx.GetHumanPlayer()
	if player == nil || player.HomeSystem == nil {
		cb.addFeedMessage("No home system found", utils.SystemRed)
		return
	}
	cb.ctx.GetViewManager().SwitchTo(views.ViewTypeGalaxy)
	homeName := "home"
	if player.HomePlanet != nil {
		homeName = player.HomePlanet.Name
	}
	cb.addFeedMessage(fmt.Sprintf("Navigated to %s", homeName), utils.SystemGreen)
}

func (cb *CommandBar) showCredits() {
	player := cb.ctx.GetHumanPlayer()
	if player == nil {
		cb.addFeedMessage("No player found", utils.SystemRed)
		return
	}
	cb.addFeedMessage(fmt.Sprintf("Credits: %d | Planets: %d | Ships: %d",
		player.Credits, len(player.OwnedPlanets), len(player.OwnedShips)), utils.TextPrimary)
}

func (cb *CommandBar) showRecentTrades() {
	el := cb.ctx.GetEventLog()
	if el == nil {
		cb.addFeedMessage("No event log available", utils.SystemRed)
		return
	}
	events := el.Recent(20)
	count := 0
	for _, ev := range events {
		if ev.Type == game.EventTrade && count < 5 {
			cb.addFeedMessage(fmt.Sprintf("[%s] %s", ev.Time, ev.Message), utils.SystemGreen)
			count++
		}
	}
	if count == 0 {
		cb.addFeedMessage("No recent trades", utils.TextSecondary)
	}
}

func (cb *CommandBar) showStatus() {
	player := cb.ctx.GetHumanPlayer()
	if player == nil {
		return
	}
	tick := cb.ctx.GetTickManager().GetCurrentTick()
	gameTime := cb.ctx.GetTickManager().GetGameTimeFormatted()
	speed := cb.ctx.GetTickManager().GetSpeedString()
	paused := ""
	if cb.ctx.GetTickManager().IsPaused() {
		paused = " [PAUSED]"
	}
	cb.addFeedMessage(fmt.Sprintf("Tick %d | %s | %s%s", tick, gameTime, speed, paused), utils.TextPrimary)
	cb.addFeedMessage(fmt.Sprintf("%s: %d credits, %d planets, %d ships",
		player.Name, player.Credits, len(player.OwnedPlanets), len(player.OwnedShips)), utils.TextPrimary)
}

func (cb *CommandBar) showHelp() {
	commands := []struct {
		cmd  string
		desc string
	}{
		{"home", "Navigate to home planet"},
		{"galaxy", "Switch to galaxy view"},
		{"market", "Open market view"},
		{"players", "Open player directory"},
		{"credits", "Show your balance"},
		{"trades", "Show recent trades"},
		{"status", "Show game status"},
		{"pause", "Toggle pause"},
		{"1x/2x/4x/8x", "Set game speed"},
	}
	for i := len(commands) - 1; i >= 0; i-- {
		c := commands[i]
		cb.addFeedMessage(fmt.Sprintf("  %-12s %s", c.cmd, c.desc), utils.TextSecondary)
	}
	cb.addFeedMessage("Available commands:", utils.Highlight)
}
