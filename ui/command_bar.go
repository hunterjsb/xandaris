package ui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/tickable"
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
	userMessages []feedMessage // command output (persists until cleared)
	feedMessages []feedMessage // combined display: user msgs + events
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
		cb.userMessages = nil // Clear old command output
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

// refreshFeed rebuilds the display by merging user messages (top) with recent events.
func (cb *CommandBar) refreshFeed() {
	// Start with user messages (command output)
	cb.feedMessages = make([]feedMessage, 0, cb.maxFeed)
	for _, msg := range cb.userMessages {
		if len(cb.feedMessages) >= cb.maxFeed {
			break
		}
		cb.feedMessages = append(cb.feedMessages, msg)
	}

	// Fill remaining space with event log entries
	el := cb.ctx.GetEventLog()
	if el == nil {
		return
	}
	remaining := cb.maxFeed - len(cb.feedMessages)
	if remaining <= 0 {
		return
	}

	events := el.Recent(remaining)
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
	case strings.HasPrefix(lower, "price "):
		resource := strings.TrimPrefix(input, "price ")
		resource = strings.TrimPrefix(resource, "Price ")
		cb.showPrice(resource)
	case strings.Contains(lower, "happiness") || strings.Contains(lower, "morale"):
		cb.showHappiness()
	case strings.Contains(lower, "building") || strings.Contains(lower, "construction") || strings.Contains(lower, "queue"):
		cb.showConstruction()
	case strings.Contains(lower, "planets") || strings.Contains(lower, "colonies"):
		cb.showPlanets()
	case strings.Contains(lower, "ships") || strings.Contains(lower, "fleet"):
		cb.showShips()
	case strings.Contains(lower, "leaderboard") || strings.Contains(lower, "ranking") || strings.Contains(lower, "score"):
		cb.showLeaderboard()

	// Game action commands
	case strings.HasPrefix(lower, "build "):
		cb.handleBuild(strings.TrimPrefix(lower, "build "))
	case strings.HasPrefix(lower, "sell ") || strings.HasPrefix(lower, "buy "):
		cb.handleTrade(lower)

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
	cb.userMessages = append([]feedMessage{{Text: text, Color: c}}, cb.userMessages...)
	if len(cb.userMessages) > cb.maxFeed {
		cb.userMessages = cb.userMessages[:cb.maxFeed]
	}
	// Immediately rebuild display
	cb.refreshFeed()
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

func (cb *CommandBar) showPrice(resource string) {
	// Normalize resource name (capitalize first letter of each word)
	words := strings.Fields(resource)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	resName := strings.Join(words, " ")

	// Special case for common abbreviations
	switch strings.ToLower(resource) {
	case "rm", "rare metals", "rare metal":
		resName = "Rare Metals"
	case "he3", "helium", "helium-3":
		resName = "Helium-3"
	case "elec", "electronics":
		resName = "Electronics"
	}

	state := cb.ctx.GetState()
	if state == nil || state.Market == nil {
		cb.addFeedMessage("Market not available", utils.SystemRed)
		return
	}

	snap := state.Market.GetSnapshot()
	rm, ok := snap.Resources[resName]
	if !ok {
		cb.addFeedMessage(fmt.Sprintf("Unknown resource: %s", resName), utils.SystemRed)
		return
	}

	// Show current prices
	cb.addFeedMessage(fmt.Sprintf("%s: Buy %.0f | Sell %.0f | Base %.0f | Supply %d",
		resName, rm.BuyPrice, rm.SellPrice, rm.BasePrice, int(rm.TotalSupply)), utils.TextPrimary)

	// Show mini sparkline from price history
	if len(rm.PriceHistory) >= 5 {
		sparkline := buildSparkline(rm.PriceHistory, 20)
		cb.addFeedMessage(fmt.Sprintf("Trend: %s", sparkline), utils.SystemGreen)
	}
}

func buildSparkline(history []float64, width int) string {
	// Take last N entries
	data := history
	if len(data) > width {
		data = data[len(data)-width:]
	}

	// Find min/max
	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Build sparkline using unicode block chars
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	rng := max - min
	if rng < 0.01 {
		rng = 1
	}

	result := make([]rune, len(data))
	for i, v := range data {
		normalized := (v - min) / rng
		idx := int(normalized * float64(len(blocks)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		result[i] = blocks[idx]
	}
	return string(result)
}

func (cb *CommandBar) showHappiness() {
	player := cb.ctx.GetHumanPlayer()
	if player == nil {
		cb.addFeedMessage("No player found", utils.SystemRed)
		return
	}
	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}
		label := "Neutral"
		c := utils.TextSecondary
		if planet.Happiness >= 0.8 {
			label = "Thriving"
			c = utils.SystemGreen
		} else if planet.Happiness >= 0.6 {
			label = "Content"
			c = utils.SystemGreen
		} else if planet.Happiness >= 0.4 {
			label = "Uneasy"
			c = utils.SystemOrange
		} else if planet.Happiness >= 0.2 {
			label = "Unhappy"
			c = utils.SystemRed
		} else {
			label = "Miserable"
			c = utils.SystemRed
		}
		cb.addFeedMessage(fmt.Sprintf("%s: %s (%.0f%%) → %.1fx productivity",
			planet.Name, label, planet.Happiness*100, planet.ProductivityBonus), c)
	}
}

func (cb *CommandBar) showLeaderboard() {
	state := cb.ctx.GetState()
	if state == nil {
		return
	}

	type entry struct {
		name  string
		score int
	}
	var entries []entry

	for _, pl := range state.Players {
		if pl == nil {
			continue
		}
		var pop int64
		bldgs := 0
		stockValue := 0
		for _, planet := range pl.OwnedPlanets {
			if planet == nil {
				continue
			}
			pop += planet.Population
			bldgs += len(planet.Buildings)
			for resType, s := range planet.StoredResources {
				if s != nil && state.Market != nil {
					stockValue += int(float64(s.Amount) * state.Market.GetSellPrice(resType))
				}
			}
		}
		score := pl.Credits + stockValue + int(pop/10) + bldgs*200 + len(pl.OwnedShips)*500 + len(pl.OwnedPlanets)*2000
		entries = append(entries, entry{pl.Name, score})
	}

	// Sort descending
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].score > entries[i].score {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	for i, e := range entries {
		c := utils.TextSecondary
		if e.name == cb.ctx.GetHumanPlayer().Name {
			c = utils.Highlight
		}
		cb.addFeedMessage(fmt.Sprintf("#%d %s — %d pts", i+1, e.name, e.score), c)
	}
}

func (cb *CommandBar) showConstruction() {
	constructionSystem := tickable.GetSystemByName("Construction")
	if constructionSystem == nil {
		cb.addFeedMessage("No construction system", utils.SystemRed)
		return
	}
	cs, ok := constructionSystem.(*tickable.ConstructionSystem)
	if !ok {
		cb.addFeedMessage("Construction system unavailable", utils.SystemRed)
		return
	}

	player := cb.ctx.GetHumanPlayer()
	if player == nil {
		return
	}

	allQueues := cs.GetAllQueues()
	count := 0
	for _, items := range allQueues {
		for _, item := range items {
			if item.Owner == player.Name {
				progress := 0
				if item.TotalTicks > 0 {
					progress = 100 - (item.RemainingTicks*100)/item.TotalTicks
				}
				cb.addFeedMessage(fmt.Sprintf("%s %s — %d%% complete (%d ticks left)",
					item.Type, item.Name, progress, item.RemainingTicks), utils.SystemBlue)
				count++
			}
		}
	}
	if count == 0 {
		cb.addFeedMessage("No active construction", utils.TextSecondary)
	}
}

func (cb *CommandBar) showPlanets() {
	player := cb.ctx.GetHumanPlayer()
	if player == nil {
		return
	}
	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}
		totalRes := 0
		for _, s := range planet.StoredResources {
			if s != nil {
				totalRes += s.Amount
			}
		}
		buildings := len(planet.Buildings)
		cb.addFeedMessage(fmt.Sprintf("%s: pop %d | %d buildings | %d resources | %.0f%% happy",
			planet.Name, planet.Population, buildings, totalRes, planet.Happiness*100), utils.TextPrimary)
	}
	if len(player.OwnedPlanets) == 0 {
		cb.addFeedMessage("No planets owned", utils.TextSecondary)
	}
}

func (cb *CommandBar) showShips() {
	player := cb.ctx.GetHumanPlayer()
	if player == nil {
		return
	}
	for _, ship := range player.OwnedShips {
		if ship == nil {
			continue
		}
		status := string(ship.Status)
		extra := ""
		if ship.Status == entities.ShipStatusMoving {
			extra = fmt.Sprintf(" → sys %d (%.0f%%)", ship.TargetSystem, ship.TravelProgress*100)
		}
		cb.addFeedMessage(fmt.Sprintf("%s (%s): %s%s | Fuel %d/%d | Cargo %d/%d",
			ship.Name, ship.ShipType, status, extra,
			ship.CurrentFuel, ship.MaxFuel,
			ship.GetTotalCargo(), ship.MaxCargo), utils.TextPrimary)
	}
	if len(player.OwnedShips) == 0 {
		cb.addFeedMessage("No ships", utils.TextSecondary)
	}
}

func (cb *CommandBar) handleBuild(what string) {
	what = strings.TrimSpace(what)

	// Normalize building type
	buildingType := ""
	switch {
	case strings.Contains(what, "mine"):
		buildingType = "Mine"
	case strings.Contains(what, "trading") || strings.Contains(what, "trade post"):
		buildingType = "Trading Post"
	case strings.Contains(what, "refinery"):
		buildingType = "Refinery"
	case strings.Contains(what, "factory"):
		buildingType = "Factory"
	case strings.Contains(what, "habitat"):
		buildingType = "Habitat"
	case strings.Contains(what, "shipyard"):
		buildingType = "Shipyard"
	default:
		cb.addFeedMessage(fmt.Sprintf("Unknown building: %s", what), utils.SystemRed)
		cb.addFeedMessage("Types: mine, trading post, refinery, factory, habitat, shipyard", utils.TextSecondary)
		return
	}

	player := cb.ctx.GetHumanPlayer()
	if player == nil || len(player.OwnedPlanets) == 0 {
		cb.addFeedMessage("No planets to build on", utils.SystemRed)
		return
	}

	planet := player.OwnedPlanets[0] // Build on first planet
	cmdCh := cb.ctx.GetCommandChannel()
	if cmdCh == nil {
		cb.addFeedMessage("Command channel unavailable", utils.SystemRed)
		return
	}

	cmdCh <- game.GameCommand{
		Type: "build",
		Data: game.BuildCommandData{
			PlanetID:     planet.GetID(),
			BuildingType: buildingType,
		},
	}
	cb.addFeedMessage(fmt.Sprintf("Queued %s on %s", buildingType, planet.Name), utils.SystemGreen)
}

func (cb *CommandBar) handleTrade(input string) {
	// Parse: "buy 10 iron" or "sell 50 fuel"
	parts := strings.Fields(input)
	if len(parts) < 3 {
		cb.addFeedMessage("Usage: buy/sell <qty> <resource>", utils.SystemRed)
		return
	}

	action := parts[0]
	qty, err := strconv.Atoi(parts[1])
	if err != nil {
		cb.addFeedMessage("Invalid quantity", utils.SystemRed)
		return
	}

	resource := strings.Join(parts[2:], " ")
	// Normalize resource name
	words := strings.Fields(resource)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	resource = strings.Join(words, " ")
	switch strings.ToLower(resource) {
	case "rm", "rare metals", "rare metal":
		resource = "Rare Metals"
	case "he3", "helium", "helium-3":
		resource = "Helium-3"
	case "elec", "electronics":
		resource = "Electronics"
	}

	isBuy := action == "buy"

	player := cb.ctx.GetHumanPlayer()
	if player == nil || len(player.OwnedPlanets) == 0 {
		cb.addFeedMessage("No planet to trade from", utils.SystemRed)
		return
	}

	cmdCh := cb.ctx.GetCommandChannel()
	if cmdCh == nil {
		cb.addFeedMessage("Command channel unavailable", utils.SystemRed)
		return
	}

	cmdCh <- game.GameCommand{
		Type: "trade",
		Data: game.TradeCommandData{
			Resource: resource,
			Quantity: qty,
			Buy:      isBuy,
			PlanetID: player.OwnedPlanets[0].GetID(),
		},
	}

	verb := "Selling"
	if isBuy {
		verb = "Buying"
	}
	cb.addFeedMessage(fmt.Sprintf("%s %d %s...", verb, qty, resource), utils.SystemGreen)
}

func (cb *CommandBar) showHelp() {
	commands := []struct {
		cmd  string
		desc string
	}{
		{"home", "Navigate to home planet"},
		{"galaxy/market/players", "Switch views"},
		{"credits", "Show your balance"},
		{"planets", "Show all your planets"},
		{"ships", "Show your ships"},
		{"trades", "Show recent trades"},
		{"price <res>", "Price + sparkline trend"},
		{"happiness", "Planet happiness summary"},
		{"leaderboard", "Player rankings"},
		{"building", "Construction queue"},
		{"build <type>", "Build (mine/factory/etc)"},
		{"buy/sell <n> <res>", "Trade resources"},
		{"status", "Game status"},
		{"pause / 1x-8x", "Speed control"},
	}
	for i := len(commands) - 1; i >= 0; i-- {
		c := commands[i]
		cb.addFeedMessage(fmt.Sprintf("  %-12s %s", c.cmd, c.desc), utils.TextSecondary)
	}
	cb.addFeedMessage("Available commands:", utils.Highlight)
}
