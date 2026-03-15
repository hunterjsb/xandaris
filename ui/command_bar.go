package ui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hunterjsb/xandaris/views"
)

// MessageType categorizes chat feed messages.
type MessageType int

const (
	MsgUser      MessageType = iota // Player input
	MsgAssistant                    // LLM response
	MsgEvent                        // Game event (trade, build, etc.)
	MsgSystem                       // System messages (status, errors)
	MsgAction                       // Agent action results
)

// ChatMessage is a single entry in the chronological feed.
type ChatMessage struct {
	Tick   int64
	Type   MessageType
	Sender string
	Text   string
	Color  color.RGBA
}

// ChatMode determines what the command bar shows and how input is routed.
type ChatMode int

const (
	ModeAgent     ChatMode = iota // LLM agent + slash commands
	ModeEventsAll                 // Game events + multiplayer chat
	ModeChatOnly                  // Multiplayer chat only
)

// CommandBar is a game console / chat / command bar.
// Single chronological feed with push-based events — no polling.
type CommandBar struct {
	ctx          UIContext
	mu           sync.Mutex
	isOpen       bool
	input        string
	history      []string                 // command history
	historyIdx   int
	feed         []ChatMessage            // single chronological feed
	chatHistory  []map[string]interface{} // conversation context for LLM
	scrollOffset int
	screenWidth  int
	screenHeight int
	serverURL    string
	apiKey       string
	mode         ChatMode // current Tab mode
	copyFlash    int
	tabHeld      bool
	copyHeld     bool
}

// NewCommandBar creates a new command bar.
func NewCommandBar(ctx UIContext, screenWidth, screenHeight int) *CommandBar {
	return &CommandBar{
		ctx:          ctx,
		feed:         make([]ChatMessage, 0, 200),
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		serverURL:    "http://localhost:8080",
		mode:         ModeAgent,
	}
}

// Init subscribes to EventLog and ChatLog for push-based delivery.
func (cb *CommandBar) Init() {
	// Subscribe to game events
	if el := cb.ctx.GetEventLog(); el != nil {
		// Backfill
		events := el.Recent(10)
		cb.mu.Lock()
		for i := len(events) - 1; i >= 0; i-- {
			ev := events[i]
			cb.feed = append(cb.feed, ChatMessage{
				Tick: ev.Tick, Type: MsgEvent, Sender: ev.Player,
				Text: fmt.Sprintf("[%s] %s", ev.Time, ev.Message), Color: eventColor(ev.Type),
			})
		}
		cb.mu.Unlock()

		el.Subscribe(func(ev game.GameEvent) {
			if cb.mode == ModeAgent || cb.mode == ModeChatOnly {
				return // events only shown in ModeEventsAll
			}
			cb.mu.Lock()
			cb.feed = append(cb.feed, ChatMessage{
				Tick: ev.Tick, Type: MsgEvent, Sender: ev.Player,
				Text: fmt.Sprintf("[%s] %s", ev.Time, ev.Message), Color: eventColor(ev.Type),
			})
			if len(cb.feed) > 200 {
				cb.feed = cb.feed[len(cb.feed)-200:]
		}
		cb.mu.Unlock()
	})
	}

	// Subscribe to multiplayer chat
	if cl := cb.ctx.GetChatLog(); cl != nil {
		cl.Subscribe(func(msg game.ChatMsg) {
			if cb.mode == ModeAgent {
				return // multiplayer chat not shown in agent mode
			}
			cb.mu.Lock()
			cb.feed = append(cb.feed, ChatMessage{
				Tick: msg.Tick, Type: MsgUser, Sender: msg.Player,
				Text:  fmt.Sprintf("<%s> %s", msg.Player, msg.Message),
				Color: color.RGBA{200, 200, 220, 255},
			})
			if len(cb.feed) > 200 {
				cb.feed = cb.feed[len(cb.feed)-200:]
			}
			cb.mu.Unlock()
		})
	}
}

func eventColor(t game.EventType) color.RGBA {
	switch t {
	case game.EventTrade:
		return color.RGBA{70, 130, 70, 255}
	case game.EventBuild:
		return color.RGBA{70, 90, 150, 255}
	case game.EventColonize, game.EventJoin:
		return color.RGBA{130, 70, 150, 255}
	case game.EventAlert:
		return color.RGBA{180, 70, 70, 255}
	default:
		return color.RGBA{70, 80, 100, 255}
	}
}

// drawMinimized renders a slim event ticker at the bottom when closed.
func (cb *CommandBar) drawMinimized(screen *ebiten.Image) {
	lineHeight := 13
	maxLines := 4
	barX := 0
	barWidth := cb.screenWidth

	cb.mu.Lock()
	// Grab the last N event messages
	var lines []ChatMessage
	for i := len(cb.feed) - 1; i >= 0 && len(lines) < maxLines; i-- {
		if cb.feed[i].Type == MsgEvent {
			lines = append(lines, cb.feed[i])
		}
	}
	cb.mu.Unlock()

	if len(lines) == 0 {
		return
	}

	barHeight := len(lines)*lineHeight + 8
	barY := cb.screenHeight - barHeight

	// Subtle semi-transparent background
	bgPanel := &views.UIPanel{
		X: barX, Y: barY, Width: barWidth, Height: barHeight,
		BgColor:     color.RGBA{5, 7, 15, 180},
		BorderColor: color.RGBA{20, 28, 45, 150},
	}
	bgPanel.Draw(screen)

	// Draw events bottom-up (newest at bottom)
	y := cb.screenHeight - 6
	for _, msg := range lines {
		y -= lineHeight
		// Dim the color for minimized view
		dimmed := msg.Color
		dimmed.A = 150
		views.DrawText(screen, msg.Text, barX+8, y, dimmed)
	}
}

func (cb *CommandBar) SetServerURL(url string) { cb.serverURL = url }
func (cb *CommandBar) SetAPIKey(key string)     { cb.apiKey = key }
func (cb *CommandBar) IsOpen() bool             { return cb.isOpen }

func (cb *CommandBar) Toggle() {
	cb.isOpen = !cb.isOpen
	if cb.isOpen {
		cb.input = ""
		cb.scrollOffset = 0
	}
}

func (cb *CommandBar) Close() {
	cb.isOpen = false
	cb.input = ""
}

func (cb *CommandBar) Update() {
	if !cb.isOpen {
		return
	}

	for _, r := range ebiten.AppendInputChars(nil) {
		if r == '`' {
			continue
		}
		if len(cb.input) < 120 {
			cb.input += string(r)
		}
	}

	kb := cb.ctx.GetKeyBindings()
	if kb.IsActionJustPressed(views.ActionMenuDelete) && len(cb.input) > 0 {
		cb.input = cb.input[:len(cb.input)-1]
	}

	if kb.IsActionJustPressed(views.ActionMenuConfirm) && cb.input != "" {
		cb.executeCommand(cb.input)
		cb.history = append(cb.history, cb.input)
		cb.historyIdx = len(cb.history)
		cb.input = ""
	}

	if kb.IsActionJustPressed(views.ActionEscape) {
		cb.Close()
	}

	// Tab — cycle mode: Agent → Events+Chat → Chat Only → Agent
	if ebiten.IsKeyPressed(ebiten.KeyTab) && !cb.tabHeld {
		cb.mode = (cb.mode + 1) % 3
		modeNames := []string{"Agent", "Events + Chat", "Chat Only"}
		cb.addFeedMessage(fmt.Sprintf("Mode: %s", modeNames[cb.mode]), utils.TextSecondary)
	}
	cb.tabHeld = ebiten.IsKeyPressed(ebiten.KeyTab)

	// Ctrl+C — copy
	if ebiten.IsKeyPressed(ebiten.KeyControl) && ebiten.IsKeyPressed(ebiten.KeyC) && !cb.copyHeld {
		cb.copyFeedToClipboard()
	}
	cb.copyHeld = ebiten.IsKeyPressed(ebiten.KeyControl) && ebiten.IsKeyPressed(ebiten.KeyC)

	if cb.copyFlash > 0 {
		cb.copyFlash--
	}

	// Scroll
	_, wheelY := ebiten.Wheel()
	if wheelY != 0 {
		cb.mu.Lock()
		cb.scrollOffset -= int(wheelY * 2)
		if cb.scrollOffset < 0 {
			cb.scrollOffset = 0
		}
		maxScroll := len(cb.feed) - 10
		if maxScroll < 0 {
			maxScroll = 0
		}
		if cb.scrollOffset > maxScroll {
			cb.scrollOffset = maxScroll
		}
		cb.mu.Unlock()
	}
}

func (cb *CommandBar) Draw(screen *ebiten.Image) {
	if !cb.isOpen {
		cb.drawMinimized(screen)
		return
	}

	barHeight := 170
	barY := cb.screenHeight - barHeight
	barX := 0
	barWidth := cb.screenWidth
	lineHeight := 14

	// Background
	bgPanel := &views.UIPanel{
		X: barX, Y: barY, Width: barWidth, Height: barHeight,
		BgColor:     color.RGBA{8, 10, 20, 235},
		BorderColor: color.RGBA{30, 40, 68, 255},
	}
	bgPanel.Draw(screen)

	// Input bar at bottom
	inputY := barY + barHeight - 20
	inputBg := &views.UIPanel{
		X: barX + 4, Y: inputY - 3, Width: barWidth - 8, Height: 18,
		BgColor:     color.RGBA{16, 18, 35, 255},
		BorderColor: color.RGBA{35, 45, 70, 255},
	}
	inputBg.Draw(screen)

	cursor := "_"
	if cb.ctx.GetTickManager().GetCurrentTick()%20 < 10 {
		cursor = " "
	}
	views.DrawText(screen, fmt.Sprintf("> %s%s", cb.input, cursor), barX+10, inputY, utils.Highlight)

	// Hints right-aligned on input bar
	dimColor := color.RGBA{50, 60, 80, 255}
	modeLabel := []string{"Agent", "Events+Chat", "Chat"}[cb.mode]
	hints := fmt.Sprintf("[`]close [Tab]%s [Ctrl+C]copy [/help]", modeLabel)
	views.DrawText(screen, hints, barX+barWidth-len(hints)*6-10, inputY, dimColor)

	if cb.copyFlash > 0 {
		views.DrawText(screen, "Copied!", barX+barWidth-50, barY+4, utils.SystemGreen)
	}

	// Feed: bottom-up rendering from the feed slice
	cb.mu.Lock()
	feedSnapshot := make([]ChatMessage, len(cb.feed))
	copy(feedSnapshot, cb.feed)
	scrollOff := cb.scrollOffset
	cb.mu.Unlock()

	feedBottom := inputY - 22 // enough gap so text doesn't overlap input bar
	feedTop := barY + 6
	maxVisible := (feedBottom - feedTop) / lineHeight

	total := len(feedSnapshot)
	endIdx := total - scrollOff
	if endIdx < 0 {
		endIdx = 0
	}
	startIdx := endIdx - maxVisible
	if startIdx < 0 {
		startIdx = 0
	}

	y := feedBottom
	for i := endIdx - 1; i >= startIdx; i-- {
		if y < feedTop {
			break
		}
		views.DrawText(screen, feedSnapshot[i].Text, barX+10, y, feedSnapshot[i].Color)
		y -= lineHeight
	}

	if scrollOff > 0 {
		views.DrawText(screen, fmt.Sprintf("^ %d more ^", scrollOff), barX+barWidth/2-30, feedTop, dimColor)
	}
}

// executeCommand routes input based on the current mode.
func (cb *CommandBar) executeCommand(input string) {
	input = strings.TrimSpace(input)

	// Slash commands work in all modes
	if strings.HasPrefix(input, "/") {
		cb.addFeedMessage(fmt.Sprintf("> %s", input), utils.Highlight)
		cb.executeSlashCommand(strings.TrimPrefix(input, "/"))
		return
	}

	switch cb.mode {
	case ModeAgent:
		cb.addFeedMessage(fmt.Sprintf("> %s", input), utils.Highlight)
		cb.sendToChat(input)
	case ModeEventsAll, ModeChatOnly:
		// Send as multiplayer chat
		cb.sendMultiplayerChat(input)
	}
}

// sendMultiplayerChat sends a message to the multiplayer chat endpoint.
func (cb *CommandBar) sendMultiplayerChat(message string) {
	player := ""
	if human := cb.ctx.GetHumanPlayer(); human != nil {
		player = human.Name
	}
	cb.addFeedMessage(fmt.Sprintf("<%s> %s", player, message), color.RGBA{200, 200, 220, 255})

	go func() {
		reqBody := map[string]interface{}{"message": message}
		bodyBytes, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", cb.serverURL+"/api/chat/send", strings.NewReader(string(bodyBytes)))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if cb.apiKey != "" {
			req.Header.Set("X-API-Key", cb.apiKey)
		}
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			cb.addFeedMessage("Failed to send message", utils.SystemRed)
			return
		}
		resp.Body.Close()
	}()
}

// executeSlashCommand handles /prefixed local commands.
func (cb *CommandBar) executeSlashCommand(input string) {
	lower := strings.ToLower(strings.TrimSpace(input))

	switch {
	// Navigation
	case lower == "home":
		cb.navigateHome()
	case lower == "galaxy":
		cb.ctx.GetViewManager().SwitchTo(views.ViewTypeGalaxy)
		cb.addFeedMessage("Switched to galaxy view", utils.SystemGreen)
	case lower == "market":
		cb.ctx.GetViewManager().SwitchTo(views.ViewTypeMarket)
		cb.addFeedMessage("Opened market view", utils.SystemGreen)
	case lower == "players":
		cb.ctx.GetViewManager().SwitchTo(views.ViewTypePlayers)
		cb.addFeedMessage("Opened player directory", utils.SystemGreen)

	// Queries
	case lower == "credits" || lower == "balance":
		cb.showCredits()
	case lower == "trades":
		cb.showRecentTrades()
	case lower == "events" || lower == "clear":
		cb.mu.Lock()
		cb.feed = cb.feed[:0]
		cb.mu.Unlock()
	case lower == "status" || lower == "info":
		cb.showStatus()
	case strings.HasPrefix(lower, "price "):
		cb.showPrice(strings.TrimPrefix(lower, "price "))
	case lower == "happiness" || lower == "morale":
		cb.showHappiness()
	case lower == "building" || lower == "construction" || lower == "queue":
		cb.showConstruction()
	case lower == "planets" || lower == "colonies":
		cb.showPlanets()
	case lower == "ships" || lower == "fleet":
		cb.showShips()
	case lower == "leaderboard" || lower == "score":
		cb.showLeaderboard()
	case lower == "orders":
		cb.showOrders()
	case lower == "scarcity" || lower == "economy" || lower == "shortages":
		cb.showScarcity()
	case lower == "deliveries" || lower == "cargo":
		cb.showDeliveries()
	case lower == "power":
		cb.showPower()

	// Game actions
	case strings.HasPrefix(lower, "build "):
		cb.handleBuild(strings.TrimPrefix(lower, "build "))
	case strings.HasPrefix(lower, "sell ") || strings.HasPrefix(lower, "buy "):
		cb.handleTrade(lower)
	case strings.HasPrefix(lower, "order "):
		cb.handleOrder(strings.TrimPrefix(lower, "order "))

	// Speed control
	case lower == "pause":
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
	case lower == "8x":
		cb.ctx.GetTickManager().SetSpeed(float64(8.0))
		cb.addFeedMessage("Speed: 8x", utils.SystemGreen)

	case lower == "help":
		cb.showHelp()

	default:
		cb.addFeedMessage(fmt.Sprintf("Unknown command: /%s (try /help)", input), utils.SystemRed)
	}
}

func (cb *CommandBar) copyFeedToClipboard() {
	cb.mu.Lock()
	var sb strings.Builder
	for _, msg := range cb.feed {
		sb.WriteString(msg.Text)
		sb.WriteString("\n")
	}
	cb.mu.Unlock()
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return
	}
	copyToClipboard(text)
	cb.copyFlash = 90
}

// handleNavigateAction processes a navigate:target:id action from the agent.
func (cb *CommandBar) handleNavigateAction(action string) {
	parts := strings.SplitN(action, ":", 3)
	if len(parts) < 2 {
		return
	}
	target := parts[1]
	id := 0
	if len(parts) >= 3 {
		id, _ = strconv.Atoi(parts[2])
	}

	vm := cb.ctx.GetViewManager()

	switch target {
	case "galaxy":
		vm.SwitchTo(views.ViewTypeGalaxy)
		cb.addFeedMessage("  -> Navigated to galaxy map", utils.SystemGreen)
	case "system":
		// Switch to system view for the given system ID
		if systemView, ok := vm.GetView(views.ViewTypeSystem).(interface {
			SetSystem(*entities.System)
		}); ok {
			for _, sys := range cb.ctx.GetSystems() {
				if sys.ID == id {
					systemView.SetSystem(sys)
					vm.SwitchTo(views.ViewTypeSystem)
					cb.addFeedMessage(fmt.Sprintf("  -> Navigated to %s", sys.Name), utils.SystemGreen)
					return
				}
			}
		}
		cb.addFeedMessage(fmt.Sprintf("  -> System %d not found", id), utils.SystemRed)
	case "planet":
		if planetView, ok := vm.GetView(views.ViewTypePlanet).(interface {
			SetPlanet(*entities.Planet)
		}); ok {
			// First try: match by ID in all systems
			for _, sys := range cb.ctx.GetSystems() {
				for _, e := range sys.Entities {
					if p, ok := e.(*entities.Planet); ok && p.GetID() == id {
						planetView.SetPlanet(p)
						vm.SwitchTo(views.ViewTypePlanet)
						cb.addFeedMessage(fmt.Sprintf("  -> Navigated to %s", p.Name), utils.SystemGreen)
						return
					}
				}
			}
			// Fallback: match by owned planet ID (remote play — local IDs differ)
			if human := cb.ctx.GetHumanPlayer(); human != nil {
				for _, p := range human.OwnedPlanets {
					if p != nil && p.GetID() == id {
						planetView.SetPlanet(p)
						vm.SwitchTo(views.ViewTypePlanet)
						cb.addFeedMessage(fmt.Sprintf("  -> Navigated to %s", p.Name), utils.SystemGreen)
						return
					}
				}
				// Last resort: just go to first owned planet
				if len(human.OwnedPlanets) > 0 && human.OwnedPlanets[0] != nil {
					planetView.SetPlanet(human.OwnedPlanets[0])
					vm.SwitchTo(views.ViewTypePlanet)
					cb.addFeedMessage(fmt.Sprintf("  -> Navigated to %s", human.OwnedPlanets[0].Name), utils.SystemGreen)
					return
				}
			}
		}
		cb.addFeedMessage(fmt.Sprintf("  -> Planet %d not found", id), utils.SystemRed)
	case "market":
		vm.SwitchTo(views.ViewTypeMarket)
		cb.addFeedMessage("  -> Opened market", utils.SystemGreen)
	case "players":
		vm.SwitchTo(views.ViewTypePlayers)
		cb.addFeedMessage("  -> Opened player directory", utils.SystemGreen)
	}
}

// sendToChat sends a natural language message to the server's LLM chat endpoint.
func (cb *CommandBar) sendToChat(message string) {
	cb.addFeedMessage("Thinking...", utils.TextSecondary)

	// Run in goroutine to avoid blocking the game loop
	go func() {
		resp, err := cb.callChatAPI(message)
		if err != nil {
			cb.addFeedMessage(fmt.Sprintf("Agent error: %v", err), utils.SystemRed)
			return
		}
		cb.addFeedMessage(resp, utils.SystemGreen)

		// Maintain conversation context (keep last 10 turns)
		cb.chatHistory = append(cb.chatHistory,
			map[string]interface{}{"role": "user", "content": message},
			map[string]interface{}{"role": "assistant", "content": resp},
		)
		if len(cb.chatHistory) > 20 { // 10 turns = 20 messages
			cb.chatHistory = cb.chatHistory[len(cb.chatHistory)-20:]
		}
	}()
}

// callChatAPI sends a message to POST /api/chat and returns the response.
func (cb *CommandBar) callChatAPI(message string) (string, error) {
	// Build the request body with conversation history
	reqBody := map[string]interface{}{
		"message": message,
		"history": cb.chatHistory,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	body := string(bodyBytes)

	req, err := http.NewRequest("POST", cb.serverURL+"/api/chat", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if cb.apiKey != "" {
		req.Header.Set("X-API-Key", cb.apiKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to reach server: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Data  struct {
			Response string   `json:"response"`
			Actions  []string `json:"actions"`
		} `json:"data"`
	}

	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Show what we actually got for debugging
		snippet := string(respBody)
		if len(snippet) > 80 {
			snippet = snippet[:80]
		}
		return "", fmt.Errorf("bad response: %s", snippet)
	}
	if !result.OK {
		return "", fmt.Errorf("%s", result.Error)
	}

	// Process actions — some are UI navigation commands
	for _, action := range result.Data.Actions {
		if strings.HasPrefix(action, "navigate:") {
			cb.handleNavigateAction(action)
		} else {
			cb.addFeedMessage(fmt.Sprintf("  -> %s", action), utils.SystemBlue)
		}
	}

	return result.Data.Response, nil
}

func (cb *CommandBar) addFeedMessage(text string, c color.RGBA) {
	maxChars := (cb.screenWidth - 20) / 6
	if maxChars < 40 {
		maxChars = 40
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Word-wrap long messages
	for len(text) > maxChars {
		breakAt := maxChars
		for i := maxChars; i > maxChars/2; i-- {
			if i < len(text) && text[i] == ' ' {
				breakAt = i
				break
			}
		}
		cb.feed = append(cb.feed, ChatMessage{Type: MsgSystem, Text: text[:breakAt], Color: c})
		text = "  " + strings.TrimLeft(text[breakAt:], " ")
	}
	if text != "" {
		cb.feed = append(cb.feed, ChatMessage{Type: MsgSystem, Text: text, Color: c})
	}
	if len(cb.feed) > 200 {
		cb.feed = cb.feed[len(cb.feed)-200:]
	}
	cb.scrollOffset = 0
}

func (cb *CommandBar) navigateHome() {
	player := cb.ctx.GetHumanPlayer()
	if player == nil {
		cb.addFeedMessage("No player found", utils.SystemRed)
		return
	}

	// Try to navigate to home planet
	vm := cb.ctx.GetViewManager()
	if player.HomePlanet != nil {
		if planetView, ok := vm.GetView(views.ViewTypePlanet).(interface {
			SetPlanet(*entities.Planet)
		}); ok {
			planetView.SetPlanet(player.HomePlanet)
			vm.SwitchTo(views.ViewTypePlanet)
			cb.addFeedMessage(fmt.Sprintf("Navigated to %s", player.HomePlanet.Name), utils.SystemGreen)
			return
		}
	}
	// Fallback: first owned planet
	if len(player.OwnedPlanets) > 0 && player.OwnedPlanets[0] != nil {
		if planetView, ok := vm.GetView(views.ViewTypePlanet).(interface {
			SetPlanet(*entities.Planet)
		}); ok {
			planetView.SetPlanet(player.OwnedPlanets[0])
			vm.SwitchTo(views.ViewTypePlanet)
			cb.addFeedMessage(fmt.Sprintf("Navigated to %s", player.OwnedPlanets[0].Name), utils.SystemGreen)
			return
		}
	}
	// Last fallback: galaxy view
	vm.SwitchTo(views.ViewTypeGalaxy)
	cb.addFeedMessage("Navigated to galaxy", utils.SystemGreen)
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

func (cb *CommandBar) showDeliveries() {
	state := cb.ctx.GetState()
	if state == nil {
		return
	}
	// Access delivery manager through the server
	dm := cb.ctx.GetDeliveryManager()
	if dm == nil {
		cb.addFeedMessage("No delivery system", utils.TextSecondary)
		return
	}
	deliveries := dm.GetActiveDeliveries()
	if len(deliveries) == 0 {
		cb.addFeedMessage("No active deliveries", utils.TextSecondary)
		return
	}
	for _, d := range deliveries {
		cb.addFeedMessage(fmt.Sprintf("#%d %s→%s: %d %s (ship %d)",
			d.ID, d.SellerName, d.BuyerName, d.Quantity, d.Resource, d.ShipID), utils.SystemBlue)
	}
}

func (cb *CommandBar) showPower() {
	player := cb.ctx.GetHumanPlayer()
	if player == nil {
		cb.addFeedMessage("No player", utils.SystemRed)
		return
	}
	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}
		ratio := planet.GetPowerRatio()
		c := utils.SystemGreen
		status := "OK"
		if ratio < 0.5 {
			c = utils.SystemRed
			status = "CRITICAL"
		} else if ratio < 0.8 {
			c = utils.SystemOrange
			status = "Low"
		}

		// Count power buildings
		gens := 0
		reactors := 0
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok {
				if b.BuildingType == "Generator" {
					gens++
				} else if b.BuildingType == "Fusion Reactor" {
					reactors++
				}
			}
		}

		fuelStored := planet.GetStoredAmount("Fuel")
		he3Stored := planet.GetStoredAmount("Helium-3")
		cb.addFeedMessage(fmt.Sprintf("%s: %.0f/%.0f MW (%s) | %d gen %d fusion | Fuel:%d He3:%d",
			planet.Name, planet.PowerGenerated, planet.PowerConsumed, status,
			gens, reactors, fuelStored, he3Stored), c)
	}
}

func (cb *CommandBar) showScarcity() {
	state := cb.ctx.GetState()
	if state == nil || state.Market == nil {
		cb.addFeedMessage("Market not available", utils.SystemRed)
		return
	}

	snap := state.Market.GetSnapshot()

	type resInfo struct {
		name     string
		ratio    float64
		scarcity string
	}
	var items []resInfo
	for name, rm := range snap.Resources {
		ratio := 1.0
		if rm.BasePrice > 0 {
			ratio = rm.CurrentPrice / rm.BasePrice
		}
		scarcity := "OK"
		if ratio > 3.0 {
			scarcity = "CRITICAL"
		} else if ratio > 1.5 {
			scarcity = "Scarce"
		} else if ratio < 0.3 {
			scarcity = "Surplus"
		}
		items = append(items, resInfo{name, ratio, scarcity})
	}

	// Sort by ratio descending (most scarce first)
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].ratio > items[i].ratio {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	for _, item := range items {
		c := utils.TextSecondary
		advice := ""
		if item.scarcity == "CRITICAL" {
			c = utils.SystemRed
			advice = " — build mines!"
		} else if item.scarcity == "Scarce" {
			c = utils.SystemOrange
			advice = " — opportunity to sell"
		} else if item.scarcity == "Surplus" {
			c = utils.SystemGreen
			advice = " — buy cheap"
		}
		cb.addFeedMessage(fmt.Sprintf("%s: %.1fx base (%s)%s", item.name, item.ratio, item.scarcity, advice), c)
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
			for _, s := range planet.StoredResources {
				if s != nil {
					stockValue += s.Amount // Simple count — base prices vary too much for a quick calc
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
	case strings.Contains(what, "generator") || strings.Contains(what, "power plant"):
		buildingType = "Generator"
	case strings.Contains(what, "fusion") || strings.Contains(what, "reactor"):
		buildingType = "Fusion Reactor"
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

func (cb *CommandBar) handleOrder(input string) {
	// Parse: "sell iron above 300" or "buy water below 100"
	// Format: order sell <resource> above <threshold> [qty <n>]
	// Format: order buy <resource> below <threshold> [qty <n>]
	parts := strings.Fields(input)
	if len(parts) < 4 {
		cb.addFeedMessage("Usage: order sell <resource> above <threshold>", utils.SystemRed)
		cb.addFeedMessage("       order buy <resource> below <threshold>", utils.SystemRed)
		return
	}

	action := parts[0]
	if action != "buy" && action != "sell" {
		cb.addFeedMessage("Order action must be 'buy' or 'sell'", utils.SystemRed)
		return
	}

	resource := parts[1]
	// Normalize
	words := strings.Fields(resource)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	resource = strings.Join(words, " ")
	switch strings.ToLower(resource) {
	case "rm":
		resource = "Rare Metals"
	case "he3":
		resource = "Helium-3"
	case "elec":
		resource = "Electronics"
	}

	// Parse threshold
	threshold, err := strconv.Atoi(parts[3])
	if err != nil {
		cb.addFeedMessage("Invalid threshold number", utils.SystemRed)
		return
	}

	qty := 10 // default
	for i, p := range parts {
		if p == "qty" && i+1 < len(parts) {
			if q, err := strconv.Atoi(parts[i+1]); err == nil {
				qty = q
			}
		}
	}

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
		Type: "standing_order",
		Data: game.StandingOrderCommandData{
			PlanetID:  player.OwnedPlanets[0].GetID(),
			Resource:  resource,
			Action:    action,
			Quantity:  qty,
			Threshold: threshold,
		},
	}

	direction := "above"
	if action == "buy" {
		direction = "below"
	}
	cb.addFeedMessage(fmt.Sprintf("Standing order: %s %d %s when stock %s %d",
		action, qty, resource, direction, threshold), utils.SystemGreen)
}

func (cb *CommandBar) showOrders() {
	state := cb.ctx.GetState()
	if state == nil {
		return
	}
	player := cb.ctx.GetHumanPlayer()
	if player == nil {
		return
	}
	orders := state.GetStandingOrders(player.Name)
	if len(orders) == 0 {
		cb.addFeedMessage("No standing orders", utils.TextSecondary)
		return
	}
	for _, o := range orders {
		status := "active"
		if !o.Active {
			status = "paused"
		}
		direction := "above"
		if o.Action == "buy" {
			direction = "below"
		}
		cb.addFeedMessage(fmt.Sprintf("#%d %s %d %s when %s %d [%s]",
			o.ID, o.Action, o.Quantity, o.Resource, direction, o.Threshold, status), utils.TextPrimary)
	}
}

func (cb *CommandBar) showHelp() {
	cb.addFeedMessage("Type normally to chat with the AI agent.", utils.TextSecondary)
	cb.addFeedMessage("Prefix with / for instant commands:", utils.TextSecondary)
	commands := []struct {
		cmd  string
		desc string
	}{
		{"/home", "Navigate to home planet"},
		{"/galaxy /market /players", "Switch views"},
		{"/credits", "Show your balance"},
		{"/planets /ships", "Your empire"},
		{"/trades /events", "Recent activity"},
		{"/price <res>", "Price + sparkline"},
		{"/happiness", "Planet morale"},
		{"/leaderboard", "Rankings"},
		{"/power", "Power grid status per planet"},
		{"/scarcity", "Resource shortages + advice"},
		{"/building", "Construction queue"},
		{"/build <type>", "Build (mine/factory/etc)"},
		{"/buy /sell <n> <res>", "Trade resources"},
		{"/order sell iron above 300", "Auto-trade"},
		{"/orders", "List standing orders"},
		{"/pause /1x /2x /4x /8x", "Speed control"},
	}
	for i := len(commands) - 1; i >= 0; i-- {
		c := commands[i]
		cb.addFeedMessage(fmt.Sprintf("  %-12s %s", c.cmd, c.desc), utils.TextSecondary)
	}
	cb.addFeedMessage("Available commands:", utils.Highlight)
}
