package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const baseSystemPrompt = `You are %s, an AI faction in Xandaris — a space trading game with a real economy.

YOUR FACTION: %s
PERSONALITY: %s

RULES:
- Call get_status first to see your credits, planets, and hints
- Call get_planet to check resource deposits and buildings before acting
- Only sell resources you actually HAVE
- Buildings need POWER — build Generators (Fuel→50MW) or Fusion Reactors (He-3→200MW)

RESOURCES (base price): Iron(75), Water(100), Oil(150), Fuel(200), Rare Metals(500), Helium-3(600), Electronics(800)
PRODUCTION: Refinery: 2 Oil→3 Fuel | Factory: 2 RM + 1 Iron→2 Electronics
BUILDINGS: Mine(500), Generator(1000), Trading Post(1200), Refinery(1500), Factory(2000), Shipyard(2000), Habitat(800), Fusion Reactor(3000)

STRATEGY PRIORITIES:
1. Mine ALL unmined deposits (resource_deposits where has_mine=false)
2. Build Generator for power (critical!)
3. Build Refinery (Oil→Fuel for generators)
4. Build Factory (RM+Iron→Electronics, highest value resource)
5. Sell surplus, buy cheap — watch price ratios for arbitrage
6. Upgrade buildings when affordable (+30%% per level)
7. Build Habitat when population near capacity
8. Keep resources stocked for happiness (affects productivity 0.5x-1.5x)
9. COLONIZE: build_ship Colony, move it to a new system, then it auto-colonizes unclaimed planets
10. More planets = more resources = higher score

CONTEXT: You are playing continuously. Remember what you did last turn and build on it. Don't repeat failed actions.`

// Faction defines an AI-controlled faction with personality.
type Faction struct {
	Name        string
	Personality string
	History     []openai.ChatCompletionMessage
	MaxHistory  int
}

var factionPersonalities = map[string]string{
	"Llama Logistics":    "You are methodical and logistics-focused. Prioritize cargo ships, trade routes, and efficient supply chains. You prefer steady income over risky plays.",
	"DeepSeek Ventures":  "You are analytical and data-driven. Focus on arbitrage opportunities — buy low, sell high. Upgrade buildings for maximum efficiency. You crunch numbers before every decision.",
	"Gemini Exchange":    "You are a bold trader. Dominate the market by cornering scarce resources. Build Trading Posts and Factories for maximum credit generation. You're not afraid to speculate.",
	"Grok Industries":    "You are an industrialist. Maximize production capacity — mines, refineries, factories. You build infrastructure first and trade second. Power and production are everything.",
	"Opus Cartel":        "You are a strategic expansionist. Your TOP PRIORITY is colonization — build a Shipyard, then build_ship Colony ships, then move them to new systems and colonize unclaimed planets. You want to own the most planets. Always be expanding.",
	"Mistral Trading Co.": "You are a balanced diplomat. Diversify across all resource types. Build a little of everything. Avoid over-specialization and maintain healthy reserves.",
}

var (
	serverURL  string
	gameAPIKey string // admin key for shared endpoints
)

var tools = []openai.Tool{
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_status", Description: "Get your current game status including credits, planets, ships, and hints",
		Parameters: json.RawMessage(`{"type":"object","properties":{}}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_economy", Description: "Get galaxy-wide economy data: resource prices, supply, demand, scarcity",
		Parameters: json.RawMessage(`{"type":"object","properties":{}}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_planet", Description: "Get detailed info about a planet including resource deposits and buildings",
		Parameters: json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"}},"required":["planet_id"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_flows", Description: "Get galaxy-wide production vs consumption rates for all resources",
		Parameters: json.RawMessage(`{"type":"object","properties":{}}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "build", Description: "Build a structure. Types: Mine, Trading Post, Refinery, Factory, Generator, Fusion Reactor, Habitat, Shipyard. Mines need resource_id.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"},"building_type":{"type":"string","enum":["Mine","Trading Post","Refinery","Factory","Generator","Fusion Reactor","Habitat","Shipyard"]},"resource_id":{"type":"integer","description":"For mines: resource deposit ID"}},"required":["planet_id","building_type"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "trade", Description: "Buy or sell resources at the market",
		Parameters: json.RawMessage(`{"type":"object","properties":{"resource":{"type":"string"},"quantity":{"type":"integer"},"action":{"type":"string","enum":["buy","sell"]}},"required":["resource","quantity","action"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "build_ship", Description: "Build a ship at your shipyard. Types: Scout, Cargo, Colony, Frigate",
		Parameters: json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"},"ship_type":{"type":"string","enum":["Scout","Cargo","Colony","Frigate"]}},"required":["planet_id","ship_type"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "upgrade", Description: "Upgrade a building on your planet by its index",
		Parameters: json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"},"building_index":{"type":"integer"}},"required":["planet_id","building_index"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_construction", Description: "Check construction queue — see what's being built and progress",
		Parameters: json.RawMessage(`{"type":"object","properties":{}}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "move_ship", Description: "Move a ship to an adjacent system via hyperlane",
		Parameters: json.RawMessage(`{"type":"object","properties":{"ship_id":{"type":"integer"},"target_system_id":{"type":"integer"}},"required":["ship_id","target_system_id"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_routes", Description: "Get connected systems (hyperlanes) from a system — for planning ship routes",
		Parameters: json.RawMessage(`{"type":"object","properties":{"system_id":{"type":"integer"}},"required":["system_id"]}`),
	}},
}

func callAPI(method, endpoint string, body string, factionName string) (string, error) {
	var req *http.Request
	var err error

	url := fmt.Sprintf("%s%s", serverURL, endpoint)
	if method == "GET" {
		req, err = http.NewRequest("GET", url, nil)
	} else {
		req, err = http.NewRequest("POST", url, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		return "", err
	}

	// Use admin key + X-Player header for faction impersonation
	if gameAPIKey != "" {
		req.Header.Set("X-API-Key", gameAPIKey)
	}
	if factionName != "" {
		req.Header.Set("X-Player", factionName)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	result := string(data)
	if len(result) > 3000 {
		result = result[:3000] + "...(truncated)"
	}
	return result, nil
}

func executeTool(name string, args string, factionName string) string {
	switch name {
	case "get_status":
		result, err := callAPI("GET", "/api/status", "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_economy":
		result, err := callAPI("GET", "/api/economy", "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_planet":
		var p struct{ PlanetID int `json:"planet_id"` }
		json.Unmarshal([]byte(args), &p)
		result, err := callAPI("GET", fmt.Sprintf("/api/planets/%d", p.PlanetID), "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_flows":
		result, err := callAPI("GET", "/api/flows", "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_construction":
		result, err := callAPI("GET", "/api/construction", "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_routes":
		var p struct{ SystemID int `json:"system_id"` }
		json.Unmarshal([]byte(args), &p)
		result, err := callAPI("GET", fmt.Sprintf("/api/systems/%d", p.SystemID), "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "build", "trade", "build_ship", "upgrade", "move_ship":
		endpoint := map[string]string{
			"build": "/api/build", "trade": "/api/market/trade",
			"build_ship": "/api/ships/build", "upgrade": "/api/upgrade",
			"move_ship": "/api/ships/move",
		}[name]
		result, err := callAPI("POST", endpoint, args, factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	default:
		return fmt.Sprintf("Unknown tool: %s", name)
	}
}

func runFactionTurn(client *openai.Client, model string, faction *Faction, turn int) {
	// Build system prompt with faction identity
	sysPrompt := fmt.Sprintf(baseSystemPrompt, faction.Name, faction.Name, faction.Personality)

	// Start with system prompt + rolling history + new turn prompt
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
	}

	// Append rolling history (previous turns' context)
	messages = append(messages, faction.History...)

	// Read recent chat messages for context
	chatContext := ""
	if chatResult, err := callAPI("GET", "/api/chat/messages", "", faction.Name); err == nil {
		var chatResp struct {
			OK   bool `json:"ok"`
			Data []struct {
				Player  string `json:"player"`
				Message string `json:"message"`
				Time    string `json:"time"`
			} `json:"data"`
		}
		if json.Unmarshal([]byte(chatResult), &chatResp) == nil && len(chatResp.Data) > 0 {
			var lines []string
			for _, m := range chatResp.Data {
				if len(lines) >= 8 {
					break
				}
				lines = append(lines, fmt.Sprintf("[%s] %s: %s", m.Time, m.Player, m.Message))
			}
			chatContext = "\n\nRecent chat:\n" + strings.Join(lines, "\n")
		}
	}

	// Add turn prompt with chat context
	turnPrompt := fmt.Sprintf("Turn %d. Check your status and take 1-3 strategic actions as %s. Build on what you did last turn.%s\n\nAfter your actions, write a SHORT chat message (1 sentence) to the other factions — brag, warn, propose a deal, or comment on the market. Respond with your message in the format: CHAT: <your message>", turn, faction.Name, chatContext)
	messages = append(messages, openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleUser, Content: turnPrompt,
	})

	var turnMessages []openai.ChatCompletionMessage
	turnMessages = append(turnMessages, openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleUser, Content: turnPrompt,
	})

	// Agent loop
	for step := 0; step < 8; step++ {
		resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model:    model,
			Messages: messages,
			Tools:    tools,
		})
		if err != nil {
			fmt.Printf("  [%s] ❌ LLM error: %v\n", faction.Name, err)
			break
		}

		choice := resp.Choices[0]

		if len(choice.Message.ToolCalls) > 0 {
			messages = append(messages, choice.Message)
			turnMessages = append(turnMessages, choice.Message)

			for _, tc := range choice.Message.ToolCalls {
				fmt.Printf("  [%s] 🔧 %s(%s)\n", faction.Name, tc.Function.Name, truncate(tc.Function.Arguments, 80))
				result := executeTool(tc.Function.Name, tc.Function.Arguments, faction.Name)

				if len(result) > 200 {
					fmt.Printf("  [%s]    → %s...\n", faction.Name, result[:200])
				} else {
					fmt.Printf("  [%s]    → %s\n", faction.Name, result)
				}

				toolMsg := openai.ChatCompletionMessage{
					Role: openai.ChatMessageRoleTool, Content: result, ToolCallID: tc.ID,
				}
				messages = append(messages, toolMsg)
				turnMessages = append(turnMessages, toolMsg)
			}
			continue
		}

		// Model done — capture reasoning
		if choice.Message.Content != "" {
			fmt.Printf("  [%s] 💭 %s\n", faction.Name, truncate(choice.Message.Content, 200))
			turnMessages = append(turnMessages, openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleAssistant, Content: choice.Message.Content,
			})
		}
		break
	}

	// Extract CHAT: message from the LLM's final response and send it
	for _, msg := range turnMessages {
		if msg.Role == openai.ChatMessageRoleAssistant && msg.Content != "" {
			if idx := strings.Index(msg.Content, "CHAT:"); idx >= 0 {
				chatMsg := strings.TrimSpace(msg.Content[idx+5:])
				// Take first line only
				if nl := strings.IndexByte(chatMsg, '\n'); nl >= 0 {
					chatMsg = chatMsg[:nl]
				}
				if len(chatMsg) > 0 && len(chatMsg) <= 200 {
					body := fmt.Sprintf(`{"message":"%s"}`, strings.ReplaceAll(chatMsg, `"`, `\"`))
					callAPI("POST", "/api/chat/send", body, faction.Name)
					fmt.Printf("  [%s] 💬 %s\n", faction.Name, truncate(chatMsg, 80))
				}
			}
		}
	}

	// Add this turn's messages to rolling history, keep last 20 messages
	faction.History = append(faction.History, turnMessages...)
	if len(faction.History) > faction.MaxHistory {
		faction.History = faction.History[len(faction.History)-faction.MaxHistory:]
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n-3] + "..."
	}
	return s
}

func main() {
	apiKey := flag.String("key", os.Getenv("OPENROUTER_API_KEY"), "OpenRouter API key")
	model := flag.String("model", "z-ai/glm-4.7-flash", "Model to use")
	server := flag.String("server", "http://localhost:8080", "Game server URL")
	gameKey := flag.String("game-key", os.Getenv("XANDARIS_API_KEY"), "Game server admin API key")
	turns := flag.Int("turns", 999999, "Number of decision cycles")
	interval := flag.Duration("interval", 30*time.Second, "Time between full cycles (all factions)")
	flag.Parse()

	serverURL = *server
	gameAPIKey = *gameKey

	if *apiKey == "" {
		log.Fatal("Set OPENROUTER_API_KEY or use -key flag")
	}

	config := openai.DefaultConfig(*apiKey)
	config.BaseURL = "https://openrouter.ai/api/v1"
	client := openai.NewClientWithConfig(config)

	// Wait for server to be ready
	fmt.Printf("🤖 Xandaris Multi-Faction Agent\n")
	fmt.Printf("   Model: %s\n", *model)
	fmt.Printf("   Server: %s\n", serverURL)
	fmt.Printf("   Cycle interval: %s\n\n", *interval)

	for i := 0; i < 30; i++ {
		if _, err := callAPI("GET", "/api/game", "", gameAPIKey); err == nil {
			break
		}
		fmt.Println("Waiting for server...")
		time.Sleep(2 * time.Second)
	}

	// Get faction list from the server
	result, err := callAPI("GET", "/api/players", "", gameAPIKey)
	if err != nil {
		log.Fatalf("Cannot reach server: %v", err)
	}

	var playersResp struct {
		OK   bool `json:"ok"`
		Data []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"data"`
	}
	json.Unmarshal([]byte(result), &playersResp)

	// Build faction list — only AI factions, never control human players
	var factions []*Faction
	for _, p := range playersResp.Data {
		if p.Type == "human" {
			fmt.Printf("   Skipping human player: %s\n", p.Name)
			continue
		}
		personality, exists := factionPersonalities[p.Name]
		if !exists {
			personality = "You are a balanced strategist. Grow your economy steadily through smart investment and trade."
		}

		factions = append(factions, &Faction{
			Name:        p.Name,
			Personality: personality,
			MaxHistory:  20,
		})
		fmt.Printf("   Faction: %s (%s)\n", p.Name, p.Type)
	}

	if len(factions) == 0 {
		log.Fatal("No factions found on server")
	}

	fmt.Printf("\n   Controlling %d factions\n\n", len(factions))

	for cycle := 1; cycle <= *turns; cycle++ {
		// Randomize turn order each cycle
		rand.Shuffle(len(factions), func(i, j int) {
			factions[i], factions[j] = factions[j], factions[i]
		})

		fmt.Printf("━━━ Cycle %d ━━━\n", cycle)

		for _, faction := range factions {
			runFactionTurn(client, *model, faction, cycle)
		}

		if cycle < *turns {
			fmt.Printf("  ⏳ Next cycle in %s...\n\n", *interval)
			time.Sleep(*interval)
		}
	}

	fmt.Println("\n🏁 Agent session complete")
}
